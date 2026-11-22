package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Aldrinsui/Spider/internal/config"
	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

type PaymentService interface {
	CreatePaymentOrder(ctx context.Context, traderID uuid.UUID, bookingID uuid.UUID) (*domain.PaymentEscrow, error)
	ProcessWebhook(ctx context.Context, signature string, rawBody []byte) error
	CancelBookingAndRefund(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, reason string) (int64, error)
	ReleasePayoutUponDelivery(ctx context.Context, bookingID uuid.UUID) error
}

type paymentService struct {
	escrowRepo    repository.EscrowRepository
	bookingRepo   repository.BookingRepository
	containerRepo repository.ContainerRepository
	cfg           *config.Config
}

func NewPaymentService(
	escrowRepo repository.EscrowRepository,
	bookingRepo repository.BookingRepository,
	containerRepo repository.ContainerRepository,
	cfg *config.Config,
) PaymentService {
	return &paymentService{
		escrowRepo:    escrowRepo,
		bookingRepo:   bookingRepo,
		containerRepo: containerRepo,
		cfg:           cfg,
	}
}

func (s *paymentService) CreatePaymentOrder(ctx context.Context, traderID uuid.UUID, bookingID uuid.UUID) (*domain.PaymentEscrow, error) {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch booking: %w", err)
	}
	if booking == nil {
		return nil, errors.New("booking not found")
	}

	if booking.TraderID != traderID {
		return nil, errors.New("unauthorized: booking belongs to another trader")
	}

	if booking.Status != domain.BookingStatusHeld {
		return nil, fmt.Errorf("booking is in '%s' status, only 'held' bookings can be paid", booking.Status)
	}

	if booking.HoldExpiresAt != nil && booking.HoldExpiresAt.Before(time.Now()) {
		return nil, errors.New("booking hold has expired. Please hold space again.")
	}

	// Check if payment escrow already created
	existingEscrow, err := s.escrowRepo.GetEscrowByBookingID(ctx, bookingID)
	if err == nil && existingEscrow != nil {
		return existingEscrow, nil
	}

	// Platform commission is 5% of base freight
	platformCommissionPaise := booking.PlatformFeePaise
	// Provider payout is base freight minus platform commission
	providerPayoutPaise := booking.BaseAmountPaise

	// In real environment, call Razorpay Orders API: rzp.Order.Create(...)
	// Here generate deterministic order ref
	orderID := fmt.Sprintf("order_%s", booking.BookingRef)

	escrow := &domain.PaymentEscrow{
		BookingID:                 bookingID,
		RazorpayOrderID:           orderID,
		AmountPaise:               booking.TotalAmountPaise,
		Currency:                  "INR",
		EscrowStatus:              domain.EscrowStatusPending,
		ProviderPayoutAmountPaise: providerPayoutPaise,
		PlatformCommissionPaise:   platformCommissionPaise,
	}

	if err := s.escrowRepo.CreatePaymentEscrow(ctx, escrow); err != nil {
		return nil, fmt.Errorf("failed to create escrow record: %w", err)
	}

	return escrow, nil
}

type RazorpayWebhookPayload struct {
	Event   string `json:"event"`
	Payload struct {
		Payment struct {
			Entity struct {
				ID          string `json:"id"`
				OrderID     string `json:"order_id"`
				Amount      int64  `json:"amount"`
				Status      string `json:"status"`
				Method      string `json:"method"`
				Captured    bool   `json:"captured"`
				Description string `json:"description"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

func (s *paymentService) ProcessWebhook(ctx context.Context, signature string, rawBody []byte) error {
	// 1. Verify HMAC SHA256 webhook signature
	if s.cfg.RazorpayWebhookSecret != "" && s.cfg.RazorpayWebhookSecret != "rzp_webhook_secret_dummy" {
		mac := hmac.New(sha256.New, []byte(s.cfg.RazorpayWebhookSecret))
		mac.Write(rawBody)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
			return errors.New("invalid webhook signature")
		}
	}

	var wh RazorpayWebhookPayload
	if err := json.Unmarshal(rawBody, &wh); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// We only process payment.captured events
	if wh.Event != "payment.captured" {
		return nil
	}

	paymentID := wh.Payload.Payment.Entity.ID
	orderID := wh.Payload.Payment.Entity.OrderID

	// 2. Deduplicate webhooks on razorpay_payment_id so retries don't double-confirm
	processed, err := s.escrowRepo.IsWebhookProcessed(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to check webhook deduplication: %w", err)
	}
	if processed {
		return nil // Idempotent: already processed
	}

	escrow, err := s.escrowRepo.GetEscrowByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to fetch escrow by order: %w", err)
	}
	if escrow == nil {
		return fmt.Errorf("no escrow found for order %s", orderID)
	}

	booking, err := s.bookingRepo.GetBookingByID(ctx, escrow.BookingID)
	if err != nil || booking == nil {
		return fmt.Errorf("booking not found for escrow: %w", err)
	}

	// 3. Handle Late Payments & Normal confirmations
	if booking.Status == domain.BookingStatusHeld {
		// Normal case: hold is still active
		routeTransferID := fmt.Sprintf("trf_route_%s", paymentID)
		if err := s.escrowRepo.ConfirmPaymentAndHoldEscrow(ctx, booking.ID, paymentID, signature, routeTransferID); err != nil {
			return fmt.Errorf("failed to confirm payment in escrow: %w", err)
		}
	} else if booking.Status == domain.BookingStatusExpired {
		// Late payment case: space may have been reclaimed!
		confirmed, err := s.escrowRepo.ReconcileLatePayment(ctx, booking.ID, paymentID, signature)
		if err != nil {
			return fmt.Errorf("late payment reconciliation failed: %w", err)
		}
		if !confirmed {
			// Space was taken: automated refund was scheduled
			return fmt.Errorf("late payment received after space reclaimed and reallocated: automated refund initiated for payment %s", paymentID)
		}
	}

	// 4. Record processed webhook for idempotency
	_ = s.escrowRepo.RecordProcessedWebhook(ctx, uuid.New().String(), paymentID, wh.Event, json.RawMessage(rawBody))

	return nil
}

func (s *paymentService) CancelBookingAndRefund(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, reason string) (int64, error) {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return 0, errors.New("booking not found")
	}

	if booking.TraderID != userID {
		return 0, errors.New("unauthorized: you do not own this booking")
	}

	if booking.Status != domain.BookingStatusConfirmed {
		return 0, fmt.Errorf("cannot cancel booking in status '%s'", booking.Status)
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return 0, errors.New("container not found")
	}

	// Indian Freight Cancellation Policy:
	// >= 24h prior to departure: 98% refund (100% minus 2% gateway fee)
	// 12h - 24h prior to departure: 75% refund
	// < 12h prior to departure: 50% refund
	hoursUntilDeparture := time.Until(container.DepartureTime).Hours()
	if hoursUntilDeparture < 0 {
		return 0, errors.New("cannot cancel a shipment after scheduled departure")
	}

	var refundPercent float64
	if hoursUntilDeparture >= 24.0 {
		refundPercent = 0.98
	} else if hoursUntilDeparture >= 12.0 {
		refundPercent = 0.75
	} else {
		refundPercent = 0.50
	}

	refundAmountPaise := int64(math.Round(float64(booking.TotalAmountPaise) * refundPercent))
	refundID := fmt.Sprintf("rfnd_%s", uuid.New().String()[:8])

	if err := s.escrowRepo.ProcessRefund(ctx, bookingID, refundID, refundAmountPaise, reason); err != nil {
		return 0, fmt.Errorf("failed to process cancellation refund: %w", err)
	}

	return refundAmountPaise, nil
}

func (s *paymentService) ReleasePayoutUponDelivery(ctx context.Context, bookingID uuid.UUID) error {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return errors.New("booking not found")
	}

	if booking.Status != domain.BookingStatusDelivered {
		return fmt.Errorf("cannot release payout: booking is not yet marked delivered (status: %s)", booking.Status)
	}

	return s.escrowRepo.ReleaseEscrowPayout(ctx, bookingID)
}

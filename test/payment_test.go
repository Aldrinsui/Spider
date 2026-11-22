package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Aldrinsui/Spider/internal/config"
	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/google/uuid"
)

func TestPaymentService_WebhookDeduplicationAndEscrow(t *testing.T) {
	containerRepo := &MockContainerRepository{containers: make(map[uuid.UUID]*domain.Container)}
	outboxRepo := NewMockOutboxRepository()
	bookingRepo := NewMockBookingRepository(containerRepo, outboxRepo).(*MockBookingRepository)
	escrowRepo := NewMockEscrowRepository(bookingRepo, containerRepo, outboxRepo)
	cfg := &config.Config{RazorpayWebhookSecret: "rzp_webhook_secret_dummy"}
	paymentSvc := service.NewPaymentService(escrowRepo, bookingRepo, containerRepo, cfg)
	ctx := context.Background()

	// 1. Setup container and booking in held status
	containerID := uuid.New()
	containerRepo.containers[containerID] = &domain.Container{
		ID:                containerID,
		DepartureTime:     time.Now().Add(48 * time.Hour),
		AvailableCBM:      20.0,
		AvailableWeightKG: 15000.0,
	}

	traderID := uuid.New()
	bookingID := uuid.New()
	bookingRef := "SP-202611-TEST01"
	booking := &domain.Booking{
		ID:                  bookingID,
		BookingRef:          bookingRef,
		ContainerID:         containerID,
		TraderID:            traderID,
		Status:              domain.BookingStatusHeld,
		ChargeableVolumeCBM: 1.5,
		ActualWeightKG:      500,
		BaseAmountPaise:     100000,
		PlatformFeePaise:    5000,
		GSTAmountPaise:      5900,
		TotalAmountPaise:    110900,
	}
	bookingRepo.bookings[bookingID] = booking

	// 2. Create payment order
	escrow, err := paymentSvc.CreatePaymentOrder(ctx, traderID, bookingID)
	if err != nil {
		t.Fatalf("expected nil error on CreatePaymentOrder, got: %v", err)
	}
	if escrow.EscrowStatus != domain.EscrowStatusPending {
		t.Fatalf("expected pending escrow status, got: %s", escrow.EscrowStatus)
	}

	// 3. Simulate Razorpay webhook payment.captured
	paymentID := "pay_test_spider_987654"
	orderID := escrow.RazorpayOrderID
	webhookJSON := fmt.Sprintf(`{
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {
					"id": "%s",
					"order_id": "%s",
					"amount": 110900,
					"status": "captured",
					"method": "upi",
					"captured": true,
					"description": "Freight payment"
				}
			}
		}
	}`, paymentID, orderID)

	err = paymentSvc.ProcessWebhook(ctx, "mock_signature", []byte(webhookJSON))
	if err != nil {
		t.Fatalf("expected success processing webhook, got: %v", err)
	}

	// Verify booking was confirmed
	b := bookingRepo.bookings[bookingID]
	if b.Status != domain.BookingStatusConfirmed {
		t.Fatalf("expected booking status 'confirmed', got: %s", b.Status)
	}

	// Verify escrow status is 'held' (escrow holds funds pending POD delivery)
	e, _ := escrowRepo.GetEscrowByBookingID(ctx, bookingID)
	if e.EscrowStatus != domain.EscrowStatusHeld {
		t.Fatalf("expected escrow status 'held', got: %s", e.EscrowStatus)
	}
	if e.RazorpayPaymentID != paymentID {
		t.Fatalf("expected payment ID %s, got %s", paymentID, e.RazorpayPaymentID)
	}

	// 4. Webhook deduplication test: resending identical webhook does NOT reprocess
	err = paymentSvc.ProcessWebhook(ctx, "mock_signature", []byte(webhookJSON))
	if err != nil {
		t.Fatalf("expected duplicate webhook to succeed idempotently, got: %v", err)
	}

	// 5. Escrow release upon delivery
	// Attempt before delivery -> fails
	err = paymentSvc.ReleasePayoutUponDelivery(ctx, bookingID)
	if err == nil {
		t.Fatalf("expected failure releasing payout before delivery, got nil")
	}

	// Mark delivered and release -> succeeds
	b.Status = domain.BookingStatusDelivered
	err = paymentSvc.ReleasePayoutUponDelivery(ctx, bookingID)
	if err != nil {
		t.Fatalf("expected success releasing payout after delivery, got: %v", err)
	}
	e, _ = escrowRepo.GetEscrowByBookingID(ctx, bookingID)
	if e.EscrowStatus != domain.EscrowStatusReleasedToProvider {
		t.Fatalf("expected escrow status 'released_to_provider', got: %s", e.EscrowStatus)
	}
}

func TestPaymentService_LatePaymentReconciliation(t *testing.T) {
	containerRepo := &MockContainerRepository{containers: make(map[uuid.UUID]*domain.Container)}
	outboxRepo := NewMockOutboxRepository()
	bookingRepo := NewMockBookingRepository(containerRepo, outboxRepo).(*MockBookingRepository)
	escrowRepo := NewMockEscrowRepository(bookingRepo, containerRepo, outboxRepo)
	cfg := &config.Config{RazorpayWebhookSecret: "rzp_webhook_secret_dummy"}
	paymentSvc := service.NewPaymentService(escrowRepo, bookingRepo, containerRepo, cfg)
	ctx := context.Background()

	containerID := uuid.New()
	container := &domain.Container{
		ID:                containerID,
		DepartureTime:     time.Now().Add(48 * time.Hour),
		AvailableCBM:      5.0,
		AvailableWeightKG: 5000.0,
	}
	containerRepo.containers[containerID] = container

	bookingID := uuid.New()
	orderID := "order_SP-LATE-01"
	// Booking has EXPIRED before webhook arrived
	bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:                  bookingID,
		BookingRef:          "SP-LATE-01",
		ContainerID:         containerID,
		Status:              domain.BookingStatusExpired,
		ChargeableVolumeCBM: 2.0,
		ActualWeightKG:      1000,
		TotalAmountPaise:    150000,
	}
	_ = escrowRepo.CreatePaymentEscrow(ctx, &domain.PaymentEscrow{
		BookingID:       bookingID,
		RazorpayOrderID: orderID,
		AmountPaise:     150000,
		EscrowStatus:    domain.EscrowStatusPending,
	})

	// Case A: Space is still available (container has 5 CBM, booking needs 2 CBM)
	// Webhook arrives late -> confirms booking and deducts space
	paymentIDA := "pay_late_avail_123"
	webhookJSONA := fmt.Sprintf(`{"event":"payment.captured","payload":{"payment":{"entity":{"id":"%s","order_id":"%s","amount":150000,"status":"captured","captured":true}}}}`, paymentIDA, orderID)

	err := paymentSvc.ProcessWebhook(ctx, "mock_sig", []byte(webhookJSONA))
	if err != nil {
		t.Fatalf("expected late payment with available space to succeed, got: %v", err)
	}
	if bookingRepo.bookings[bookingID].Status != domain.BookingStatusConfirmed {
		t.Fatalf("expected late booking to be confirmed since space was free, got: %s", bookingRepo.bookings[bookingID].Status)
	}
	if container.AvailableCBM != 3.0 { // 5.0 - 2.0 = 3.0
		t.Fatalf("expected available space decremented to 3.0 CBM, got: %f", container.AvailableCBM)
	}

	// Case B: Container space was taken by other traders (AvailableCBM = 0)
	bookingIDB := uuid.New()
	orderIDB := "order_SP-LATE-02"
	container.AvailableCBM = 0.5 // only 0.5 CBM left, booking needs 2.0 CBM
	bookingRepo.bookings[bookingIDB] = &domain.Booking{
		ID:                  bookingIDB,
		BookingRef:          "SP-LATE-02",
		ContainerID:         containerID,
		Status:              domain.BookingStatusExpired,
		ChargeableVolumeCBM: 2.0,
		ActualWeightKG:      1000,
		TotalAmountPaise:    150000,
	}
	_ = escrowRepo.CreatePaymentEscrow(ctx, &domain.PaymentEscrow{
		BookingID:       bookingIDB,
		RazorpayOrderID: orderIDB,
		AmountPaise:     150000,
		EscrowStatus:    domain.EscrowStatusPending,
	})

	paymentIDB := "pay_late_taken_456"
	webhookJSONB := fmt.Sprintf(`{"event":"payment.captured","payload":{"payment":{"entity":{"id":"%s","order_id":"%s","amount":150000,"status":"captured","captured":true}}}}`, paymentIDB, orderIDB)

	// Late webhook arrives when space is gone -> auto-refunds and reports status
	err = paymentSvc.ProcessWebhook(ctx, "mock_sig", []byte(webhookJSONB))
	if err == nil {
		t.Fatalf("expected auto-refund notification error for late payment without space, got nil")
	}

	// Verify booking was refunded
	if bookingRepo.bookings[bookingIDB].Status != domain.BookingStatusRefunded {
		t.Fatalf("expected booking status 'refunded', got: %s", bookingRepo.bookings[bookingIDB].Status)
	}
	escrowB, _ := escrowRepo.GetEscrowByBookingID(ctx, bookingIDB)
	if escrowB.EscrowStatus != domain.EscrowStatusRefundedToTrader {
		t.Fatalf("expected escrow status 'refunded_to_trader', got: %s", escrowB.EscrowStatus)
	}
}

func TestPaymentService_CancellationPolicyAsCode(t *testing.T) {
	containerRepo := &MockContainerRepository{containers: make(map[uuid.UUID]*domain.Container)}
	outboxRepo := NewMockOutboxRepository()
	bookingRepo := NewMockBookingRepository(containerRepo, outboxRepo).(*MockBookingRepository)
	escrowRepo := NewMockEscrowRepository(bookingRepo, containerRepo, outboxRepo)
	cfg := &config.Config{RazorpayWebhookSecret: "rzp_webhook_secret_dummy"}
	paymentSvc := service.NewPaymentService(escrowRepo, bookingRepo, containerRepo, cfg)
	ctx := context.Background()

	traderID := uuid.New()
	containerID := uuid.New()
	containerRepo.containers[containerID] = &domain.Container{
		ID:                containerID,
		DepartureTime:     time.Now().Add(30 * time.Hour), // > 24 hours away
		AvailableCBM:      10.0,
		AvailableWeightKG: 10000.0,
	}

	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:                  bookingID,
		ContainerID:         containerID,
		TraderID:            traderID,
		Status:              domain.BookingStatusConfirmed,
		ChargeableVolumeCBM: 2.0,
		ActualWeightKG:      500,
		TotalAmountPaise:    100000, // Rs 1000
	}
	bookingRepo.bookings[bookingID] = booking
	_ = escrowRepo.CreatePaymentEscrow(ctx, &domain.PaymentEscrow{
		BookingID:    bookingID,
		AmountPaise:  100000,
		EscrowStatus: domain.EscrowStatusHeld,
	})

	// Cancel > 24 hours prior -> 98% refund (100000 * 0.98 = 98000 paise / Rs 980)
	refundPaise, err := paymentSvc.CancelBookingAndRefund(ctx, traderID, bookingID, "Production plan delayed")
	if err != nil {
		t.Fatalf("expected success cancelling booking, got: %v", err)
	}
	if refundPaise != 98000 {
		t.Fatalf("expected 98%% refund (98000 paise), got: %d", refundPaise)
	}
	if booking.Status != domain.BookingStatusCancelled {
		t.Fatalf("expected booking status 'cancelled', got: %s", booking.Status)
	}

	// Container space restored (10.0 + 2.0 = 12.0)
	if containerRepo.containers[containerID].AvailableCBM != 12.0 {
		t.Fatalf("expected container space restored to 12.0 CBM, got: %f", containerRepo.containers[containerID].AvailableCBM)
	}
}

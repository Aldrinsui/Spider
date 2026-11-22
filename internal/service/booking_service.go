package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

var prohibitedKeywords = []string{
	"explosive", "dynamite", "gunpowder", "radioactive", "uranium", "toxic gas", "contraband", "narcotics",
}

type HoldSpaceRequest struct {
	ContainerID         uuid.UUID `json:"container_id"`
	ConsignmentLengthCM float64   `json:"consignment_length_cm"`
	ConsignmentWidthCM  float64   `json:"consignment_width_cm"`
	ConsignmentHeightCM float64   `json:"consignment_height_cm"`
	ActualWeightKG      float64   `json:"actual_weight_kg"`
	IsDangerousGoods    bool      `json:"is_dangerous_goods"`
	CargoDescription    string    `json:"cargo_description"`
	EWayBillRef         string    `json:"eway_bill_ref,omitempty"`
}

type BookingService interface {
	HoldSpace(ctx context.Context, traderID uuid.UUID, req HoldSpaceRequest) (*domain.Booking, error)
	GetBooking(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	GetBookingByRef(ctx context.Context, ref string) (*domain.Booking, error)
	ListTraderBookings(ctx context.Context, traderID uuid.UUID) ([]domain.Booking, error)
	ReclaimExpiredHolds(ctx context.Context) (int, error)
}

type bookingService struct {
	bookingRepo   repository.BookingRepository
	containerSvc  ContainerService
	outboxRepo    repository.OutboxRepository
}

func NewBookingService(bookingRepo repository.BookingRepository, containerSvc ContainerService, outboxRepo repository.OutboxRepository) BookingService {
	return &bookingService{
		bookingRepo:  bookingRepo,
		containerSvc: containerSvc,
		outboxRepo:   outboxRepo,
	}
}

func (s *bookingService) HoldSpace(ctx context.Context, traderID uuid.UUID, req HoldSpaceRequest) (*domain.Booking, error) {
	// 1. Validate cargo description & check prohibited goods list
	desc := strings.ToLower(strings.TrimSpace(req.CargoDescription))
	if desc == "" {
		return nil, errors.New("cargo description is required")
	}
	for _, prohibited := range prohibitedKeywords {
		if strings.Contains(desc, prohibited) {
			return nil, fmt.Errorf("prohibited cargo detected: '%s' cannot be transported on this platform", prohibited)
		}
	}

	// 2. Calculate quote, volumetric check, and pricing
	quote, err := s.containerSvc.CalculateQuote(ctx, QuoteRequest{
		ContainerID:      req.ContainerID,
		LengthCM:         req.ConsignmentLengthCM,
		WidthCM:          req.ConsignmentWidthCM,
		HeightCM:         req.ConsignmentHeightCM,
		ActualWeightKG:   req.ActualWeightKG,
		IsDangerousGoods: req.IsDangerousGoods,
	})
	if err != nil {
		return nil, fmt.Errorf("quote calculation failed: %w", err)
	}

	bookingID := uuid.New()

	// 3. Prepare Booking record
	booking := &domain.Booking{
		ID:                  bookingID,
		ContainerID:         req.ContainerID,
		TraderID:            traderID,
		ConsignmentLengthCM: req.ConsignmentLengthCM,
		ConsignmentWidthCM:  req.ConsignmentWidthCM,
		ConsignmentHeightCM: req.ConsignmentHeightCM,
		ActualWeightKG:      req.ActualWeightKG,
		ComputedVolumeCBM:   quote.ComputedVolumeCBM,
		ChargeableVolumeCBM: quote.ChargeableVolumeCBM,
		ChargeableWeightKG:  quote.ChargeableWeightKG,
		IsDangerousGoods:    req.IsDangerousGoods,
		CargoDescription:    req.CargoDescription,
		EWayBillRef:         req.EWayBillRef,
		BaseAmountPaise:     quote.BaseAmountPaise,
		PlatformFeePaise:    quote.PlatformFeePaise,
		GSTAmountPaise:      quote.GSTAmountPaise,
		TotalAmountPaise:    quote.TotalAmountPaise,
	}

	// 4. Prepare Transactional Outbox event (atomic with booking hold)
	payload, _ := json.Marshal(map[string]interface{}{
		"booking_id":            bookingID,
		"container_id":          req.ContainerID,
		"trader_id":             traderID,
		"chargeable_volume_cbm": quote.ChargeableVolumeCBM,
		"actual_weight_kg":      req.ActualWeightKG,
		"total_amount_paise":    quote.TotalAmountPaise,
		"hold_duration_minutes": 10,
	})

	outboxEvent := &domain.OutboxEvent{
		EventType:     "booking.held",
		AggregateType: "booking",
		AggregateID:   bookingID,
		Payload:       payload,
	}

	// 5. Execute atomic transaction (container lock + space decrement + booking insert + outbox insert)
	if err := s.bookingRepo.HoldSpaceTransaction(ctx, booking, outboxEvent); err != nil {
		return nil, fmt.Errorf("failed to hold container space: %w", err)
	}

	return booking, nil
}

func (s *bookingService) GetBooking(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return s.bookingRepo.GetBookingByID(ctx, id)
}

func (s *bookingService) GetBookingByRef(ctx context.Context, ref string) (*domain.Booking, error) {
	return s.bookingRepo.GetBookingByRef(ctx, ref)
}

func (s *bookingService) ListTraderBookings(ctx context.Context, traderID uuid.UUID) ([]domain.Booking, error) {
	return s.bookingRepo.ListBookingsByTrader(ctx, traderID)
}

func (s *bookingService) ReclaimExpiredHolds(ctx context.Context) (int, error) {
	return s.bookingRepo.ReclaimExpiredHolds(ctx)
}

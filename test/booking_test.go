package test

import (
	"context"
	"testing"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/google/uuid"
)

func TestBookingService_AtomicHoldAndReclaimer(t *testing.T) {
	containerRepo := &MockContainerRepository{containers: make(map[uuid.UUID]*domain.Container)}
	outboxRepo := NewMockOutboxRepository()
	bookingRepo := NewMockBookingRepository(containerRepo, outboxRepo)
	kycRepo := NewMockKYCRepository()
	containerSvc := service.NewContainerService(containerRepo, kycRepo)
	bookingSvc := service.NewBookingService(bookingRepo, containerSvc, outboxRepo)
	ctx := context.Background()

	// Create test container
	containerID := uuid.New()
	initialCBM := 18.6
	initialWeight := 14000.0
	containerRepo.containers[containerID] = &domain.Container{
		ID:                containerID,
		ISOContainerCode:  "MSKU 482913",
		TransportMode:     domain.TransportModeRail,
		OriginTitle:       "Chennai Central",
		DestTitle:         "Bengaluru Inland Depot",
		TotalCBM:          30.0,
		AvailableCBM:      initialCBM,
		TotalWeightKG:     20000.0,
		AvailableWeightKG: initialWeight,
		RatePerCBMPaise:   74000,
		MinChargeableCBM:  0.1,
		Status:            domain.ContainerStatusScheduled,
	}

	traderID := uuid.New()

	// 1. Prohibited goods test
	_, err := bookingSvc.HoldSpace(ctx, traderID, service.HoldSpaceRequest{
		ContainerID:         containerID,
		ConsignmentLengthCM: 120,
		ConsignmentWidthCM:  80,
		ConsignmentHeightCM: 90,
		ActualWeightKG:      640,
		CargoDescription:    "High grade military explosive parts",
	})
	if err == nil {
		t.Fatalf("expected error for prohibited cargo, got nil")
	}

	// 2. Successful atomic space hold (0.86 CBM, 640 kg)
	holdReq := service.HoldSpaceRequest{
		ContainerID:         containerID,
		ConsignmentLengthCM: 120,
		ConsignmentWidthCM:  80,
		ConsignmentHeightCM: 90,
		ActualWeightKG:      640,
		IsDangerousGoods:    false,
		CargoDescription:    "Automotive gearboxes and spare parts",
		EWayBillRef:         "241098765432",
	}

	booking, err := bookingSvc.HoldSpace(ctx, traderID, holdReq)
	if err != nil {
		t.Fatalf("expected success holding space, got: %v", err)
	}

	if booking.Status != domain.BookingStatusHeld {
		t.Fatalf("expected booking status 'held', got: %s", booking.Status)
	}
	if booking.HoldExpiresAt == nil {
		t.Fatalf("expected hold_expires_at to be set")
	}

	// Verify container inventory was decremented atomically
	c := containerRepo.containers[containerID]
	expectedCBM := initialCBM - 0.86
	if c.AvailableCBM != expectedCBM {
		t.Fatalf("expected available CBM %.2f, got %.2f", expectedCBM, c.AvailableCBM)
	}
	expectedWeight := initialWeight - 640.0
	if c.AvailableWeightKG != expectedWeight {
		t.Fatalf("expected available weight %.2f, got %.2f", expectedWeight, c.AvailableWeightKG)
	}

	// Verify transactional outbox event was recorded
	if len(outboxRepo.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[0].EventType != "booking.held" {
		t.Fatalf("expected outbox event 'booking.held', got '%s'", outboxRepo.events[0].EventType)
	}

	// 3. Exceeded capacity test: trying to book 20 CBM when only ~17.74 CBM is available
	_, err = bookingSvc.HoldSpace(ctx, traderID, service.HoldSpaceRequest{
		ContainerID:         containerID,
		ConsignmentLengthCM: 400,
		ConsignmentWidthCM:  250,
		ConsignmentHeightCM: 200, // 20 CBM
		ActualWeightKG:      5000,
		CargoDescription:    "Heavy textiles",
	})
	if err == nil {
		t.Fatalf("expected error when requesting more than available capacity, got nil")
	}

	// Verify container inventory was NOT changed by the failed request
	if c.AvailableCBM != expectedCBM {
		t.Fatalf("container inventory should not change after failed hold: expected %.2f, got %.2f", expectedCBM, c.AvailableCBM)
	}

	// 4. Expiry reclaimer test: simulate hold expiry
	past := time.Now().Add(-1 * time.Minute)
	booking.HoldExpiresAt = &past

	reclaimed, err := bookingSvc.ReclaimExpiredHolds(ctx)
	if err != nil {
		t.Fatalf("expected nil error on ReclaimExpiredHolds, got: %v", err)
	}
	if reclaimed != 1 {
		t.Fatalf("expected 1 reclaimed hold, got %d", reclaimed)
	}

	// Verify booking status changed to expired
	b, _ := bookingSvc.GetBooking(ctx, booking.ID)
	if b.Status != domain.BookingStatusExpired {
		t.Fatalf("expected booking status 'expired', got '%s'", b.Status)
	}

	// Verify container capacity was restored back to initial values!
	if c.AvailableCBM != initialCBM {
		t.Fatalf("expected container capacity restored to %.2f, got %.2f", initialCBM, c.AvailableCBM)
	}
	if c.AvailableWeightKG != initialWeight {
		t.Fatalf("expected container weight restored to %.2f, got %.2f", initialWeight, c.AvailableWeightKG)
	}
}

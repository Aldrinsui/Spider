package test

import (
	"context"
	"testing"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/google/uuid"
)

func TestContainerService_CreationAndQuoteCalculation(t *testing.T) {
	containerRepo := NewMockContainerRepository()
	kycRepo := NewMockKYCRepository()
	containerSvc := service.NewContainerService(containerRepo, kycRepo)
	ctx := context.Background()

	providerID := uuid.New()

	// Approve provider in KYC repo
	_ = kycRepo.CreateOrUpdateKYC(ctx, &domain.ProviderKYC{
		UserID: providerID,
		Status: domain.KYCStatusApproved,
	})

	// 1. Create container with Rail mode, MSKU 482913 (matching mockups!)
	now := time.Now()
	container := domain.Container{
		ISOContainerCode:     "MSKU 482913",
		TransportMode:        domain.TransportModeRail,
		OriginTitle:          "Chennai Central",
		OriginLocation:       domain.LocationPoint{Latitude: 13.0827, Longitude: 80.2707},
		DestTitle:            "Bengaluru Inland Depot",
		DestLocation:         domain.LocationPoint{Latitude: 12.9716, Longitude: 77.5946},
		DepartureTime:        now.Add(24 * time.Hour),
		EstimatedArrivalTime: now.Add(36 * time.Hour),
		TotalCBM:             30.0,
		AvailableCBM:         18.6, // 18.6 of 30 CBM free as in Mockup Screen 1 & 2
		TotalWeightKG:        20000.0,
		AvailableWeightKG:    14000.0,
		RatePerCBMPaise:      74000, // Rs 740 / CBM
		MinChargeableCBM:     0.1,
	}

	created, err := containerSvc.CreateContainer(ctx, providerID, container)
	if err != nil {
		t.Fatalf("expected nil error creating container, got: %v", err)
	}
	if created.ISOContainerCode != "MSKU 482913" {
		t.Fatalf("expected ISO code MSKU 482913, got %s", created.ISOContainerCode)
	}

	// 2. Consignment Quote calculation matching HTML Screen 2:
	// Length: 120 cm, Width: 80 cm, Height: 90 cm, Weight: 640 kg
	// Volume = (120 * 80 * 90) / 1,000,000 = 0.864 -> 0.86 CBM
	// Rate = Rs 740 / CBM
	// Base = 0.86 * 740 = Rs 636.40 (63640 paise)
	quoteReq := service.QuoteRequest{
		ContainerID:    created.ID,
		LengthCM:       120,
		WidthCM:        80,
		HeightCM:       90,
		ActualWeightKG: 640,
	}

	quote, err := containerSvc.CalculateQuote(ctx, quoteReq)
	if err != nil {
		t.Fatalf("expected nil error calculating quote, got: %v", err)
	}

	if quote.ComputedVolumeCBM != 0.86 {
		t.Fatalf("expected computed volume 0.86 CBM, got %f", quote.ComputedVolumeCBM)
	}
	if quote.ChargeableVolumeCBM != 0.86 {
		t.Fatalf("expected chargeable volume 0.86 CBM, got %f", quote.ChargeableVolumeCBM)
	}
	if quote.BaseAmountPaise != 63640 {
		t.Fatalf("expected base amount 63640 paise (Rs 636.40), got %d paise", quote.BaseAmountPaise)
	}
	if quote.PlatformFeePaise <= 0 {
		t.Fatalf("expected platform fee > 0, got %d", quote.PlatformFeePaise)
	}
	if quote.GSTAmountPaise <= 0 {
		t.Fatalf("expected GST amount > 0, got %d", quote.GSTAmountPaise)
	}
	if quote.TotalAmountPaise <= quote.BaseAmountPaise {
		t.Fatalf("expected total amount to include base + fee + GST, got %d", quote.TotalAmountPaise)
	}
}

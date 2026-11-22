package test

import (
	"context"
	"testing"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/google/uuid"
)

func TestTrackingService_LiveLocationPODAndDisputeFlow(t *testing.T) {
	containerRepo := &MockContainerRepository{containers: make(map[uuid.UUID]*domain.Container)}
	outboxRepo := NewMockOutboxRepository()
	bookingRepo := NewMockBookingRepository(containerRepo, outboxRepo).(*MockBookingRepository)
	escrowRepo := NewMockEscrowRepository(bookingRepo, containerRepo, outboxRepo).(*MockEscrowRepository)
	docRepo := NewMockDocumentRepository()
	trackingRepo := NewMockTrackingRepository(bookingRepo, escrowRepo, containerRepo)
	disputeRepo := NewMockDisputeRepository(bookingRepo, escrowRepo)

	trackingSvc := service.NewTrackingService(trackingRepo, docRepo, disputeRepo, bookingRepo, containerRepo)
	ctx := context.Background()

	providerID := uuid.New()
	traderID := uuid.New()
	containerID := uuid.New()

	containerRepo.containers[containerID] = &domain.Container{
		ID:         containerID,
		ProviderID: providerID,
	}

	bookingID := uuid.New()
	bookingRepo.bookings[bookingID] = &domain.Booking{
		ID:          bookingID,
		ContainerID: containerID,
		TraderID:    traderID,
		Status:      domain.BookingStatusConfirmed,
	}
	escrowRepo.escrows[bookingID] = &domain.PaymentEscrow{
		BookingID:    bookingID,
		EscrowStatus: domain.EscrowStatusHeld,
	}

	// 1. Live GPS update by provider
	lat := 13.0827
	lng := 80.2707
	err := trackingSvc.UpdateLocation(ctx, providerID, containerID, lat, lng)
	if err != nil {
		t.Fatalf("expected nil error updating GPS location, got: %v", err)
	}
	if containerRepo.containers[containerID].CurrentLocation == nil {
		t.Fatalf("expected container current location to be updated")
	}

	// 2. Document vault: upload Bill of Lading
	doc, err := trackingSvc.UploadDocument(ctx, traderID, bookingID, "bill_of_lading", "https://cdn.spider.io/bol_123.pdf", "bol_123.pdf", 204800, "application/pdf")
	if err != nil {
		t.Fatalf("expected success uploading document, got: %v", err)
	}
	docs, err := trackingSvc.ListDocuments(ctx, traderID, bookingID)
	if err != nil || len(docs) != 1 {
		t.Fatalf("expected 1 document in vault, got %d", len(docs))
	}
	if docs[0].DocType != "bill_of_lading" {
		t.Fatalf("expected doc_type bill_of_lading, got %s", doc.DocType)
	}

	// 3. Proof of Delivery (POD) flow:
	// Trader cannot submit POD -> only provider
	_, err = trackingSvc.SubmitProofOfDelivery(ctx, traderID, bookingID, "https://cdn.spider.io/pod.jpg", "Rajesh Kumar", "+919876543210", "https://cdn.spider.io/sig.png", &lat, &lng)
	if err == nil {
		t.Fatalf("expected error when non-provider submits POD, got nil")
	}

	// Provider submits POD photo & recipient info
	pod, err := trackingSvc.SubmitProofOfDelivery(ctx, providerID, bookingID, "https://cdn.spider.io/pod.jpg", "Rajesh Kumar", "+919876543210", "https://cdn.spider.io/sig.png", &lat, &lng)
	if err != nil {
		t.Fatalf("expected success submitting POD, got: %v", err)
	}
	if pod.RecipientName != "Rajesh Kumar" {
		t.Fatalf("expected recipient name Rajesh Kumar, got %s", pod.RecipientName)
	}

	// Verify booking transitioned to delivered
	if bookingRepo.bookings[bookingID].Status != domain.BookingStatusDelivered {
		t.Fatalf("expected booking status 'delivered', got: %s", bookingRepo.bookings[bookingID].Status)
	}

	// Verify escrow delayed payout was released to provider!
	if escrowRepo.escrows[bookingID].EscrowStatus != domain.EscrowStatusReleasedToProvider {
		t.Fatalf("expected escrow status 'released_to_provider', got: %s", escrowRepo.escrows[bookingID].EscrowStatus)
	}

	// 4. Dispute handling test on a new shipment
	bookingID2 := uuid.New()
	bookingRepo.bookings[bookingID2] = &domain.Booking{
		ID:          bookingID2,
		ContainerID: containerID,
		TraderID:    traderID,
		Status:      domain.BookingStatusConfirmed,
	}
	escrowRepo.escrows[bookingID2] = &domain.PaymentEscrow{
		BookingID:    bookingID2,
		EscrowStatus: domain.EscrowStatusHeld,
	}

	// Trader raises dispute for damaged goods
	dispute, err := trackingSvc.RaiseDispute(ctx, traderID, bookingID2, "Cargo Damaged", "Boxes arrived crushed and water damaged", []string{"https://cdn.spider.io/dmg1.jpg"})
	if err != nil {
		t.Fatalf("expected success raising dispute, got: %v", err)
	}
	if dispute.Status != "open" {
		t.Fatalf("expected dispute status 'open', got: %s", dispute.Status)
	}

	// Verify escrow is frozen in 'disputed_freeze'
	if escrowRepo.escrows[bookingID2].EscrowStatus != domain.EscrowStatusDisputedFreeze {
		t.Fatalf("expected escrow status 'disputed_freeze', got: %s", escrowRepo.escrows[bookingID2].EscrowStatus)
	}

	// Admin resolves dispute with refund to trader
	adminID := uuid.New()
	err = trackingSvc.ResolveDispute(ctx, adminID, dispute.ID, "resolved_refund", "Evidence clearly shows water ingress damage.")
	if err != nil {
		t.Fatalf("expected success resolving dispute, got: %v", err)
	}

	if escrowRepo.escrows[bookingID2].EscrowStatus != domain.EscrowStatusRefundedToTrader {
		t.Fatalf("expected escrow status 'refunded_to_trader', got: %s", escrowRepo.escrows[bookingID2].EscrowStatus)
	}
	if bookingRepo.bookings[bookingID2].Status != domain.BookingStatusRefunded {
		t.Fatalf("expected booking status 'refunded', got: %s", bookingRepo.bookings[bookingID2].Status)
	}
}

package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/google/uuid"
)

func TestChatService_ContextualChatAndAuthorization(t *testing.T) {
	chatRepo := NewMockChatRepository()
	containerRepo := &MockContainerRepository{containers: make(map[uuid.UUID]*domain.Container)}
	outboxRepo := NewMockOutboxRepository()
	bookingRepo := NewMockBookingRepository(containerRepo, outboxRepo).(*MockBookingRepository)
	chatSvc := service.NewChatService(chatRepo, bookingRepo, containerRepo)
	ctx := context.Background()

	providerID := uuid.New()
	traderID := uuid.New()
	outsiderID := uuid.New()

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

	// 1. Outsider tries to send a message -> rejected with 403 Forbidden
	_, err := chatSvc.SendMessage(ctx, outsiderID, bookingID, "Can I see your cargo?", "")
	if err == nil {
		t.Fatalf("expected error for unauthorized non-participant, got nil")
	}

	// 2. Trader sends message
	msg1, err := chatSvc.SendMessage(ctx, traderID, bookingID, "Hi, driver will arrive at Gate 4 at 10 AM.", "")
	if err != nil {
		t.Fatalf("expected success sending chat message, got: %v", err)
	}
	if msg1.SenderID != traderID {
		t.Fatalf("expected sender %s, got %s", traderID, msg1.SenderID)
	}

	// 3. Provider replies
	_, err = chatSvc.SendMessage(ctx, providerID, bookingID, "Noted. Dock entry slip is verified.", "")
	if err != nil {
		t.Fatalf("expected success from provider, got: %v", err)
	}

	// 4. Retrieve messages and verify read tracking
	msgs, err := chatSvc.GetMessages(ctx, providerID, bookingID, 50, nil)
	if err != nil {
		t.Fatalf("expected nil error getting messages, got: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	// Verify that trader's message was marked as read by provider
	for _, m := range msgs {
		if m.SenderID == traderID && !m.IsRead {
			t.Fatalf("expected trader message to be marked read after provider viewed it")
		}
	}
}

func TestNotificationDispatcher_OutboxProcessing(t *testing.T) {
	outboxRepo := NewMockOutboxRepository()
	dispatcher := service.NewNotificationDispatcher(outboxRepo)
	ctx := context.Background()

	// Add test events to outbox
	bID := uuid.New()
	payload, _ := json.Marshal(map[string]interface{}{
		"booking_id": bID,
		"status":     "confirmed",
	})
	outboxRepo.events = append(outboxRepo.events, domain.OutboxEvent{
		ID:          uuid.New(),
		EventType:   "booking.confirmed",
		AggregateID: bID,
		Payload:     payload,
		Status:      domain.OutboxStatusPending,
		CreatedAt:   time.Now(),
	})

	dispatched, err := dispatcher.DispatchPendingNotifications(ctx, 10)
	if err != nil {
		t.Fatalf("expected success dispatching outbox, got: %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("expected 1 event dispatched, got %d", dispatched)
	}
	if outboxRepo.events[0].Status != domain.OutboxStatusCompleted {
		t.Fatalf("expected event status 'completed', got: %s", outboxRepo.events[0].Status)
	}
}

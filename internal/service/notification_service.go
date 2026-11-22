package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
)

type NotificationDispatcher interface {
	DispatchPendingNotifications(ctx context.Context, batchSize int) (int, error)
}

type notificationDispatcher struct {
	outboxRepo repository.OutboxRepository
}

func NewNotificationDispatcher(outboxRepo repository.OutboxRepository) NotificationDispatcher {
	return &notificationDispatcher{outboxRepo: outboxRepo}
}

func (d *notificationDispatcher) DispatchPendingNotifications(ctx context.Context, batchSize int) (int, error) {
	events, err := d.outboxRepo.GetPendingEvents(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	dispatched := 0
	for _, event := range events {
		if err := d.processEvent(ctx, event); err != nil {
			log.Printf("[Outbox] Event %s failed: %v", event.ID, err)
			_ = d.outboxRepo.MarkEventFailed(ctx, event.ID, err.Error(), event.RetryCount+1)
		} else {
			_ = d.outboxRepo.MarkEventCompleted(ctx, event.ID)
			dispatched++
		}
	}

	return dispatched, nil
}

func (d *notificationDispatcher) processEvent(ctx context.Context, event domain.OutboxEvent) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		payload = make(map[string]interface{})
	}

	switch event.EventType {
	case "booking.held":
		log.Printf("[Notification] Push: Cargo space held for 10 minutes. Booking: %v", payload["booking_id"])
	case "booking.confirmed":
		log.Printf("[Notification] WhatsApp + FCM: Booking confirmed! Order: %v. Space reserved.", payload["booking_id"])
	case "booking.expired":
		log.Printf("[Notification] Push: Hold expired. Space returned to inventory: %v", payload["booking_id"])
	case "escrow.payout_released":
		log.Printf("[Notification] WhatsApp: Provider payout released upon POD delivery confirmation: %v", payload["booking_id"])
	case "refund.processed":
		log.Printf("[Notification] WhatsApp: Refund initiated: %v, Amount: %v", payload["booking_id"], payload["refund_amount_paise"])
	default:
		log.Printf("[Notification] Event dispatched: %s (ID: %s)", event.EventType, event.ID)
	}

	return nil
}

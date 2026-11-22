package domain

import (
	"time"

	"github.com/google/uuid"
)

type ChatMessage struct {
	ID            uuid.UUID `json:"id"`
	BookingID     uuid.UUID `json:"booking_id"`
	SenderID      uuid.UUID `json:"sender_id"`
	Content       string    `json:"content"`
	AttachmentURL string    `json:"attachment_url,omitempty"`
	IsRead        bool      `json:"is_read"`
	CreatedAt     time.Time `json:"created_at"`
}

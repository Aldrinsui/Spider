package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type ChatRepository interface {
	SendMessage(ctx context.Context, msg *domain.ChatMessage) error
	ListMessages(ctx context.Context, bookingID uuid.UUID, limit int, before *time.Time) ([]domain.ChatMessage, error)
	MarkMessagesAsRead(ctx context.Context, bookingID uuid.UUID, readerID uuid.UUID) error
}

type sqlChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) ChatRepository {
	return &sqlChatRepository{db: db}
}

func (r *sqlChatRepository) SendMessage(ctx context.Context, msg *domain.ChatMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	msg.CreatedAt = time.Now()

	insertQuery := `
		INSERT INTO chat_messages (id, booking_id, sender_id, content, attachment_url, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.ExecContext(ctx, insertQuery,
		msg.ID, msg.BookingID, msg.SenderID, msg.Content, msg.AttachmentURL, msg.IsRead, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert chat message: %w", err)
	}

	// Trigger PostgreSQL LISTEN/NOTIFY channel
	channel := fmt.Sprintf("chat_%s", msg.BookingID.String())
	payloadBytes, _ := json.Marshal(msg)
	notifyQuery := fmt.Sprintf("NOTIFY %s, '%s'", channel, string(payloadBytes))
	_, _ = tx.ExecContext(ctx, notifyQuery)

	return tx.Commit()
}

func (r *sqlChatRepository) ListMessages(ctx context.Context, bookingID uuid.UUID, limit int, before *time.Time) ([]domain.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var query string
	var args []interface{}

	if before != nil {
		query = `
			SELECT id, booking_id, sender_id, content, attachment_url, is_read, created_at
			FROM chat_messages
			WHERE booking_id = $1 AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []interface{}{bookingID, *before, limit}
	} else {
		query = `
			SELECT id, booking_id, sender_id, content, attachment_url, is_read, created_at
			FROM chat_messages
			WHERE booking_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []interface{}{bookingID, limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.ChatMessage
	for rows.Next() {
		var m domain.ChatMessage
		var attachment sql.NullString
		if err := rows.Scan(
			&m.ID, &m.BookingID, &m.SenderID, &m.Content, &attachment, &m.IsRead, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		if attachment.Valid {
			m.AttachmentURL = attachment.String
		}
		messages = append(messages, m)
	}

	// Reverse to chronological order (oldest to newest)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *sqlChatRepository) MarkMessagesAsRead(ctx context.Context, bookingID uuid.UUID, readerID uuid.UUID) error {
	query := `
		UPDATE chat_messages
		SET is_read = TRUE
		WHERE booking_id = $1 AND sender_id != $2 AND is_read = FALSE
	`
	_, err := r.db.ExecContext(ctx, query, bookingID, readerID)
	return err
}

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type OutboxRepository interface {
	GetPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkEventCompleted(ctx context.Context, id uuid.UUID) error
	MarkEventFailed(ctx context.Context, id uuid.UUID, lastErr string, retryCount int) error
}

type sqlOutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) OutboxRepository {
	return &sqlOutboxRepository{db: db}
}

func (r *sqlOutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	query := `
		SELECT id, event_type, aggregate_type, aggregate_id, payload, status, retry_count, max_retries, last_error, next_attempt_at, created_at, processed_at
		FROM outbox
		WHERE status = 'pending' AND next_attempt_at <= NOW()
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	for rows.Next() {
		var e domain.OutboxEvent
		var lastErr sql.NullString
		var processedAt sql.NullTime

		if err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.AggregateType,
			&e.AggregateID,
			&e.Payload,
			&e.Status,
			&e.RetryCount,
			&e.MaxRetries,
			&lastErr,
			&e.NextAttemptAt,
			&e.CreatedAt,
			&processedAt,
		); err != nil {
			return nil, err
		}

		if lastErr.Valid {
			e.LastError = lastErr.String
		}
		if processedAt.Valid {
			e.ProcessedAt = &processedAt.Time
		}

		events = append(events, e)
	}

	return events, nil
}

func (r *sqlOutboxRepository) MarkEventCompleted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE outbox
		SET status = 'completed', processed_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *sqlOutboxRepository) MarkEventFailed(ctx context.Context, id uuid.UUID, lastErr string, retryCount int) error {
	nextAttempt := time.Now().Add(time.Duration(retryCount*retryCount*2) * time.Second)
	status := domain.OutboxStatusPending
	if retryCount >= 5 {
		status = domain.OutboxStatusFailed
	}

	query := `
		UPDATE outbox
		SET status = $2, retry_count = $3, last_error = $4, next_attempt_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, retryCount, lastErr, nextAttempt)
	return err
}

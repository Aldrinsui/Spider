package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type DisputeItem struct {
	ID              uuid.UUID       `json:"id"`
	BookingID       uuid.UUID       `json:"booking_id"`
	RaisedBy        uuid.UUID       `json:"raised_by"`
	Reason          string          `json:"reason"`
	Description     string          `json:"description"`
	EvidenceURLs    json.RawMessage `json:"evidence_urls"`
	Status          string          `json:"status"`
	ResolutionNotes string          `json:"resolution_notes,omitempty"`
	ResolvedBy      *uuid.UUID      `json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type DisputeRepository interface {
	CreateDispute(ctx context.Context, d *DisputeItem) error
	GetDisputeByID(ctx context.Context, id uuid.UUID) (*DisputeItem, error)
	ListDisputes(ctx context.Context, status string) ([]DisputeItem, error)
	ResolveDispute(ctx context.Context, id uuid.UUID, status string, notes string, adminID uuid.UUID) error
}

type sqlDisputeRepository struct {
	db *sql.DB
}

func NewDisputeRepository(db *sql.DB) DisputeRepository {
	return &sqlDisputeRepository{db: db}
}

func (r *sqlDisputeRepository) CreateDispute(ctx context.Context, d *DisputeItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.Status == "" {
		d.Status = "open"
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	insertQuery := `
		INSERT INTO disputes (id, booking_id, raised_by, reason, description, evidence_urls, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, insertQuery,
		d.ID, d.BookingID, d.RaisedBy, d.Reason, d.Description, d.EvidenceURLs, d.Status, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Update booking status to disputed
	_, err = tx.ExecContext(ctx, `UPDATE bookings SET status = 'disputed', updated_at = NOW() WHERE id = $1`, d.BookingID)
	if err != nil {
		return err
	}

	// Freeze escrow
	_, err = tx.ExecContext(ctx, `UPDATE payments_escrow SET escrow_status = 'disputed_freeze', updated_at = NOW() WHERE booking_id = $1`, d.BookingID)
	if err != nil {
		return err
	}

	// Outbox event
	outboxQuery := `
		INSERT INTO outbox (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at)
		VALUES ($1, 'dispute.raised', 'dispute', $2, json_build_object('dispute_id', $1, 'booking_id', $2), 'pending', NOW(), NOW())
	`
	_, _ = tx.ExecContext(ctx, outboxQuery, d.ID, d.BookingID)

	return tx.Commit()
}

func (r *sqlDisputeRepository) GetDisputeByID(ctx context.Context, id uuid.UUID) (*DisputeItem, error) {
	query := `
		SELECT id, booking_id, raised_by, reason, description, evidence_urls, status, resolution_notes, resolved_by, resolved_at, created_at, updated_at
		FROM disputes
		WHERE id = $1
	`
	d := &DisputeItem{}
	var notes, resolvedByStr sql.NullString
	var resAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.BookingID, &d.RaisedBy, &d.Reason, &d.Description, &d.EvidenceURLs, &d.Status,
		&notes, &resolvedByStr, &resAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if notes.Valid {
		d.ResolutionNotes = notes.String
	}
	if resolvedByStr.Valid {
		if uid, err := uuid.Parse(resolvedByStr.String); err == nil {
			d.ResolvedBy = &uid
		}
	}
	if resAt.Valid {
		d.ResolvedAt = &resAt.Time
	}

	return d, nil
}

func (r *sqlDisputeRepository) ListDisputes(ctx context.Context, status string) ([]DisputeItem, error) {
	var query string
	var args []interface{}
	if status != "" {
		query = `SELECT id, booking_id, raised_by, reason, description, evidence_urls, status, resolution_notes, resolved_by, resolved_at, created_at, updated_at FROM disputes WHERE status = $1 ORDER BY created_at DESC`
		args = []interface{}{status}
	} else {
		query = `SELECT id, booking_id, raised_by, reason, description, evidence_urls, status, resolution_notes, resolved_by, resolved_at, created_at, updated_at FROM disputes ORDER BY created_at DESC`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DisputeItem
	for rows.Next() {
		var d DisputeItem
		var notes, resolvedByStr sql.NullString
		var resAt sql.NullTime

		if err := rows.Scan(
			&d.ID, &d.BookingID, &d.RaisedBy, &d.Reason, &d.Description, &d.EvidenceURLs, &d.Status,
			&notes, &resolvedByStr, &resAt, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if notes.Valid {
			d.ResolutionNotes = notes.String
		}
		if resolvedByStr.Valid {
			if uid, err := uuid.Parse(resolvedByStr.String); err == nil {
				d.ResolvedBy = &uid
			}
		}
		if resAt.Valid {
			d.ResolvedAt = &resAt.Time
		}

		list = append(list, d)
	}

	return list, nil
}

func (r *sqlDisputeRepository) ResolveDispute(ctx context.Context, id uuid.UUID, status string, notes string, adminID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var bookingID uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT booking_id FROM disputes WHERE id = $1 FOR UPDATE`, id).Scan(&bookingID)
	if err != nil {
		return err
	}

	// Update dispute
	updateDispute := `
		UPDATE disputes
		SET status = $2, resolution_notes = $3, resolved_by = $4, resolved_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.ExecContext(ctx, updateDispute, id, status, notes, adminID); err != nil {
		return err
	}

	if status == "resolved_refund" {
		_, _ = tx.ExecContext(ctx, `UPDATE bookings SET status = 'refunded', updated_at = NOW() WHERE id = $1`, bookingID)
		_, _ = tx.ExecContext(ctx, `UPDATE payments_escrow SET escrow_status = 'refunded_to_trader', refunded_at = NOW(), updated_at = NOW() WHERE booking_id = $1`, bookingID)
	} else if status == "resolved_payout" {
		_, _ = tx.ExecContext(ctx, `UPDATE bookings SET status = 'delivered', updated_at = NOW() WHERE id = $1`, bookingID)
		_, _ = tx.ExecContext(ctx, `UPDATE payments_escrow SET escrow_status = 'released_to_provider', payout_released_at = NOW(), updated_at = NOW() WHERE booking_id = $1`, bookingID)
	}

	return tx.Commit()
}

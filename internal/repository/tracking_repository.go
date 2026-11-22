package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type DeliveryProof struct {
	ID                    uuid.UUID `json:"id"`
	BookingID             uuid.UUID `json:"booking_id"`
	PODPhotoURL           string    `json:"pod_photo_url"`
	Latitude              *float64  `json:"latitude,omitempty"`
	Longitude             *float64  `json:"longitude,omitempty"`
	RecipientName         string    `json:"recipient_name"`
	RecipientPhone        string    `json:"recipient_phone"`
	RecipientSignatureURL string    `json:"recipient_signature_url,omitempty"`
	ConfirmedByTrader     bool      `json:"confirmed_by_trader"`
	DeliveredAt           time.Time `json:"delivered_at"`
}

type TrackingRepository interface {
	UpdateContainerLocation(ctx context.Context, containerID uuid.UUID, lat, lng float64) error
	SubmitDeliveryProof(ctx context.Context, proof *DeliveryProof) error
	GetDeliveryProof(ctx context.Context, bookingID uuid.UUID) (*DeliveryProof, error)
}

type sqlTrackingRepository struct {
	db *sql.DB
}

func NewTrackingRepository(db *sql.DB) TrackingRepository {
	return &sqlTrackingRepository{db: db}
}

func (r *sqlTrackingRepository) UpdateContainerLocation(ctx context.Context, containerID uuid.UUID, lat, lng float64) error {
	query := `
		UPDATE containers
		SET current_location = ST_SetSRID(ST_MakePoint($2, $3), 4326),
		    last_location_updated_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, containerID, lng, lat)
	return err
}

func (r *sqlTrackingRepository) SubmitDeliveryProof(ctx context.Context, proof *DeliveryProof) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if proof.ID == uuid.Nil {
		proof.ID = uuid.New()
	}
	proof.DeliveredAt = time.Now()

	insertProofQuery := `
		INSERT INTO delivery_proofs (
			id, booking_id, pod_photo_url, latitude, longitude, recipient_name, recipient_phone, recipient_signature_url, confirmed_by_trader, delivered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.ExecContext(ctx, insertProofQuery,
		proof.ID, proof.BookingID, proof.PODPhotoURL, proof.Latitude, proof.Longitude,
		proof.RecipientName, proof.RecipientPhone, proof.RecipientSignatureURL, proof.ConfirmedByTrader, proof.DeliveredAt,
	)
	if err != nil {
		return err
	}

	// Transition booking to delivered
	updateBookingQuery := `
		UPDATE bookings
		SET status = 'delivered', delivered_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.ExecContext(ctx, updateBookingQuery, proof.BookingID); err != nil {
		return err
	}

	// Release Escrow payout to provider automatically upon POD submission
	updateEscrowQuery := `
		UPDATE payments_escrow
		SET escrow_status = 'released_to_provider', payout_released_at = NOW(), updated_at = NOW()
		WHERE booking_id = $1 AND escrow_status = 'held'
	`
	if _, err := tx.ExecContext(ctx, updateEscrowQuery, proof.BookingID); err != nil {
		return err
	}

	// Insert outbox notification
	outboxQuery := `
		INSERT INTO outbox (
			id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at
		) VALUES ($1, 'booking.delivered', 'booking', $2, json_build_object('booking_id', $2, 'pod_url', $3), 'pending', NOW(), NOW())
	`
	if _, err := tx.ExecContext(ctx, outboxQuery, uuid.New(), proof.BookingID, proof.PODPhotoURL); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlTrackingRepository) GetDeliveryProof(ctx context.Context, bookingID uuid.UUID) (*DeliveryProof, error) {
	query := `
		SELECT id, booking_id, pod_photo_url, latitude, longitude, recipient_name, recipient_phone, recipient_signature_url, confirmed_by_trader, delivered_at
		FROM delivery_proofs
		WHERE booking_id = $1
	`
	p := &DeliveryProof{}
	var lat, lng sql.NullFloat64
	var sig sql.NullString

	err := r.db.QueryRowContext(ctx, query, bookingID).Scan(
		&p.ID, &p.BookingID, &p.PODPhotoURL, &lat, &lng, &p.RecipientName, &p.RecipientPhone, &sig, &p.ConfirmedByTrader, &p.DeliveredAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if lat.Valid {
		p.Latitude = &lat.Float64
	}
	if lng.Valid {
		p.Longitude = &lng.Float64
	}
	if sig.Valid {
		p.RecipientSignatureURL = sig.String
	}

	return p, nil
}

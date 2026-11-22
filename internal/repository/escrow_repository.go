package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type EscrowRepository interface {
	CreatePaymentEscrow(ctx context.Context, escrow *domain.PaymentEscrow) error
	GetEscrowByBookingID(ctx context.Context, bookingID uuid.UUID) (*domain.PaymentEscrow, error)
	GetEscrowByOrderID(ctx context.Context, orderID string) (*domain.PaymentEscrow, error)
	IsWebhookProcessed(ctx context.Context, paymentID string) (bool, error)
	RecordProcessedWebhook(ctx context.Context, eventID, paymentID, eventType string, payload json.RawMessage) error

	ConfirmPaymentAndHoldEscrow(ctx context.Context, bookingID uuid.UUID, paymentID, signature, routeTransferID string) error
	ReconcileLatePayment(ctx context.Context, bookingID uuid.UUID, paymentID, signature string) (bool, error)
	ReleaseEscrowPayout(ctx context.Context, bookingID uuid.UUID) error
	ProcessRefund(ctx context.Context, bookingID uuid.UUID, refundID string, refundAmountPaise int64, reason string) error
}

type sqlEscrowRepository struct {
	db *sql.DB
}

func NewEscrowRepository(db *sql.DB) EscrowRepository {
	return &sqlEscrowRepository{db: db}
}

func (r *sqlEscrowRepository) CreatePaymentEscrow(ctx context.Context, e *domain.PaymentEscrow) error {
	query := `
		INSERT INTO payments_escrow (
			id, booking_id, razorpay_order_id, amount_paise, currency, escrow_status,
			provider_payout_amount_paise, platform_commission_paise, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Currency == "" {
		e.Currency = "INR"
	}
	if e.EscrowStatus == "" {
		e.EscrowStatus = domain.EscrowStatusPending
	}

	return r.db.QueryRowContext(ctx, query,
		e.ID, e.BookingID, e.RazorpayOrderID, e.AmountPaise, e.Currency, e.EscrowStatus,
		e.ProviderPayoutAmountPaise, e.PlatformCommissionPaise,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (r *sqlEscrowRepository) GetEscrowByBookingID(ctx context.Context, bookingID uuid.UUID) (*domain.PaymentEscrow, error) {
	query := `
		SELECT 
			id, booking_id, razorpay_order_id, razorpay_payment_id, razorpay_signature,
			amount_paise, currency, escrow_status, provider_route_transfer_id,
			provider_payout_amount_paise, platform_commission_paise, payout_released_at,
			refund_id, refund_amount_paise, refunded_at, created_at, updated_at
		FROM payments_escrow
		WHERE booking_id = $1
	`
	e := &domain.PaymentEscrow{}
	var payID, sig, routeTransferID, refID sql.NullString
	var payoutAt, refAt sql.NullTime
	var refAmt sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, bookingID).Scan(
		&e.ID, &e.BookingID, &e.RazorpayOrderID, &payID, &sig,
		&e.AmountPaise, &e.Currency, &e.EscrowStatus, &routeTransferID,
		&e.ProviderPayoutAmountPaise, &e.PlatformCommissionPaise, &payoutAt,
		&refID, &refAmt, &refAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if payID.Valid {
		e.RazorpayPaymentID = payID.String
	}
	if sig.Valid {
		e.RazorpaySignature = sig.String
	}
	if routeTransferID.Valid {
		e.ProviderRouteTransferID = routeTransferID.String
	}
	if payoutAt.Valid {
		e.PayoutReleasedAt = &payoutAt.Time
	}
	if refID.Valid {
		e.RefundID = refID.String
	}
	if refAmt.Valid {
		e.RefundAmountPaise = &refAmt.Int64
	}
	if refAt.Valid {
		e.RefundedAt = &refAt.Time
	}

	return e, nil
}

func (r *sqlEscrowRepository) GetEscrowByOrderID(ctx context.Context, orderID string) (*domain.PaymentEscrow, error) {
	query := `
		SELECT 
			id, booking_id, razorpay_order_id, razorpay_payment_id, razorpay_signature,
			amount_paise, currency, escrow_status, provider_route_transfer_id,
			provider_payout_amount_paise, platform_commission_paise, payout_released_at,
			refund_id, refund_amount_paise, refunded_at, created_at, updated_at
		FROM payments_escrow
		WHERE razorpay_order_id = $1
	`
	e := &domain.PaymentEscrow{}
	var payID, sig, routeTransferID, refID sql.NullString
	var payoutAt, refAt sql.NullTime
	var refAmt sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&e.ID, &e.BookingID, &e.RazorpayOrderID, &payID, &sig,
		&e.AmountPaise, &e.Currency, &e.EscrowStatus, &routeTransferID,
		&e.ProviderPayoutAmountPaise, &e.PlatformCommissionPaise, &payoutAt,
		&refID, &refAmt, &refAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if payID.Valid {
		e.RazorpayPaymentID = payID.String
	}
	if sig.Valid {
		e.RazorpaySignature = sig.String
	}
	if routeTransferID.Valid {
		e.ProviderRouteTransferID = routeTransferID.String
	}
	if payoutAt.Valid {
		e.PayoutReleasedAt = &payoutAt.Time
	}
	if refID.Valid {
		e.RefundID = refID.String
	}
	if refAmt.Valid {
		e.RefundAmountPaise = &refAmt.Int64
	}
	if refAt.Valid {
		e.RefundedAt = &refAt.Time
	}

	return e, nil
}

func (r *sqlEscrowRepository) IsWebhookProcessed(ctx context.Context, paymentID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM processed_webhooks WHERE payment_id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, paymentID).Scan(&exists)
	return exists, err
}

func (r *sqlEscrowRepository) RecordProcessedWebhook(ctx context.Context, eventID, paymentID, eventType string, payload json.RawMessage) error {
	query := `
		INSERT INTO processed_webhooks (id, event_id, payment_id, event_type, payload, processed_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (payment_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, uuid.New(), eventID, paymentID, eventType, payload)
	return err
}

func (r *sqlEscrowRepository) ConfirmPaymentAndHoldEscrow(ctx context.Context, bookingID uuid.UUID, paymentID, signature, routeTransferID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Update booking to confirmed
	updateBookingQuery := `
		UPDATE bookings
		SET status = 'confirmed', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.ExecContext(ctx, updateBookingQuery, bookingID); err != nil {
		return err
	}

	// Update escrow status to held (funds safely in escrow pending POD)
	updateEscrowQuery := `
		UPDATE payments_escrow
		SET razorpay_payment_id = $2,
		    razorpay_signature = $3,
		    provider_route_transfer_id = $4,
		    escrow_status = 'held',
		    updated_at = NOW()
		WHERE booking_id = $1
	`
	if _, err := tx.ExecContext(ctx, updateEscrowQuery, bookingID, paymentID, signature, routeTransferID); err != nil {
		return err
	}

	// Insert outbox notification for confirmed booking
	outboxQuery := `
		INSERT INTO outbox (
			id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at
		) VALUES ($1, 'booking.confirmed', 'booking', $2, json_build_object('booking_id', $2, 'payment_id', $3), 'pending', NOW(), NOW())
	`
	if _, err := tx.ExecContext(ctx, outboxQuery, uuid.New(), bookingID, paymentID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlEscrowRepository) ReconcileLatePayment(ctx context.Context, bookingID uuid.UUID, paymentID, signature string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	// Get booking and container details
	var containerID uuid.UUID
	var reqCBM, reqWeight float64
	var status string
	bQuery := `SELECT container_id, chargeable_volume_cbm, actual_weight_kg, status FROM bookings WHERE id = $1 FOR UPDATE`
	if err := tx.QueryRowContext(ctx, bQuery, bookingID).Scan(&containerID, &reqCBM, &reqWeight, &status); err != nil {
		return false, err
	}

	// Check container availability
	var availCBM, availWeight float64
	cQuery := `SELECT available_cbm, available_weight_kg FROM containers WHERE id = $1 FOR UPDATE`
	if err := tx.QueryRowContext(ctx, cQuery, containerID).Scan(&availCBM, &availWeight); err != nil {
		return false, err
	}

	// Rule: If space is still free, confirm booking. Otherwise, auto-refund!
	if availCBM >= reqCBM && availWeight >= reqWeight {
		// Decrement container capacity
		_, _ = tx.ExecContext(ctx, `UPDATE containers SET available_cbm = available_cbm - $2, available_weight_kg = available_weight_kg - $3 WHERE id = $1`, containerID, reqCBM, reqWeight)
		// Confirm booking
		_, _ = tx.ExecContext(ctx, `UPDATE bookings SET status = 'confirmed', confirmed_at = NOW(), updated_at = NOW() WHERE id = $1`, bookingID)
		// Set escrow to held
		_, _ = tx.ExecContext(ctx, `UPDATE payments_escrow SET razorpay_payment_id = $2, razorpay_signature = $3, escrow_status = 'held', updated_at = NOW() WHERE booking_id = $1`, bookingID, paymentID, signature)
		// Outbox event
		_, _ = tx.ExecContext(ctx, `INSERT INTO outbox (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at) VALUES ($1, 'booking.late_confirmed', 'booking', $2, json_build_object('booking_id', $2, 'payment_id', $3), 'pending', NOW(), NOW())`, uuid.New(), bookingID, paymentID)

		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil // Confirmed
	}

	// Space no longer free -> trigger automated refund!
	_, _ = tx.ExecContext(ctx, `UPDATE bookings SET status = 'refunded', updated_at = NOW() WHERE id = $1`, bookingID)
	_, _ = tx.ExecContext(ctx, `UPDATE payments_escrow SET razorpay_payment_id = $2, razorpay_signature = $3, escrow_status = 'refunded_to_trader', refunded_at = NOW(), updated_at = NOW() WHERE booking_id = $1`, bookingID, paymentID, signature)
	// Outbox event for payment refund
	_, _ = tx.ExecContext(ctx, `INSERT INTO outbox (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at) VALUES ($1, 'refund.auto_triggered', 'booking', $2, json_build_object('booking_id', $2, 'payment_id', $3, 'reason', 'space_no_longer_available_on_late_payment'), 'pending', NOW(), NOW())`, uuid.New(), bookingID, paymentID)

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return false, nil // Refunded
}

func (r *sqlEscrowRepository) ReleaseEscrowPayout(ctx context.Context, bookingID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	query := `
		UPDATE payments_escrow
		SET escrow_status = 'released_to_provider', payout_released_at = NOW(), updated_at = NOW()
		WHERE booking_id = $1 AND escrow_status = 'held'
	`
	res, err := tx.ExecContext(ctx, query, bookingID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("escrow is not in held status or already released")
	}

	outboxQuery := `
		INSERT INTO outbox (
			id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at
		) VALUES ($1, 'escrow.payout_released', 'escrow', $2, json_build_object('booking_id', $2), 'pending', NOW(), NOW())
	`
	if _, err := tx.ExecContext(ctx, outboxQuery, uuid.New(), bookingID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlEscrowRepository) ProcessRefund(ctx context.Context, bookingID uuid.UUID, refundID string, refundAmountPaise int64, reason string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Update booking to cancelled / refunded
	_, err = tx.ExecContext(ctx, `UPDATE bookings SET status = 'cancelled', cancelled_at = NOW(), updated_at = NOW() WHERE id = $1`, bookingID)
	if err != nil {
		return err
	}

	// Update escrow record
	updateEscrow := `
		UPDATE payments_escrow
		SET escrow_status = 'refunded_to_trader', refund_id = $2, refund_amount_paise = $3, refunded_at = NOW(), updated_at = NOW()
		WHERE booking_id = $1
	`
	if _, err := tx.ExecContext(ctx, updateEscrow, bookingID, refundID, refundAmountPaise); err != nil {
		return err
	}

	// Restore container space
	restoreSpace := `
		UPDATE containers
		SET available_cbm = available_cbm + b.chargeable_volume_cbm,
		    available_weight_kg = available_weight_kg + b.actual_weight_kg,
		    updated_at = NOW()
		FROM bookings b
		WHERE containers.id = b.container_id AND b.id = $1
	`
	if _, err := tx.ExecContext(ctx, restoreSpace, bookingID); err != nil {
		return err
	}

	// Outbox event
	outboxQuery := `
		INSERT INTO outbox (
			id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at
		) VALUES ($1, 'refund.processed', 'escrow', $2, json_build_object('booking_id', $2, 'refund_amount_paise', $3, 'reason', $4), 'pending', NOW(), NOW())
	`
	if _, err := tx.ExecContext(ctx, outboxQuery, uuid.New(), bookingID, refundAmountPaise, reason); err != nil {
		return err
	}

	return tx.Commit()
}

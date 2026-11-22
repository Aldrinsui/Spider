package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type BookingRepository interface {
	HoldSpaceTransaction(ctx context.Context, b *domain.Booking, outboxEvent *domain.OutboxEvent) error
	ReclaimExpiredHolds(ctx context.Context) (int, error)
	GetBookingByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	GetBookingByRef(ctx context.Context, ref string) (*domain.Booking, error)
	ListBookingsByTrader(ctx context.Context, traderID uuid.UUID) ([]domain.Booking, error)
	ListBookingsByContainer(ctx context.Context, containerID uuid.UUID) ([]domain.Booking, error)
	UpdateBookingStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error
}

type sqlBookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) BookingRepository {
	return &sqlBookingRepository{db: db}
}

func (r *sqlBookingRepository) HoldSpaceTransaction(ctx context.Context, b *domain.Booking, outboxEvent *domain.OutboxEvent) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Lock container row with SELECT ... FOR UPDATE to eliminate race conditions
	var availableCBM float64
	var availableWeightKG float64
	lockQuery := `
		SELECT available_cbm, available_weight_kg 
		FROM containers 
		WHERE id = $1 AND status = 'scheduled'
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, lockQuery, b.ContainerID).Scan(&availableCBM, &availableWeightKG)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("container not found or not available for booking")
		}
		return fmt.Errorf("failed to lock container row: %w", err)
	}

	// 2. Validate capacity strictly before decrementing
	if availableCBM < b.ChargeableVolumeCBM {
		return fmt.Errorf("insufficient container capacity: requested %.2f CBM, but only %.2f CBM is available", b.ChargeableVolumeCBM, availableCBM)
	}
	if availableWeightKG < b.ActualWeightKG {
		return fmt.Errorf("insufficient payload weight capacity: requested %.2f kg, but only %.2f kg is available", b.ActualWeightKG, availableWeightKG)
	}

	// 3. Decrement container capacity atomically
	updateContainerQuery := `
		UPDATE containers
		SET available_cbm = available_cbm - $2,
		    available_weight_kg = available_weight_kg - $3,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, updateContainerQuery, b.ContainerID, b.ChargeableVolumeCBM, b.ActualWeightKG)
	if err != nil {
		return fmt.Errorf("failed to decrement container space: %w", err)
	}

	// 4. Insert booking in 'held' status with 10-minute hold window
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.BookingRef == "" {
		b.BookingRef = fmt.Sprintf("SP-%s-%s", time.Now().Format("20060102"), b.ID.String()[:8])
	}
	now := time.Now()
	holdExpiry := now.Add(10 * time.Minute)
	b.Status = domain.BookingStatusHeld
	b.HoldExpiresAt = &holdExpiry
	b.CreatedAt = now
	b.UpdatedAt = now

	insertBookingQuery := `
		INSERT INTO bookings (
			id, booking_ref, container_id, trader_id, status,
			consignment_length_cm, consignment_width_cm, consignment_height_cm,
			actual_weight_kg, computed_volume_cbm, chargeable_volume_cbm, chargeable_weight_kg,
			is_dangerous_goods, cargo_description, eway_bill_ref,
			base_amount_paise, platform_fee_paise, gst_amount_paise, total_amount_paise,
			hold_expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22
		)
	`
	_, err = tx.ExecContext(ctx, insertBookingQuery,
		b.ID, b.BookingRef, b.ContainerID, b.TraderID, b.Status,
		b.ConsignmentLengthCM, b.ConsignmentWidthCM, b.ConsignmentHeightCM,
		b.ActualWeightKG, b.ComputedVolumeCBM, b.ChargeableVolumeCBM, b.ChargeableWeightKG,
		b.IsDangerousGoods, b.CargoDescription, b.EWayBillRef,
		b.BaseAmountPaise, b.PlatformFeePaise, b.GSTAmountPaise, b.TotalAmountPaise,
		b.HoldExpiresAt, b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert booking record: %w", err)
	}

	// 5. Insert Transactional Outbox event in same transaction
	if outboxEvent != nil {
		if outboxEvent.ID == uuid.Nil {
			outboxEvent.ID = uuid.New()
		}
		outboxEvent.CreatedAt = now
		outboxEvent.NextAttemptAt = now
		if outboxEvent.Status == "" {
			outboxEvent.Status = domain.OutboxStatusPending
		}

		insertOutboxQuery := `
			INSERT INTO outbox (
				id, event_type, aggregate_type, aggregate_id, payload, status, retry_count, max_retries, next_attempt_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`
		_, err = tx.ExecContext(ctx, insertOutboxQuery,
			outboxEvent.ID, outboxEvent.EventType, outboxEvent.AggregateType, outboxEvent.AggregateID,
			outboxEvent.Payload, outboxEvent.Status, outboxEvent.RetryCount, outboxEvent.MaxRetries,
			outboxEvent.NextAttemptAt, outboxEvent.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to write transactional outbox event: %w", err)
		}
	}

	// 6. Commit atomic transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit hold transaction: %w", err)
	}

	return nil
}

func (r *sqlBookingRepository) ReclaimExpiredHolds(ctx context.Context) (int, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Find expired holds with SKIP LOCKED to support multiple background reclaimer workers without conflict
	findExpiredQuery := `
		SELECT id, container_id, chargeable_volume_cbm, actual_weight_kg
		FROM bookings
		WHERE status = 'held' AND hold_expires_at < NOW()
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.QueryContext(ctx, findExpiredQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to query expired holds: %w", err)
	}
	defer rows.Close()

	type expiredItem struct {
		bookingID   uuid.UUID
		containerID uuid.UUID
		volumeCBM   float64
		weightKG    float64
	}

	var expiredList []expiredItem
	for rows.Next() {
		var item expiredItem
		if err := rows.Scan(&item.bookingID, &item.containerID, &item.volumeCBM, &item.weightKG); err != nil {
			return 0, err
		}
		expiredList = append(expiredList, item)
	}
	rows.Close()

	if len(expiredList) == 0 {
		return 0, nil
	}

	for _, item := range expiredList {
		// 1. Restore container volume & weight
		restoreQuery := `
			UPDATE containers
			SET available_cbm = available_cbm + $2,
			    available_weight_kg = available_weight_kg + $3,
			    updated_at = NOW()
			WHERE id = $1
		`
		if _, err := tx.ExecContext(ctx, restoreQuery, item.containerID, item.volumeCBM, item.weightKG); err != nil {
			return 0, fmt.Errorf("failed to restore container capacity: %w", err)
		}

		// 2. Mark booking as expired
		updateBookingQuery := `
			UPDATE bookings
			SET status = 'expired', updated_at = NOW()
			WHERE id = $1
		`
		if _, err := tx.ExecContext(ctx, updateBookingQuery, item.bookingID); err != nil {
			return 0, fmt.Errorf("failed to mark booking expired: %w", err)
		}

		// 3. Write outbox event for notification/analytics
		payload, _ := json.Marshal(map[string]interface{}{
			"booking_id":   item.bookingID,
			"container_id": item.containerID,
			"reason":       "hold_time_exceeded",
		})
		insertOutbox := `
			INSERT INTO outbox (
				id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, created_at
			) VALUES ($1, 'booking.expired', 'booking', $2, $3, 'pending', NOW(), NOW())
		`
		if _, err := tx.ExecContext(ctx, insertOutbox, uuid.New(), item.bookingID, payload); err != nil {
			return 0, fmt.Errorf("failed to write outbox event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit reclaimer transaction: %w", err)
	}

	return len(expiredList), nil
}

func (r *sqlBookingRepository) GetBookingByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	query := `
		SELECT 
			id, booking_ref, container_id, trader_id, status,
			consignment_length_cm, consignment_width_cm, consignment_height_cm,
			actual_weight_kg, computed_volume_cbm, chargeable_volume_cbm, chargeable_weight_kg,
			is_dangerous_goods, cargo_description, eway_bill_ref,
			base_amount_paise, platform_fee_paise, gst_amount_paise, total_amount_paise,
			hold_expires_at, confirmed_at, delivered_at, cancelled_at,
			created_at, updated_at
		FROM bookings
		WHERE id = $1
	`
	b := &domain.Booking{}
	var ewayBill sql.NullString
	var holdExpires, confirmedAt, deliveredAt, cancelledAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.BookingRef, &b.ContainerID, &b.TraderID, &b.Status,
		&b.ConsignmentLengthCM, &b.ConsignmentWidthCM, &b.ConsignmentHeightCM,
		&b.ActualWeightKG, &b.ComputedVolumeCBM, &b.ChargeableVolumeCBM, &b.ChargeableWeightKG,
		&b.IsDangerousGoods, &b.CargoDescription, &ewayBill,
		&b.BaseAmountPaise, &b.PlatformFeePaise, &b.GSTAmountPaise, &b.TotalAmountPaise,
		&holdExpires, &confirmedAt, &deliveredAt, &cancelledAt,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if ewayBill.Valid {
		b.EWayBillRef = ewayBill.String
	}
	if holdExpires.Valid {
		b.HoldExpiresAt = &holdExpires.Time
	}
	if confirmedAt.Valid {
		b.ConfirmedAt = &confirmedAt.Time
	}
	if deliveredAt.Valid {
		b.DeliveredAt = &deliveredAt.Time
	}
	if cancelledAt.Valid {
		b.CancelledAt = &cancelledAt.Time
	}

	return b, nil
}

func (r *sqlBookingRepository) GetBookingByRef(ctx context.Context, ref string) (*domain.Booking, error) {
	query := `
		SELECT 
			id, booking_ref, container_id, trader_id, status,
			consignment_length_cm, consignment_width_cm, consignment_height_cm,
			actual_weight_kg, computed_volume_cbm, chargeable_volume_cbm, chargeable_weight_kg,
			is_dangerous_goods, cargo_description, eway_bill_ref,
			base_amount_paise, platform_fee_paise, gst_amount_paise, total_amount_paise,
			hold_expires_at, confirmed_at, delivered_at, cancelled_at,
			created_at, updated_at
		FROM bookings
		WHERE booking_ref = $1
	`
	b := &domain.Booking{}
	var ewayBill sql.NullString
	var holdExpires, confirmedAt, deliveredAt, cancelledAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, ref).Scan(
		&b.ID, &b.BookingRef, &b.ContainerID, &b.TraderID, &b.Status,
		&b.ConsignmentLengthCM, &b.ConsignmentWidthCM, &b.ConsignmentHeightCM,
		&b.ActualWeightKG, &b.ComputedVolumeCBM, &b.ChargeableVolumeCBM, &b.ChargeableWeightKG,
		&b.IsDangerousGoods, &b.CargoDescription, &ewayBill,
		&b.BaseAmountPaise, &b.PlatformFeePaise, &b.GSTAmountPaise, &b.TotalAmountPaise,
		&holdExpires, &confirmedAt, &deliveredAt, &cancelledAt,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if ewayBill.Valid {
		b.EWayBillRef = ewayBill.String
	}
	if holdExpires.Valid {
		b.HoldExpiresAt = &holdExpires.Time
	}
	if confirmedAt.Valid {
		b.ConfirmedAt = &confirmedAt.Time
	}
	if deliveredAt.Valid {
		b.DeliveredAt = &deliveredAt.Time
	}
	if cancelledAt.Valid {
		b.CancelledAt = &cancelledAt.Time
	}

	return b, nil
}

func (r *sqlBookingRepository) ListBookingsByTrader(ctx context.Context, traderID uuid.UUID) ([]domain.Booking, error) {
	query := `
		SELECT 
			id, booking_ref, container_id, trader_id, status,
			consignment_length_cm, consignment_width_cm, consignment_height_cm,
			actual_weight_kg, computed_volume_cbm, chargeable_volume_cbm, chargeable_weight_kg,
			is_dangerous_goods, cargo_description, eway_bill_ref,
			base_amount_paise, platform_fee_paise, gst_amount_paise, total_amount_paise,
			hold_expires_at, confirmed_at, delivered_at, cancelled_at,
			created_at, updated_at
		FROM bookings
		WHERE trader_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, traderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Booking
	for rows.Next() {
		var b domain.Booking
		var ewayBill sql.NullString
		var holdExpires, confirmedAt, deliveredAt, cancelledAt sql.NullTime

		if err := rows.Scan(
			&b.ID, &b.BookingRef, &b.ContainerID, &b.TraderID, &b.Status,
			&b.ConsignmentLengthCM, &b.ConsignmentWidthCM, &b.ConsignmentHeightCM,
			&b.ActualWeightKG, &b.ComputedVolumeCBM, &b.ChargeableVolumeCBM, &b.ChargeableWeightKG,
			&b.IsDangerousGoods, &b.CargoDescription, &ewayBill,
			&b.BaseAmountPaise, &b.PlatformFeePaise, &b.GSTAmountPaise, &b.TotalAmountPaise,
			&holdExpires, &confirmedAt, &deliveredAt, &cancelledAt,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if ewayBill.Valid {
			b.EWayBillRef = ewayBill.String
		}
		if holdExpires.Valid {
			b.HoldExpiresAt = &holdExpires.Time
		}
		if confirmedAt.Valid {
			b.ConfirmedAt = &confirmedAt.Time
		}
		if deliveredAt.Valid {
			b.DeliveredAt = &deliveredAt.Time
		}
		if cancelledAt.Valid {
			b.CancelledAt = &cancelledAt.Time
		}

		list = append(list, b)
	}

	return list, nil
}

func (r *sqlBookingRepository) ListBookingsByContainer(ctx context.Context, containerID uuid.UUID) ([]domain.Booking, error) {
	query := `
		SELECT 
			id, booking_ref, container_id, trader_id, status,
			consignment_length_cm, consignment_width_cm, consignment_height_cm,
			actual_weight_kg, computed_volume_cbm, chargeable_volume_cbm, chargeable_weight_kg,
			is_dangerous_goods, cargo_description, eway_bill_ref,
			base_amount_paise, platform_fee_paise, gst_amount_paise, total_amount_paise,
			hold_expires_at, confirmed_at, delivered_at, cancelled_at,
			created_at, updated_at
		FROM bookings
		WHERE container_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, containerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Booking
	for rows.Next() {
		var b domain.Booking
		var ewayBill sql.NullString
		var holdExpires, confirmedAt, deliveredAt, cancelledAt sql.NullTime

		if err := rows.Scan(
			&b.ID, &b.BookingRef, &b.ContainerID, &b.TraderID, &b.Status,
			&b.ConsignmentLengthCM, &b.ConsignmentWidthCM, &b.ConsignmentHeightCM,
			&b.ActualWeightKG, &b.ComputedVolumeCBM, &b.ChargeableVolumeCBM, &b.ChargeableWeightKG,
			&b.IsDangerousGoods, &b.CargoDescription, &ewayBill,
			&b.BaseAmountPaise, &b.PlatformFeePaise, &b.GSTAmountPaise, &b.TotalAmountPaise,
			&holdExpires, &confirmedAt, &deliveredAt, &cancelledAt,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}

		list = append(list, b)
	}

	return list, nil
}

func (r *sqlBookingRepository) UpdateBookingStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error {
	query := `
		UPDATE bookings
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

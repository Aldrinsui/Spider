package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	RecordDPDPConsent(ctx context.Context, userID uuid.UUID, version string) error

	CreateOTP(ctx context.Context, otp *domain.OTPVerification) error
	GetLatestActiveOTP(ctx context.Context, phone string) (*domain.OTPVerification, error)
	IncrementOTPAttempts(ctx context.Context, id uuid.UUID) error
	MarkOTPVerified(ctx context.Context, id uuid.UUID) error
}

type sqlUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &sqlUserRepository{db: db}
}

func (r *sqlUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, phone, role, full_name, email, is_active, dpdp_consent_at, dpdp_consent_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	return r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Phone,
		user.Role,
		user.FullName,
		user.Email,
		user.IsActive,
		user.DPDPConsentAt,
		user.DPDPConsentVersion,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *sqlUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, phone, role, full_name, email, is_active, dpdp_consent_at, dpdp_consent_version, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	u := &domain.User{}
	var dpdpAt sql.NullTime
	var dpdpVer sql.NullString
	var fullName, email sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Phone,
		&u.Role,
		&fullName,
		&email,
		&u.IsActive,
		&dpdpAt,
		&dpdpVer,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if fullName.Valid {
		u.FullName = fullName.String
	}
	if email.Valid {
		u.Email = email.String
	}
	if dpdpAt.Valid {
		u.DPDPConsentAt = &dpdpAt.Time
	}
	if dpdpVer.Valid {
		u.DPDPConsentVersion = dpdpVer.String
	}

	return u, nil
}

func (r *sqlUserRepository) GetUserByPhone(ctx context.Context, phone string) (*domain.User, error) {
	query := `
		SELECT id, phone, role, full_name, email, is_active, dpdp_consent_at, dpdp_consent_version, created_at, updated_at
		FROM users
		WHERE phone = $1
	`
	u := &domain.User{}
	var dpdpAt sql.NullTime
	var dpdpVer sql.NullString
	var fullName, email sql.NullString

	err := r.db.QueryRowContext(ctx, query, phone).Scan(
		&u.ID,
		&u.Phone,
		&u.Role,
		&fullName,
		&email,
		&u.IsActive,
		&dpdpAt,
		&dpdpVer,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if fullName.Valid {
		u.FullName = fullName.String
	}
	if email.Valid {
		u.Email = email.String
	}
	if dpdpAt.Valid {
		u.DPDPConsentAt = &dpdpAt.Time
	}
	if dpdpVer.Valid {
		u.DPDPConsentVersion = dpdpVer.String
	}

	return u, nil
}

func (r *sqlUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET full_name = $2, email = $3, role = $4, is_active = $5, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.FullName, user.Email, user.Role, user.IsActive)
	return err
}

func (r *sqlUserRepository) RecordDPDPConsent(ctx context.Context, userID uuid.UUID, version string) error {
	query := `
		UPDATE users
		SET dpdp_consent_at = NOW(), dpdp_consent_version = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID, version)
	return err
}

func (r *sqlUserRepository) CreateOTP(ctx context.Context, otp *domain.OTPVerification) error {
	query := `
		INSERT INTO otp_verifications (id, phone, otp_hash, attempts, max_attempts, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if otp.ID == uuid.Nil {
		otp.ID = uuid.New()
	}
	otp.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		otp.ID,
		otp.Phone,
		otp.OTPHash,
		otp.Attempts,
		otp.MaxAttempts,
		otp.ExpiresAt,
		otp.CreatedAt,
	)
	return err
}

func (r *sqlUserRepository) GetLatestActiveOTP(ctx context.Context, phone string) (*domain.OTPVerification, error) {
	query := `
		SELECT id, phone, otp_hash, attempts, max_attempts, expires_at, verified_at, created_at
		FROM otp_verifications
		WHERE phone = $1 AND verified_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`
	otp := &domain.OTPVerification{}
	var verifiedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, phone).Scan(
		&otp.ID,
		&otp.Phone,
		&otp.OTPHash,
		&otp.Attempts,
		&otp.MaxAttempts,
		&otp.ExpiresAt,
		&verifiedAt,
		&otp.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if verifiedAt.Valid {
		otp.VerifiedAt = &verifiedAt.Time
	}

	return otp, nil
}

func (r *sqlUserRepository) IncrementOTPAttempts(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE otp_verifications
		SET attempts = attempts + 1
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *sqlUserRepository) MarkOTPVerified(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE otp_verifications
		SET verified_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

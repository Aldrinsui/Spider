package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type KYCRepository interface {
	CreateOrUpdateKYC(ctx context.Context, kyc *domain.ProviderKYC) error
	GetKYCByUserID(ctx context.Context, userID uuid.UUID) (*domain.ProviderKYC, error)
	GetKYCByID(ctx context.Context, id uuid.UUID) (*domain.ProviderKYC, error)
	ListPendingKYC(ctx context.Context, limit, offset int) ([]domain.ProviderKYC, error)
	UpdateKYCStatus(ctx context.Context, kycID uuid.UUID, status domain.KYCStatus, rejectionReason string, reviewedBy *uuid.UUID) error

	AddDocument(ctx context.Context, doc *domain.ProviderDocument) error
	GetDocumentsByKYCID(ctx context.Context, kycID uuid.UUID) ([]domain.ProviderDocument, error)
}

type sqlKYCRepository struct {
	db *sql.DB
}

func NewKYCRepository(db *sql.DB) KYCRepository {
	return &sqlKYCRepository{db: db}
}

func (r *sqlKYCRepository) CreateOrUpdateKYC(ctx context.Context, kyc *domain.ProviderKYC) error {
	query := `
		INSERT INTO providers_kyc (
			id, user_id, company_name, gstin, gstin_verified, iec, iec_verified, pan, fleet_size, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id) DO UPDATE SET
			company_name = EXCLUDED.company_name,
			gstin = EXCLUDED.gstin,
			iec = EXCLUDED.iec,
			pan = EXCLUDED.pan,
			fleet_size = EXCLUDED.fleet_size,
			status = EXCLUDED.status,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if kyc.ID == uuid.Nil {
		kyc.ID = uuid.New()
	}
	now := time.Now()
	kyc.CreatedAt = now
	kyc.UpdatedAt = now

	return r.db.QueryRowContext(ctx, query,
		kyc.ID,
		kyc.UserID,
		kyc.CompanyName,
		kyc.GSTIN,
		kyc.GSTINVerified,
		kyc.IEC,
		kyc.IECVerified,
		kyc.PAN,
		kyc.FleetSize,
		kyc.Status,
		kyc.CreatedAt,
		kyc.UpdatedAt,
	).Scan(&kyc.ID, &kyc.CreatedAt, &kyc.UpdatedAt)
}

func (r *sqlKYCRepository) GetKYCByUserID(ctx context.Context, userID uuid.UUID) (*domain.ProviderKYC, error) {
	query := `
		SELECT id, user_id, company_name, gstin, gstin_verified, iec, iec_verified, pan, fleet_size, status, rejection_reason, reviewed_by, reviewed_at, created_at, updated_at
		FROM providers_kyc
		WHERE user_id = $1
	`
	k := &domain.ProviderKYC{}
	var iec, pan, rejectionReason sql.NullString
	var reviewedByStr sql.NullString
	var reviewedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&k.ID,
		&k.UserID,
		&k.CompanyName,
		&k.GSTIN,
		&k.GSTINVerified,
		&iec,
		&k.IECVerified,
		&pan,
		&k.FleetSize,
		&k.Status,
		&rejectionReason,
		&reviewedByStr,
		&reviewedAt,
		&k.CreatedAt,
		&k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if iec.Valid {
		k.IEC = iec.String
	}
	if pan.Valid {
		k.PAN = pan.String
	}
	if rejectionReason.Valid {
		k.RejectionReason = rejectionReason.String
	}
	if reviewedByStr.Valid {
		if id, err := uuid.Parse(reviewedByStr.String); err == nil {
			k.ReviewedBy = &id
		}
	}
	if reviewedAt.Valid {
		k.ReviewedAt = &reviewedAt.Time
	}

	docs, err := r.GetDocumentsByKYCID(ctx, k.ID)
	if err == nil {
		k.Documents = docs
	}

	return k, nil
}

func (r *sqlKYCRepository) GetKYCByID(ctx context.Context, id uuid.UUID) (*domain.ProviderKYC, error) {
	query := `
		SELECT id, user_id, company_name, gstin, gstin_verified, iec, iec_verified, pan, fleet_size, status, rejection_reason, reviewed_by, reviewed_at, created_at, updated_at
		FROM providers_kyc
		WHERE id = $1
	`
	k := &domain.ProviderKYC{}
	var iec, pan, rejectionReason sql.NullString
	var reviewedByStr sql.NullString
	var reviewedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&k.ID,
		&k.UserID,
		&k.CompanyName,
		&k.GSTIN,
		&k.GSTINVerified,
		&iec,
		&k.IECVerified,
		&pan,
		&k.FleetSize,
		&k.Status,
		&rejectionReason,
		&reviewedByStr,
		&reviewedAt,
		&k.CreatedAt,
		&k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if iec.Valid {
		k.IEC = iec.String
	}
	if pan.Valid {
		k.PAN = pan.String
	}
	if rejectionReason.Valid {
		k.RejectionReason = rejectionReason.String
	}
	if reviewedByStr.Valid {
		if id, err := uuid.Parse(reviewedByStr.String); err == nil {
			k.ReviewedBy = &id
		}
	}
	if reviewedAt.Valid {
		k.ReviewedAt = &reviewedAt.Time
	}

	docs, err := r.GetDocumentsByKYCID(ctx, k.ID)
	if err == nil {
		k.Documents = docs
	}

	return k, nil
}

func (r *sqlKYCRepository) ListPendingKYC(ctx context.Context, limit, offset int) ([]domain.ProviderKYC, error) {
	query := `
		SELECT id, user_id, company_name, gstin, gstin_verified, iec, iec_verified, pan, fleet_size, status, rejection_reason, reviewed_by, reviewed_at, created_at, updated_at
		FROM providers_kyc
		WHERE status = 'pending_review'
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProviderKYC
	for rows.Next() {
		var k domain.ProviderKYC
		var iec, pan, rejectionReason sql.NullString
		var reviewedByStr sql.NullString
		var reviewedAt sql.NullTime

		if err := rows.Scan(
			&k.ID,
			&k.UserID,
			&k.CompanyName,
			&k.GSTIN,
			&k.GSTINVerified,
			&iec,
			&k.IECVerified,
			&pan,
			&k.FleetSize,
			&k.Status,
			&rejectionReason,
			&reviewedByStr,
			&reviewedAt,
			&k.CreatedAt,
			&k.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if iec.Valid {
			k.IEC = iec.String
		}
		if pan.Valid {
			k.PAN = pan.String
		}
		if rejectionReason.Valid {
			k.RejectionReason = rejectionReason.String
		}

		docs, _ := r.GetDocumentsByKYCID(ctx, k.ID)
		k.Documents = docs
		result = append(result, k)
	}

	return result, nil
}

func (r *sqlKYCRepository) UpdateKYCStatus(ctx context.Context, kycID uuid.UUID, status domain.KYCStatus, rejectionReason string, reviewedBy *uuid.UUID) error {
	query := `
		UPDATE providers_kyc
		SET status = $2, rejection_reason = $3, reviewed_by = $4, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, kycID, status, rejectionReason, reviewedBy)
	return err
}

func (r *sqlKYCRepository) AddDocument(ctx context.Context, doc *domain.ProviderDocument) error {
	query := `
		INSERT INTO provider_documents (
			id, provider_kyc_id, doc_type, file_url, file_name, file_size_bytes, mime_type, latitude, longitude, captured_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	doc.CreatedAt = time.Now()

	return r.db.QueryRowContext(ctx, query,
		doc.ID,
		doc.ProviderKYCID,
		doc.DocType,
		doc.FileURL,
		doc.FileName,
		doc.FileSizeBytes,
		doc.MimeType,
		doc.Latitude,
		doc.Longitude,
		doc.CapturedAt,
		doc.CreatedAt,
	).Scan(&doc.ID, &doc.CreatedAt)
}

func (r *sqlKYCRepository) GetDocumentsByKYCID(ctx context.Context, kycID uuid.UUID) ([]domain.ProviderDocument, error) {
	query := `
		SELECT id, provider_kyc_id, doc_type, file_url, file_name, file_size_bytes, mime_type, latitude, longitude, captured_at, created_at
		FROM provider_documents
		WHERE provider_kyc_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, kycID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []domain.ProviderDocument
	for rows.Next() {
		var doc domain.ProviderDocument
		var lat, lng sql.NullFloat64
		var capAt sql.NullTime

		if err := rows.Scan(
			&doc.ID,
			&doc.ProviderKYCID,
			&doc.DocType,
			&doc.FileURL,
			&doc.FileName,
			&doc.FileSizeBytes,
			&doc.MimeType,
			&lat,
			&lng,
			&capAt,
			&doc.CreatedAt,
		); err != nil {
			return nil, err
		}

		if lat.Valid {
			doc.Latitude = &lat.Float64
		}
		if lng.Valid {
			doc.Longitude = &lng.Float64
		}
		if capAt.Valid {
			doc.CapturedAt = &capAt.Time
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

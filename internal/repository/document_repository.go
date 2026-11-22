package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type DocumentVaultItem struct {
	ID            uuid.UUID `json:"id"`
	BookingID     uuid.UUID `json:"booking_id"`
	DocType       string    `json:"doc_type"`
	FileURL       string    `json:"file_url"`
	FileName      string    `json:"file_name"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	MimeType      string    `json:"mime_type"`
	UploadedBy    uuid.UUID `json:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type DocumentRepository interface {
	AddDocument(ctx context.Context, doc *DocumentVaultItem) error
	ListDocumentsByBooking(ctx context.Context, bookingID uuid.UUID) ([]DocumentVaultItem, error)
}

type sqlDocumentRepository struct {
	db *sql.DB
}

func NewDocumentRepository(db *sql.DB) DocumentRepository {
	return &sqlDocumentRepository{db: db}
}

func (r *sqlDocumentRepository) AddDocument(ctx context.Context, doc *DocumentVaultItem) error {
	query := `
		INSERT INTO booking_documents (id, booking_id, doc_type, file_url, file_name, file_size_bytes, mime_type, uploaded_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, created_at
	`
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	return r.db.QueryRowContext(ctx, query,
		doc.ID, doc.BookingID, doc.DocType, doc.FileURL, doc.FileName, doc.FileSizeBytes, doc.MimeType, doc.UploadedBy,
	).Scan(&doc.ID, &doc.CreatedAt)
}

func (r *sqlDocumentRepository) ListDocumentsByBooking(ctx context.Context, bookingID uuid.UUID) ([]DocumentVaultItem, error) {
	query := `
		SELECT id, booking_id, doc_type, file_url, file_name, file_size_bytes, mime_type, uploaded_by, created_at
		FROM booking_documents
		WHERE booking_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []DocumentVaultItem
	for rows.Next() {
		var d DocumentVaultItem
		if err := rows.Scan(
			&d.ID, &d.BookingID, &d.DocType, &d.FileURL, &d.FileName, &d.FileSizeBytes, &d.MimeType, &d.UploadedBy, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	return docs, nil
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

type KYCStatus string

const (
	KYCStatusDraft         KYCStatus = "draft"
	KYCStatusPendingReview KYCStatus = "pending_review"
	KYCStatusApproved      KYCStatus = "approved"
	KYCStatusRejected      KYCStatus = "rejected"
)

type ProviderKYC struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"user_id"`
	CompanyName     string             `json:"company_name"`
	GSTIN           string             `json:"gstin"`
	GSTINVerified   bool               `json:"gstin_verified"`
	IEC             string             `json:"iec,omitempty"`
	IECVerified     bool               `json:"iec_verified"`
	PAN             string             `json:"pan,omitempty"`
	FleetSize       int                `json:"fleet_size"`
	Status          KYCStatus          `json:"status"`
	RejectionReason string             `json:"rejection_reason,omitempty"`
	ReviewedBy      *uuid.UUID         `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time         `json:"reviewed_at,omitempty"`
	Documents       []ProviderDocument `json:"documents,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type DocumentType string

const (
	DocTransportPermit DocumentType = "transport_permit"
	DocFleetRC         DocumentType = "fleet_rc"
	DocContainerFront  DocumentType = "container_front"
	DocContainerSide   DocumentType = "container_side"
	DocContainerDoors  DocumentType = "container_doors"
	DocGSTCertificate  DocumentType = "gst_certificate"
	DocIECCertificate  DocumentType = "iec_certificate"
	DocOther           DocumentType = "other"
)

type ProviderDocument struct {
	ID            uuid.UUID    `json:"id"`
	ProviderKYCID uuid.UUID    `json:"provider_kyc_id"`
	DocType       DocumentType `json:"doc_type"`
	FileURL       string       `json:"file_url"`
	FileName      string       `json:"file_name"`
	FileSizeBytes int64        `json:"file_size_bytes"`
	MimeType      string       `json:"mime_type"`
	Latitude      *float64     `json:"latitude,omitempty"`
	Longitude     *float64     `json:"longitude,omitempty"`
	CapturedAt    *time.Time   `json:"captured_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

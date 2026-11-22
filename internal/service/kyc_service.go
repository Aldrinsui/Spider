package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

var (
	gstinRegex = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)
	iecRegex   = regexp.MustCompile(`^[A-Z0-9]{10}$`)
	panRegex   = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`)
)

type KYCService interface {
	SubmitProfile(ctx context.Context, userID uuid.UUID, companyName, gstin, iec, pan string, fleetSize int) (*domain.ProviderKYC, error)
	UploadDocument(ctx context.Context, userID uuid.UUID, docType domain.DocumentType, fileURL, fileName string, fileSizeBytes int64, mimeType string, lat, lng *float64) (*domain.ProviderDocument, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.ProviderKYC, error)
	GetProfileByID(ctx context.Context, kycID uuid.UUID) (*domain.ProviderKYC, error)
	ListPendingReviews(ctx context.Context, limit, offset int) ([]domain.ProviderKYC, error)
	ReviewKYC(ctx context.Context, adminID uuid.UUID, kycID uuid.UUID, approved bool, rejectionReason string) error
}

type kycService struct {
	kycRepo   repository.KYCRepository
	userRepo  repository.UserRepository
	auditRepo repository.AuditRepository
}

func NewKYCService(kycRepo repository.KYCRepository, userRepo repository.UserRepository, auditRepo repository.AuditRepository) KYCService {
	return &kycService{kycRepo: kycRepo, userRepo: userRepo, auditRepo: auditRepo}
}

func (s *kycService) SubmitProfile(ctx context.Context, userID uuid.UUID, companyName, gstin, iec, pan string, fleetSize int) (*domain.ProviderKYC, error) {
	companyName = strings.TrimSpace(companyName)
	if companyName == "" {
		return nil, errors.New("company name is required")
	}

	gstin = strings.ToUpper(strings.TrimSpace(gstin))
	if !gstinRegex.MatchString(gstin) {
		return nil, errors.New("invalid Indian GSTIN format (must be 15 characters, e.g. 33AAAAA0000A1Z5)")
	}

	iec = strings.ToUpper(strings.TrimSpace(iec))
	if iec != "" && !iecRegex.MatchString(iec) {
		return nil, errors.New("invalid IEC format (must be 10 characters)")
	}

	pan = strings.ToUpper(strings.TrimSpace(pan))
	if pan != "" && !panRegex.MatchString(pan) {
		return nil, errors.New("invalid PAN format (must be 10 characters, e.g. ABCDE1234F)")
	}

	if fleetSize < 1 {
		fleetSize = 1
	}

	kyc := &domain.ProviderKYC{
		UserID:        userID,
		CompanyName:   companyName,
		GSTIN:         gstin,
		GSTINVerified: true, // In production, verified against GSTN API
		IEC:           iec,
		IECVerified:   iec != "",
		PAN:           pan,
		FleetSize:     fleetSize,
		Status:        domain.KYCStatusPendingReview,
	}

	if err := s.kycRepo.CreateOrUpdateKYC(ctx, kyc); err != nil {
		return nil, fmt.Errorf("failed to save provider KYC: %w", err)
	}

	return kyc, nil
}

func (s *kycService) UploadDocument(ctx context.Context, userID uuid.UUID, docType domain.DocumentType, fileURL, fileName string, fileSizeBytes int64, mimeType string, lat, lng *float64) (*domain.ProviderDocument, error) {
	kyc, err := s.kycRepo.GetKYCByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch KYC profile: %w", err)
	}
	if kyc == nil {
		return nil, errors.New("provider profile not found. Please submit company details first.")
	}

	// Geotag enforcement for container photos as per Indian freight security requirements
	if docType == domain.DocContainerFront || docType == domain.DocContainerSide || docType == domain.DocContainerDoors {
		if lat == nil || lng == nil {
			return nil, errors.New("geotag (latitude and longitude) is strictly required for container inspection photos")
		}
	}

	now := time.Now()
	doc := &domain.ProviderDocument{
		ProviderKYCID: kyc.ID,
		DocType:       docType,
		FileURL:       fileURL,
		FileName:      fileName,
		FileSizeBytes: fileSizeBytes,
		MimeType:      mimeType,
		Latitude:      lat,
		Longitude:     lng,
		CapturedAt:    &now,
	}

	if err := s.kycRepo.AddDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to save document record: %w", err)
	}

	return doc, nil
}

func (s *kycService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.ProviderKYC, error) {
	return s.kycRepo.GetKYCByUserID(ctx, userID)
}

func (s *kycService) GetProfileByID(ctx context.Context, kycID uuid.UUID) (*domain.ProviderKYC, error) {
	return s.kycRepo.GetKYCByID(ctx, kycID)
}

func (s *kycService) ListPendingReviews(ctx context.Context, limit, offset int) ([]domain.ProviderKYC, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.kycRepo.ListPendingKYC(ctx, limit, offset)
}

func (s *kycService) ReviewKYC(ctx context.Context, adminID uuid.UUID, kycID uuid.UUID, approved bool, rejectionReason string) error {
	kyc, err := s.kycRepo.GetKYCByID(ctx, kycID)
	if err != nil {
		return fmt.Errorf("failed to fetch KYC: %w", err)
	}
	if kyc == nil {
		return errors.New("KYC profile not found")
	}

	newStatus := domain.KYCStatusApproved
	if !approved {
		newStatus = domain.KYCStatusRejected
		if strings.TrimSpace(rejectionReason) == "" {
			return errors.New("rejection reason is required when rejecting a provider")
		}
	}

	if err := s.kycRepo.UpdateKYCStatus(ctx, kycID, newStatus, rejectionReason, &adminID); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Update user role if approved
	if approved {
		user, err := s.userRepo.GetUserByID(ctx, kyc.UserID)
		if err == nil && user != nil {
			user.Role = domain.RoleProvider
			_ = s.userRepo.UpdateUser(ctx, user)
		}
	}

	return nil
}

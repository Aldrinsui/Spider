package test

import (
	"context"
	"testing"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/google/uuid"
)

func TestKYCService_GSTINValidationAndGeotagging(t *testing.T) {
	kycRepo := NewMockKYCRepository()
	userRepo := NewMockUserRepository()
	auditRepo := NewMockAuditRepository()
	kycSvc := service.NewKYCService(kycRepo, userRepo, auditRepo)
	ctx := context.Background()

	providerID := uuid.New()
	user := &domain.User{
		ID:       providerID,
		Phone:    "+919876543211",
		Role:     domain.RoleTrader,
		IsActive: true,
	}
	_ = userRepo.CreateUser(ctx, user)

	// 1. Submit with invalid GSTIN (fails)
	_, err := kycSvc.SubmitProfile(ctx, providerID, "Test Logistics", "INVALID_GST", "IEC1234567", "ABCDE1234F", 5)
	if err == nil {
		t.Fatalf("expected error for invalid GSTIN, got nil")
	}

	// 2. Submit with valid Indian GSTIN (passes: 33AAAAA0000A1Z5)
	validGSTIN := "33AAAAA0000A1Z5"
	kyc, err := kycSvc.SubmitProfile(ctx, providerID, "Rail Logistics Co.", validGSTIN, "IEC1234567", "AAAAA0000A", 10)
	if err != nil {
		t.Fatalf("expected success for valid GSTIN, got: %v", err)
	}
	if kyc.Status != domain.KYCStatusPendingReview {
		t.Fatalf("expected status %s, got %s", domain.KYCStatusPendingReview, kyc.Status)
	}

	// 3. Upload container photo without geotag (fails)
	_, err = kycSvc.UploadDocument(ctx, providerID, domain.DocContainerFront, "https://cdn.spider.io/front.jpg", "front.jpg", 102400, "image/jpeg", nil, nil)
	if err == nil {
		t.Fatalf("expected error when container photo lacks geotag, got nil")
	}

	// 4. Upload container photo with geotag (passes)
	lat := 13.0827
	lng := 80.2707
	doc, err := kycSvc.UploadDocument(ctx, providerID, domain.DocContainerFront, "https://cdn.spider.io/front.jpg", "front.jpg", 102400, "image/jpeg", &lat, &lng)
	if err != nil {
		t.Fatalf("expected success uploading geotagged container photo, got: %v", err)
	}
	if doc.Latitude == nil || *doc.Latitude != lat {
		t.Fatalf("expected latitude %f, got %v", lat, doc.Latitude)
	}

	// 5. Admin reviews and approves KYC
	adminID := uuid.New()
	err = kycSvc.ReviewKYC(ctx, adminID, kyc.ID, true, "")
	if err != nil {
		t.Fatalf("expected success approving KYC, got: %v", err)
	}

	updatedKYC, _ := kycSvc.GetProfileByID(ctx, kyc.ID)
	if updatedKYC.Status != domain.KYCStatusApproved {
		t.Fatalf("expected status %s, got %s", domain.KYCStatusApproved, updatedKYC.Status)
	}

	// Provider user's role promoted to RoleProvider
	updatedUser, _ := userRepo.GetUserByID(ctx, providerID)
	if updatedUser.Role != domain.RoleProvider {
		t.Fatalf("expected user role to be promoted to provider, got %s", updatedUser.Role)
	}
}

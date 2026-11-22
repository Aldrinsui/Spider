package test

import (
	"context"
	"testing"

	"github.com/Aldrinsui/Spider/internal/config"
	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/service"
)

func TestAuthService_SendAndVerifyOTP(t *testing.T) {
	userRepo := NewMockUserRepository()
	cfg := &config.Config{
		JWTSecret: "test_jwt_secret_key_1234567890",
	}
	authSvc := service.NewAuthService(userRepo, cfg)
	ctx := context.Background()

	phone := "+919876543210"

	// 1. Send OTP
	code, err := authSvc.SendOTP(ctx, phone)
	if err != nil {
		t.Fatalf("expected nil error on SendOTP, got: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit OTP code, got: %s", code)
	}

	// 2. Verify with wrong OTP
	_, err = authSvc.VerifyOTP(ctx, phone, "000000", domain.RoleTrader, true, "v1.0")
	if err == nil {
		t.Fatalf("expected error on invalid OTP, got nil")
	}

	// 3. Verify with correct OTP
	authResp, err := authSvc.VerifyOTP(ctx, phone, code, domain.RoleTrader, true, "v1.0")
	if err != nil {
		t.Fatalf("expected success verifying OTP, got error: %v", err)
	}
	if authResp.Token == "" {
		t.Fatalf("expected non-empty JWT token")
	}
	if authResp.User.Phone != phone {
		t.Fatalf("expected user phone %s, got %s", phone, authResp.User.Phone)
	}
	if authResp.User.DPDPConsentAt == nil {
		t.Fatalf("expected DPDP consent timestamp to be recorded")
	}
	if authResp.User.DPDPConsentVersion != "v1.0" {
		t.Fatalf("expected DPDP consent version v1.0, got %s", authResp.User.DPDPConsentVersion)
	}

	// 4. Repeated OTP verify fails because OTP was already verified/consumed
	_, err = authSvc.VerifyOTP(ctx, phone, code, domain.RoleTrader, true, "v1.0")
	if err == nil {
		t.Fatalf("expected failure verifying already consumed OTP, got nil")
	}
}

func TestAuthService_InvalidPhoneFormat(t *testing.T) {
	userRepo := NewMockUserRepository()
	cfg := &config.Config{JWTSecret: "test_jwt_secret"}
	authSvc := service.NewAuthService(userRepo, cfg)
	ctx := context.Background()

	invalidPhones := []string{
		"9876543210",       // missing leading +
		"+0123456789",      // invalid country code
		"abcdefghij",       // non-numeric
		"+91-98765-43210",  // dashes not permitted in E.164
	}

	for _, p := range invalidPhones {
		_, err := authSvc.SendOTP(ctx, p)
		if err == nil {
			t.Errorf("expected error for invalid phone '%s', got nil", p)
		}
	}
}

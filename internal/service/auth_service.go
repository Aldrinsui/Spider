package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/Aldrinsui/Spider/internal/config"
	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{9,14}$`)

type AuthService interface {
	SendOTP(ctx context.Context, phone string) (string, error)
	VerifyOTP(ctx context.Context, phone, otp string, role domain.UserRole, dpdpConsent bool, dpdpVersion string) (*domain.AuthResponse, error)
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{userRepo: userRepo, cfg: cfg}
}

func (s *authService) SendOTP(ctx context.Context, phone string) (string, error) {
	if !phoneRegex.MatchString(phone) {
		return "", errors.New("invalid phone number format. Must be in E.164 international format (e.g. +919876543210)")
	}

	// Generate 6-digit cryptographic OTP
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("failed to generate random OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	// Hash OTP with bcrypt for secure storage
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash OTP: %w", err)
	}

	otpRecord := &domain.OTPVerification{
		Phone:       phone,
		OTPHash:     string(hash),
		Attempts:    0,
		MaxAttempts: 3,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	if err := s.userRepo.CreateOTP(ctx, otpRecord); err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	// For staging/development, return code directly. In production, send via SMS gateway.
	return code, nil
}

func (s *authService) VerifyOTP(ctx context.Context, phone, otp string, role domain.UserRole, dpdpConsent bool, dpdpVersion string) (*domain.AuthResponse, error) {
	if !phoneRegex.MatchString(phone) {
		return nil, errors.New("invalid phone number format")
	}

	if role == "" {
		role = domain.RoleTrader
	}
	if role != domain.RoleTrader && role != domain.RoleProvider && role != domain.RoleAdmin {
		return nil, errors.New("invalid role specified")
	}

	activeOTP, err := s.userRepo.GetLatestActiveOTP(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if activeOTP == nil {
		return nil, errors.New("no active OTP found or OTP has expired. Please request a new OTP.")
	}

	if activeOTP.Attempts >= activeOTP.MaxAttempts {
		return nil, errors.New("maximum OTP attempts exceeded. Please request a new OTP.")
	}

	// Verify hash
	if err := bcrypt.CompareHashAndPassword([]byte(activeOTP.OTPHash), []byte(otp)); err != nil {
		_ = s.userRepo.IncrementOTPAttempts(ctx, activeOTP.ID)
		return nil, errors.New("incorrect OTP")
	}

	_ = s.userRepo.MarkOTPVerified(ctx, activeOTP.ID)

	// Fetch or create user
	user, err := s.userRepo.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	isNewUser := false
	now := time.Now()

	if user == nil {
		isNewUser = true
		user = &domain.User{
			Phone:    phone,
			Role:     role,
			IsActive: true,
		}
		if dpdpConsent {
			user.DPDPConsentAt = &now
			user.DPDPConsentVersion = dpdpVersion
			if user.DPDPConsentVersion == "" {
				user.DPDPConsentVersion = "v1.0"
			}
		}

		if err := s.userRepo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to register new user: %w", err)
		}
	} else {
		if !user.IsActive {
			return nil, errors.New("user account is deactivated")
		}
		if dpdpConsent && user.DPDPConsentAt == nil {
			ver := dpdpVersion
			if ver == "" {
				ver = "v1.0"
			}
			_ = s.userRepo.RecordDPDPConsent(ctx, user.ID, ver)
			user.DPDPConsentAt = &now
			user.DPDPConsentVersion = ver
		}
	}

	// Issue JWT token (7-day duration for app sessions)
	token, err := middleware.GenerateToken(user.ID, user.Phone, user.Role, s.cfg.JWTSecret, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken := uuid.New().String()

	return &domain.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *user,
		IsNewUser:    isNewUser,
	}, nil
}

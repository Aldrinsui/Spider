package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleTrader   UserRole = "trader"
	RoleProvider UserRole = "provider"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	ID                 uuid.UUID  `json:"id"`
	Phone              string     `json:"phone"`
	Role               UserRole   `json:"role"`
	FullName           string     `json:"full_name"`
	Email              string     `json:"email"`
	IsActive           bool       `json:"is_active"`
	DPDPConsentAt      *time.Time `json:"dpdp_consent_at"`
	DPDPConsentVersion string     `json:"dpdp_consent_version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type OTPVerification struct {
	ID          uuid.UUID  `json:"id"`
	Phone       string     `json:"phone"`
	OTPHash     string     `json:"-"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	ExpiresAt   time.Time  `json:"expires_at"`
	VerifiedAt  *time.Time `json:"verified_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
	IsNewUser    bool   `json:"is_new_user"`
}

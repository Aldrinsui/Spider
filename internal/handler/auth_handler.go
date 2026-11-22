package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/Aldrinsui/Spider/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
	userRepo    repository.UserRepository
}

func NewAuthHandler(authService service.AuthService, userRepo repository.UserRepository) *AuthHandler {
	return &AuthHandler{authService: authService, userRepo: userRepo}
}

type SendOTPRequest struct {
	Phone string `json:"phone"`
}

type SendOTPResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"` // Included in dev mode
}

func (h *AuthHandler) HandleSendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	code, err := h.authService.SendOTP(r.Context(), req.Phone)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, SendOTPResponse{
		Message: "OTP sent successfully via SMS",
		Code:    code, // Returned in dev mode for easy testing
	})
}

type VerifyOTPRequest struct {
	Phone       string          `json:"phone"`
	OTP         string          `json:"otp"`
	Role        domain.UserRole `json:"role"`
	DPDPConsent bool            `json:"dpdp_consent"`
	DPDPVersion string          `json:"dpdp_version"`
}

func (h *AuthHandler) HandleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.authService.VerifyOTP(r.Context(), req.Phone, req.OTP, req.Role, req.DPDPConsent, req.DPDPVersion)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) HandleGetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	kycService   service.KYCService
	auditService service.AuditService
}

func NewAdminHandler(kycService service.KYCService, auditService service.AuditService) *AdminHandler {
	return &AdminHandler{kycService: kycService, auditService: auditService}
}

func (h *AdminHandler) HandleListPendingKYC(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	providers, err := h.kycService.ListPendingReviews(r.Context(), limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
		"count":     len(providers),
	})
}

func (h *AdminHandler) HandleGetKYCDetails(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	kycID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid KYC UUID"})
		return
	}

	kyc, err := h.kycService.GetProfileByID(r.Context(), kycID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if kyc == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "KYC profile not found"})
		return
	}

	respondJSON(w, http.StatusOK, kyc)
}

type ReviewKYCRequest struct {
	Approved        bool   `json:"approved"`
	RejectionReason string `json:"rejection_reason"`
}

func (h *AdminHandler) HandleReviewKYC(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	kycID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid KYC UUID"})
		return
	}

	var req ReviewKYCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.kycService.ReviewKYC(r.Context(), claims.UserID, kycID, req.Approved, req.RejectionReason); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Record immutable audit log
	action := "provider.approved"
	if !req.Approved {
		action = "provider.rejected"
	}
	_ = h.auditService.Log(
		r.Context(),
		&claims.UserID,
		action,
		"provider_kyc",
		kycID,
		map[string]interface{}{
			"approved":         req.Approved,
			"rejection_reason": req.RejectionReason,
		},
		r.RemoteAddr,
		r.UserAgent(),
	)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "KYC decision processed and audit log recorded",
	})
}

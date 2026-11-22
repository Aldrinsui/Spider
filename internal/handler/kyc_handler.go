package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/service"
)

type KYCHandler struct {
	kycService service.KYCService
}

func NewKYCHandler(kycService service.KYCService) *KYCHandler {
	return &KYCHandler{kycService: kycService}
}

type SubmitKYCRequest struct {
	CompanyName string `json:"company_name"`
	GSTIN       string `json:"gstin"`
	IEC         string `json:"iec"`
	PAN         string `json:"pan"`
	FleetSize   int    `json:"fleet_size"`
}

func (h *KYCHandler) HandleSubmitKYC(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req SubmitKYCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	kyc, err := h.kycService.SubmitProfile(r.Context(), claims.UserID, req.CompanyName, req.GSTIN, req.IEC, req.PAN, req.FleetSize)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, kyc)
}

type UploadDocRequest struct {
	DocType       domain.DocumentType `json:"doc_type"`
	FileURL       string              `json:"file_url"`
	FileName      string              `json:"file_name"`
	FileSizeBytes int64               `json:"file_size_bytes"`
	MimeType      string              `json:"mime_type"`
	Latitude      *float64            `json:"latitude,omitempty"`
	Longitude     *float64            `json:"longitude,omitempty"`
}

func (h *KYCHandler) HandleUploadDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req UploadDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	doc, err := h.kycService.UploadDocument(
		r.Context(),
		claims.UserID,
		req.DocType,
		req.FileURL,
		req.FileName,
		req.FileSizeBytes,
		req.MimeType,
		req.Latitude,
		req.Longitude,
	)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

func (h *KYCHandler) HandleGetMyKYC(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	kyc, err := h.kycService.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if kyc == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"message": "no KYC profile submitted yet"})
		return
	}

	respondJSON(w, http.StatusOK, kyc)
}

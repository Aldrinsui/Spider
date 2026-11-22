package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TrackingHandler struct {
	trackingService service.TrackingService
}

func NewTrackingHandler(trackingService service.TrackingService) *TrackingHandler {
	return &TrackingHandler{trackingService: trackingService}
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (h *TrackingHandler) HandleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	containerID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid container UUID"})
		return
	}

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.trackingService.UpdateLocation(r.Context(), claims.UserID, containerID, req.Latitude, req.Longitude); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "location updated"})
}

type SubmitPODRequest struct {
	PODPhotoURL           string   `json:"pod_photo_url"`
	RecipientName         string   `json:"recipient_name"`
	RecipientPhone        string   `json:"recipient_phone"`
	RecipientSignatureURL string   `json:"recipient_signature_url,omitempty"`
	Latitude              *float64 `json:"latitude,omitempty"`
	Longitude             *float64 `json:"longitude,omitempty"`
}

func (h *TrackingHandler) HandleSubmitPOD(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	bookingID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking UUID"})
		return
	}

	var req SubmitPODRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	proof, err := h.trackingService.SubmitProofOfDelivery(
		r.Context(), claims.UserID, bookingID, req.PODPhotoURL,
		req.RecipientName, req.RecipientPhone, req.RecipientSignatureURL,
		req.Latitude, req.Longitude,
	)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, proof)
}

func (h *TrackingHandler) HandleGetPOD(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	bookingID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking UUID"})
		return
	}

	proof, err := h.trackingService.GetDeliveryProof(r.Context(), bookingID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if proof == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "delivery proof not submitted yet"})
		return
	}

	respondJSON(w, http.StatusOK, proof)
}

type AddDocRequest struct {
	DocType       string `json:"doc_type"`
	FileURL       string `json:"file_url"`
	FileName      string `json:"file_name"`
	FileSizeBytes int64  `json:"file_size_bytes"`
	MimeType      string `json:"mime_type"`
}

func (h *TrackingHandler) HandleAddDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	bookingID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking UUID"})
		return
	}

	var req AddDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	doc, err := h.trackingService.UploadDocument(
		r.Context(), claims.UserID, bookingID,
		req.DocType, req.FileURL, req.FileName, req.FileSizeBytes, req.MimeType,
	)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

func (h *TrackingHandler) HandleListDocuments(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	bookingID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking UUID"})
		return
	}

	docs, err := h.trackingService.ListDocuments(r.Context(), claims.UserID, bookingID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"documents": docs,
		"count":     len(docs),
	})
}

type RaiseDisputeRequest struct {
	Reason       string   `json:"reason"`
	Description  string   `json:"description"`
	EvidenceURLs []string `json:"evidence_urls"`
}

func (h *TrackingHandler) HandleRaiseDispute(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	bookingID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking UUID"})
		return
	}

	var req RaiseDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	dispute, err := h.trackingService.RaiseDispute(
		r.Context(), claims.UserID, bookingID, req.Reason, req.Description, req.EvidenceURLs,
	)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, dispute)
}

type ResolveDisputeRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

func (h *TrackingHandler) HandleResolveDispute(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idParam := chi.URLParam(r, "id")
	disputeID, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid dispute UUID"})
		return
	}

	var req ResolveDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.trackingService.ResolveDispute(r.Context(), claims.UserID, disputeID, req.Status, req.Notes); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "dispute resolved successfully"})
}

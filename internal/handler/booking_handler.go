package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type BookingHandler struct {
	bookingService service.BookingService
}

func NewBookingHandler(bookingService service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

func (h *BookingHandler) HandleHoldSpace(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req service.HoldSpaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hold request body"})
		return
	}

	booking, err := h.bookingService.HoldSpace(r.Context(), claims.UserID, req)
	if err != nil {
		respondJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, booking)
}

func (h *BookingHandler) HandleGetBooking(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking UUID"})
		return
	}

	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if booking == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "booking not found"})
		return
	}

	respondJSON(w, http.StatusOK, booking)
}

func (h *BookingHandler) HandleListMyBookings(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bookings, err := h.bookingService.ListTraderBookings(r.Context(), claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"bookings": bookings,
		"count":    len(bookings),
	})
}

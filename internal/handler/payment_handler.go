package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentService service.PaymentService
}

func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

type CreateOrderRequest struct {
	BookingID uuid.UUID `json:"booking_id"`
}

func (h *PaymentHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order request body"})
		return
	}

	escrow, err := h.paymentService.CreatePaymentOrder(r.Context(), claims.UserID, req.BookingID)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, escrow)
}

func (h *PaymentHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Razorpay-Signature")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read webhook body"})
		return
	}

	if err := h.paymentService.ProcessWebhook(r.Context(), signature, body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type CancelBookingRequest struct {
	Reason string `json:"reason"`
}

func (h *PaymentHandler) HandleCancelBooking(w http.ResponseWriter, r *http.Request) {
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

	var req CancelBookingRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	refundAmountPaise, err := h.paymentService.CancelBookingAndRefund(r.Context(), claims.UserID, bookingID, req.Reason)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":             "booking cancelled successfully",
		"refund_amount_paise": refundAmountPaise,
		"refund_currency":     "INR",
	})
}

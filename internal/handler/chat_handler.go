package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService service.ChatService
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

type SendChatMessageRequest struct {
	Content       string `json:"content"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}

func (h *ChatHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
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

	var req SendChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	msg, err := h.chatService.SendMessage(r.Context(), claims.UserID, bookingID, req.Content, req.AttachmentURL)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, msg)
}

func (h *ChatHandler) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
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

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	var beforeTime *time.Time
	if bStr := r.URL.Query().Get("before"); bStr != "" {
		if t, err := time.Parse(time.RFC3339, bStr); err == nil {
			beforeTime = &t
		}
	}

	messages, err := h.chatService.GetMessages(r.Context(), claims.UserID, bookingID, limit, beforeTime)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
}

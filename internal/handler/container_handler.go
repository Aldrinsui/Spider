package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/middleware"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/Aldrinsui/Spider/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ContainerHandler struct {
	containerService service.ContainerService
}

func NewContainerHandler(containerService service.ContainerService) *ContainerHandler {
	return &ContainerHandler{containerService: containerService}
}

func (h *ContainerHandler) HandleCreateContainer(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var c domain.Container
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	res, err := h.containerService.CreateContainer(r.Context(), claims.UserID, c)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, res)
}

func (h *ContainerHandler) HandleGetContainer(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid container UUID"})
		return
	}

	c, err := h.containerService.GetContainer(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if c == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "container not found"})
		return
	}

	respondJSON(w, http.StatusOK, c)
}

func (h *ContainerHandler) HandleSearchContainers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := repository.ContainerSearchFilter{
		TransportMode: q.Get("mode"),
		SortBy:        q.Get("sort"),
	}

	if rkm, err := strconv.ParseFloat(q.Get("radius_km"), 64); err == nil {
		filter.RadiusKM = rkm
	}
	if cbm, err := strconv.ParseFloat(q.Get("min_cbm"), 64); err == nil {
		filter.MinCBM = cbm
	}
	if kg, err := strconv.ParseFloat(q.Get("min_kg"), 64); err == nil {
		filter.MinWeightKG = kg
	}

	if olat, err := strconv.ParseFloat(q.Get("origin_lat"), 64); err == nil {
		filter.OriginLat = &olat
	}
	if olng, err := strconv.ParseFloat(q.Get("origin_lng"), 64); err == nil {
		filter.OriginLng = &olng
	}
	if dlat, err := strconv.ParseFloat(q.Get("dest_lat"), 64); err == nil {
		filter.DestLat = &dlat
	}
	if dlng, err := strconv.ParseFloat(q.Get("dest_lng"), 64); err == nil {
		filter.DestLng = &dlng
	}

	if l, err := strconv.Atoi(q.Get("limit")); err == nil {
		filter.Limit = l
	}
	if o, err := strconv.Atoi(q.Get("offset")); err == nil {
		filter.Offset = o
	}

	containers, err := h.containerService.SearchContainers(r.Context(), filter)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"routes": containers,
		"count":  len(containers),
	})
}

func (h *ContainerHandler) HandleListMyContainers(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	containers, err := h.containerService.ListProviderContainers(r.Context(), claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"containers": containers,
		"count":      len(containers),
	})
}

func (h *ContainerHandler) HandleCalculateQuote(w http.ResponseWriter, r *http.Request) {
	var req service.QuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid quote request body"})
		return
	}

	quote, err := h.containerService.CalculateQuote(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, quote)
}

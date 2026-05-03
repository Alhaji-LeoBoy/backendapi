package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"femProjectSqlc/internal/cache"
	"femProjectSqlc/internal/db"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/middleware"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/utils"

	"github.com/go-chi/chi/v5"
)

type EventHandler struct {
	eventStore store.EventStoreInterface
	logger     *logger.Logger
	cache      *cache.InMemoryCache
}

// Constructor
func NewEventHandler(eventStore store.EventStoreInterface, log *logger.Logger, cache *cache.InMemoryCache) *EventHandler {
	return &EventHandler{
		eventStore: eventStore,
		logger:     log,
		cache:      cache,
	}
}

type CreateEventRequest struct {
	Title       string  `json:"title"`
	OwnerName   string  `json:"owner_name"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	StartTime   string  `json:"start_time"`
	ImageURL    string  `json:"image_url"`
	Price       float64 `json:"price"`
}

type PatchEventRequest struct {
	Title       string  `json:"title,omitempty"`
	OwnerName   string  `json:"owner_name,omitempty"`
	Description string  `json:"description,omitempty"`
	Location    string  `json:"location,omitempty"`
	StartTime   string  `json:"start_time,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	Price       float64 `json:"price,omitempty"`
}

// ==================== Event Handlers ==================== //

// ListEvents returns paginated events with caching
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := utils.ParsePagination(r)

	// Generate a unique cache key for this page
	key := fmt.Sprintf("events:limit=%d:offset=%d", p.Limit, p.Offset)

	// Try to return cached response first
	cached, err := h.cache.Get(ctx, key)
	if err == nil && cached != "" {
		var response any
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.logger.Info("Returning cached events") // cache hit
			utils.RespondWithJSON(w, http.StatusOK, response)
			return
		}
		// If unmarshal fails, continue to fetch fresh data
	}

	// Cache miss → fetch events from DB
	// Fetch events from DB
	h.logger.Info("Fetching events from DB")
	events, err := h.eventStore.ListEvents(ctx, db.ListEventsParams{
		Limit:  p.Limit,
		Offset: p.Offset,
	})
	if err != nil {
		h.logger.Error("Failed to list events", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	// Get total count for pagination
	total, err := h.eventStore.CountEvents(ctx)
	if err != nil {
		h.logger.Error("Failed to count events", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	p.Total = total

	// Build response
	response := utils.PaginatedResponse("events", events, p)

	// Cache the response for 60 seconds
	data, _ := json.Marshal(response)
	if err := h.cache.Set(ctx, key, data, 60*time.Second); err != nil {
		h.logger.Error("Failed to cache events", err)
	}

	// Return fresh response
	utils.RespondWithJSON(w, http.StatusOK, response)
}

// CreateEvent creates a new event and invalidates cache
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateEventRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" || req.Location == "" || req.StartTime == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "title, location, and start_time are required")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid start_time format")
		return
	}

	ownerName := req.OwnerName
	if ownerName == "" {
		ownerName = authUser.Username
	}

	event, err := h.eventStore.CreateEvent(r.Context(), db.CreateEventParams{
		UserID:      authUser.ID,
		Title:       req.Title,
		OwnerName:   ownerName,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   startTime,
		ImageUrl:    req.ImageURL,
		Price:       req.Price,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	// Invalidate cached event lists after creation
	if err := h.cache.DeleteByPattern(r.Context(), "events:*"); err != nil {
		h.logger.Error("Failed to invalidate event cache", err)
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]any{"event": event})
}

// PatchEvent updates an event and invalidates cache
func (h *EventHandler) PatchEvent(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	var req PatchEventRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// If admin, we fetch the event to get the actual owner's ID
	// so the PatchEvent query (which requires user_id) still works,
	// or we use a separate logic. Let's fetch the owner ID if admin.
	targetUserID := authUser.ID
	if authUser.IsAdmin {
		existing, err := h.eventStore.GetEvent(r.Context(), id)
		if err == nil {
			targetUserID = existing.UserID
		}
	}

	params := db.PatchEventParams{ID: id, UserID: targetUserID}
	if req.Title != "" {
		params.Title = sql.NullString{String: req.Title, Valid: true}
	}
	if req.OwnerName != "" {
		params.OwnerName = sql.NullString{String: req.OwnerName, Valid: true}
	}
	if req.Description != "" {
		params.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.Location != "" {
		params.Location = sql.NullString{String: req.Location, Valid: true}
	}
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid start_time format")
			return
		}
		params.StartTime = sql.NullTime{Time: t, Valid: true}
	}
	if req.ImageURL != "" {
		params.ImageUrl = sql.NullString{String: req.ImageURL, Valid: true}
	}
	if req.Price > 0 {
		params.Price = sql.NullFloat64{Float64: req.Price, Valid: true}
	}

	updated, err := h.eventStore.PatchEvent(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to update event", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update event")
		return
	}

	// Invalidate cached event lists after update
	_ = h.cache.DeleteByPattern(r.Context(), "events:*")

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"event": updated})
}

// DeleteEvent deletes an event and invalidates cache
func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	authUser, _ := middleware.RequireUser(r.Context())
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if authUser.IsAdmin {
		_ = h.eventStore.AdminDeleteEvent(r.Context(), id)
	} else {
		_ = h.eventStore.DeleteEvent(r.Context(), id, authUser.ID)
	}

	// Invalidate cached event lists after deletion
	_ = h.cache.DeleteByPattern(r.Context(), "events:*")

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "event deleted successfully"})
}

// GetEvent returns a single event by ID
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.logger.Error("Invalid event ID", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	key := fmt.Sprintf("event:%d", id)

	// Try cache first
	if cached, err := h.cache.Get(ctx, key); err == nil && cached != "" {
		var response any
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.logger.Info(fmt.Sprintf("Returning cached event %d", id))
			utils.RespondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	// Cache miss
	h.logger.Info(fmt.Sprintf("Fetching event %d from DB", id))
	event, err := h.eventStore.GetEventWithOwner(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get event", err)
		utils.RespondWithError(w, http.StatusNotFound, "event not found")
		return
	}

	response := map[string]interface{}{"event": event}

	// Cache for 60 seconds
	data, _ := json.Marshal(response)
	if err := h.cache.Set(ctx, key, data, 60*time.Second); err != nil {
		h.logger.Error("Failed to cache event", err)
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

// ListEventsWithStats returns events with ticket stats (no cache)
func (h *EventHandler) ListEventsWithStats(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventStore.ListEventsWithTicketStats(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{"events": events})
}

// SearchEvents searches events by title or location (with cache)
func (h *EventHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query().Get("q")
	if query == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "search query required")
		return
	}

	p := utils.ParsePagination(r)
	
	key := fmt.Sprintf("events:search:%s:limit=%d:offset=%d", query, p.Limit, p.Offset)

	// Try cache first
	if cached, err := h.cache.Get(ctx, key); err == nil && cached != "" {
		var response any
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.logger.Info(fmt.Sprintf("Returning cached search for '%s'", query))
			utils.RespondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	// Cache miss
	h.logger.Info(fmt.Sprintf("Fetching search results for '%s' from DB", query))
	searchPattern := "%" + query + "%"
	events, err := h.eventStore.SearchEvents(ctx, db.SearchEventsParams{
		Title:    searchPattern,
		Location: searchPattern,
		Limit:    p.Limit,
		Offset:   p.Offset,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to search events")
		return
	}

	response := map[string]interface{}{"events": events}

	// Cache search results for 30 seconds
	data, _ := json.Marshal(response)
	if err := h.cache.Set(ctx, key, data, 30*time.Second); err != nil {
		h.logger.Error("Failed to cache search results", err)
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

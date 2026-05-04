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

type TicketHandler struct {
	ticketStore store.TicketStoreInterface
	eventStore  store.EventStoreInterface
	signer      *utils.TicketSigner
	logger      *logger.Logger
	cache       *cache.InMemoryCache
}

func NewTicketHandler(ticketStore store.TicketStoreInterface,
	eventStore store.EventStoreInterface,
	signer *utils.TicketSigner, log *logger.Logger, cah *cache.InMemoryCache) *TicketHandler {
	return &TicketHandler{
		ticketStore: ticketStore,
		eventStore:  eventStore,
		signer:      signer,
		logger:      log,
		cache:       cah,
	}
}

type CreateTicketRequest struct {
	EventID int64   `json:"event_id"`
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
}

// CreateTicket creates a new ticket for an event - authenticated users only
func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authenticated user", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTicketRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("Failed to decode request body", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EventID == 0 || req.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "event_id and name are required")
		return
	}
	if err := utils.ValidatePrice(req.Price); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify event exists
	_, err = h.eventStore.GetEvent(r.Context(), req.EventID)
	if err != nil {
		h.logger.Error("Event not found", err)
		utils.RespondWithError(w, http.StatusNotFound, "event not found")
		return
	}

	// Server generates the signature — never trust the client
	signature := h.signer.Sign(authUser.ID, req.EventID, req.Name)

	ticket, err := h.ticketStore.CreateTicket(r.Context(), db.CreateTicketParams{
		UserID:    authUser.ID,
		EventID:   req.EventID,
		Name:      req.Name,
		Signature: signature,
		Price:     req.Price,
		Status:    "active",
	})
	if err != nil {
		h.logger.Error("Failed to create ticket", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	// Cache Invalidation
	h.cache.Delete(r.Context(), fmt.Sprintf("userTickets:userId=%d", authUser.ID))
	h.cache.Delete(r.Context(), fmt.Sprintf("event:%d:tickets", req.EventID))
	h.cache.DeleteByPattern(r.Context(), "tickets:") // Invalidate admin/generic list

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"ticket": ticket,
	})
}

// GetTicket returns a ticket by ID with detailed info
func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	ticket, err := h.ticketStore.GetTicketDetailed(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get ticket", err)
		utils.RespondWithError(w, http.StatusNotFound, "ticket not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ticket": ticket,
	})
}

// ListTickets returns paginated tickets
func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := utils.ParsePagination(r)
	
	// Generate a unique cache key for this page
	key := fmt.Sprintf("tickets:limit=%d:offset=%d", p.Limit, p.Offset)

	// Try to return cached response first
	if cached, err := h.cache.Get(ctx, key); err == nil && cached != "" {
		var response any
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.logger.Info("Returning cached tickets") // cache hit
			utils.RespondWithJSON(w, http.StatusOK, response)
			return
		}
	}
	
	// Cache miss → fetch tickets from DB
	h.logger.Info("Fetching Tickets from DB")
	tickets, err := h.ticketStore.ListTickets(ctx, db.ListTicketsParams{
		Limit:  p.Limit,
		Offset: p.Offset,
	})
	if err != nil {
		h.logger.Error("Failed to list tickets", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}

	total, err := h.ticketStore.CountTickets(ctx)
	if err != nil {
		h.logger.Error("Failed to count tickets", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to count tickets")
		return
	}
	p.Total = total

	// Build response
	response := utils.PaginatedResponse("tickets", tickets, p)

	// Cache the response for 30 seconds
	data, _ := json.Marshal(response)
	if err := h.cache.Set(ctx, key, data, 30*time.Second); err != nil {
		h.logger.Error("Failed to cache tickets", err)
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

// GetEventTickets returns tickets for a specific event
func (h *TicketHandler) GetEventTickets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	key := fmt.Sprintf("event:%d:tickets", eventID)

	// Try cache first
	if cached, err := h.cache.Get(ctx, key); err == nil && cached != "" {
		var response any
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.logger.Info(fmt.Sprintf("Returning cached tickets for event %d", eventID))
			utils.RespondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	// Cache miss
	h.logger.Info(fmt.Sprintf("Fetching tickets for event %d from DB", eventID))
	tickets, err := h.ticketStore.ListTicketsForEventWithUser(ctx, eventID)
	if err != nil {
		h.logger.Error("Failed to get event tickets", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get tickets")
		return
	}

	response := map[string]interface{}{"tickets": tickets}

	// Cache for 30 seconds
	data, _ := json.Marshal(response)
	if err := h.cache.Set(ctx, key, data, 30*time.Second); err != nil {
		h.logger.Error("Failed to cache event tickets", err)
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

type PatchTicketRequest struct {
	Name   string  `json:"name,omitempty"`
	Price  float64 `json:"price,omitempty"`
	Status string  `json:"status,omitempty"`
}

// PatchTicket updates a ticket - owner only (single query with ownership check)
func (h *TicketHandler) PatchTicket(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req PatchTicketRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("Failed to decode request body", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status != "" {
		if err := utils.ValidateTicketStatus(req.Status); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Price != 0 {
		if err := utils.ValidatePrice(req.Price); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	params := db.PatchTicketParams{ID: id, UserID: authUser.ID}
	if req.Name != "" {
		params.Name = sql.NullString{String: req.Name, Valid: true}
	}
	if req.Price != 0 {
		params.Price = sql.NullFloat64{Float64: req.Price, Valid: true}
	}
	if req.Status != "" {
		params.Status = sql.NullString{String: req.Status, Valid: true}
	}

	updated, err := h.ticketStore.PatchTicket(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to update ticket", err)
		utils.RespondWithError(w, http.StatusNotFound, "ticket not found or access denied")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ticket": updated,
	})
}

// VerifyTicket checks if a ticket's signature is valid (proves it was issued by our server).
// Accepts an optional ?signature= query param: when provided, it is compared against the
// stored signature so a scanned QR code can be fully validated end-to-end.
func (h *TicketHandler) VerifyTicket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	detailed, err := h.ticketStore.GetTicketDetailed(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get ticket", err)
		utils.RespondWithError(w, http.StatusNotFound, "ticket not found")
		return
	}

	valid := h.signer.Verify(detailed.UserID, detailed.EventID, detailed.Name, detailed.Signature)

	// If the client sent the QR signature, also check it matches the stored one
	if qrSig := r.URL.Query().Get("signature"); qrSig != "" && qrSig != detailed.Signature {
		valid = false
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ticket_id":        detailed.ID,
		"event_id":         detailed.EventID,
		"valid":            valid,
		"status":           detailed.Status,
		"name":             detailed.Name,
		"price":            detailed.Price,
		"event_title":      detailed.EventTitle,
		"event_start_time": detailed.EventStartTime,
		"event_image_url":  detailed.EventImageUrl,
		"user_username":    detailed.UserUsername,
		"user_email":       detailed.UserEmail,
		"created_at":       detailed.CreatedAt,
		"updated_at":       detailed.UpdatedAt,
	})
}

// MarkTicketUsed marks a ticket as "used" (admitted to the event).
//
// Authorization: Only the event owner or an admin can mark tickets as used.
// This is called from the Flutter scan screen after an organizer scans a QR code.
//
// Flow:
//  1. Authenticate the user from JWT token (RequireUser)
//  2. Parse ticket ID from URL
//  3. Check ticket exists and is in "active" status (not already "used" or "cancelled")
//  4. Look up the event that this ticket belongs to
//  5. Verify the authenticated user owns that event OR is an admin
//  6. Update the ticket's status to "used" via PatchTicket
//
// Possible HTTP responses:
//   - 200: Success, ticket marked as used
//   - 401: Not authenticated
//   - 403: Not the event owner or admin
//   - 404: Ticket not found
//   - 409: Ticket is already used
//   - 400: Ticket is cancelled
func (h *TicketHandler) MarkTicketUsed(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	// Get the ticket to find its event
	ticket, err := h.ticketStore.GetTicket(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get ticket", err)
		utils.RespondWithError(w, http.StatusNotFound, "ticket not found")
		return
	}

	if ticket.Status == "used" {
		utils.RespondWithError(w, http.StatusConflict, "ticket is already used")
		return
	}
	if ticket.Status == "cancelled" {
		utils.RespondWithError(w, http.StatusBadRequest, "ticket is cancelled")
		return
	}

	// Verify the caller owns the event or is admin
	event, err := h.eventStore.GetEvent(r.Context(), ticket.EventID)
	if err != nil {
		h.logger.Error("Failed to get event", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to verify ownership")
		return
	}

	if event.UserID != authUser.ID && !authUser.IsAdmin {
		utils.RespondWithError(w, http.StatusForbidden, "only the event owner or admin can mark tickets as used")
		return
	}

	// Use the ticket owner's user_id so PatchTicket ownership check passes
	updated, err := h.ticketStore.PatchTicket(r.Context(), db.PatchTicketParams{
		ID:     id,
		UserID: ticket.UserID,
		Status: sql.NullString{String: "used", Valid: true},
	})
	if err != nil {
		h.logger.Error("Failed to mark ticket as used", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update ticket")
		return
	}

	// Cache Invalidation
	h.cache.Delete(r.Context(), fmt.Sprintf("userTickets:userId=%d", ticket.UserID))
	h.cache.Delete(r.Context(), fmt.Sprintf("event:%d:tickets", ticket.EventID))
	h.cache.DeleteByPattern(r.Context(), "tickets:")

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "ticket marked as used",
		"ticket":  updated,
	})
}

// DeleteTicket deletes a ticket - owner or admin
func (h *TicketHandler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	if authUser.IsAdmin {
		if err := h.ticketStore.AdminDeleteTicket(r.Context(), id); err != nil {
			h.logger.Error("Failed to delete ticket (admin)", err)
			utils.RespondWithError(w, http.StatusNotFound, "ticket not found")
			return
		}
	} else {
		if err := h.ticketStore.DeleteTicket(r.Context(), id, authUser.ID); err != nil {
			h.logger.Error("Failed to delete ticket", err)
			utils.RespondWithError(w, http.StatusNotFound, "ticket not found or access denied")
			return
		}
	}

	// Cache Invalidation (Prefix-based or specific if we had the ticket data here)
	// Since we don't have ticket.UserID easily here without another query, 
	// we'll at least clear the generic list. 
	// Ideally DeleteTicket would fetch the ticket first to get IDs for cache clearing.
	h.cache.DeleteByPattern(r.Context(), "tickets:")
	h.cache.DeleteByPattern(r.Context(), "event:")
	h.cache.DeleteByPattern(r.Context(), "userTickets:")

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "ticket deleted successfully",
	})
}

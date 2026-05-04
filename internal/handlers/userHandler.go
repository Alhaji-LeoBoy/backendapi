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
	"femProjectSqlc/internal/tokens"
	"femProjectSqlc/internal/utils"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	userStore   store.UserStoreInterface
	eventStore  store.EventStoreInterface
	ticketStore store.TicketStoreInterface
	logger      *logger.Logger
	cache       *cache.InMemoryCache
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userStore store.UserStoreInterface,
	eventStore store.EventStoreInterface, ticketStore store.TicketStoreInterface,
	log *logger.Logger, cah *cache.InMemoryCache) *UserHandler {
	return &UserHandler{
		userStore:   userStore,
		eventStore:  eventStore,
		ticketStore: ticketStore,
		logger:      log,
		cache:       cah,
	}
}

// UserProfile returns the current authenticated user's profile (no extra DB call)
func (h *UserHandler) UserProfile(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authenticated user", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       authUser.ID,
			"username": authUser.Username,
			"email":    authUser.Email,
			"bio":       authUser.Bio,
			"is_admin":  authUser.IsAdmin,
			"created_at": authUser.CreatedAt,
			"updated_at": authUser.UpdatedAt,
		},
	})
}

// GetUserByID returns a user by ID - returns limited info if not self
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user (optional - allows both auth and public)
	authUser, _ := middleware.GetUserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Error("Invalid user id", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.userStore.GetUserByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get user", err)
		utils.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	// If requesting self or not authenticated, return full info
	// Otherwise return limited public info
	if authUser.ID == id {
		h.logger.Info("Returning full user info for self")
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"user": user,
		})
	} else {
		// Return limited public info
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"user": map[string]any{
				"id":         user.ID,
				"username":   user.Username,
				"bio":        user.Bio,
				"created_at": user.CreatedAt,
			},
		})
	}
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Username string `json:"username,omitempty"`
	Bio      string `json:"bio,omitempty"`
}

// UpdateUser updates user profile - self only
func (h *UserHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authenticated user", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateUserRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("Failed to decode request body", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	params := db.PatchUserParams{
		ID: authUser.ID,
	}
	if req.Username != "" {
		params.Username = sql.NullString{String: req.Username, Valid: true}
	}
	if req.Bio != "" {
		params.Bio = sql.NullString{String: req.Bio, Valid: true}
	}

	user, err := h.userStore.PatchUser(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to update user", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"user": user,
	})
}

// UpdatePasswordRequest represents a password update request
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdatePassword updates user password - self only
func (h *UserHandler) PatchPassword(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authenticated user", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdatePasswordRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("Failed to decode request body", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get current user to verify current password
	user, err := h.userStore.GetUserByID(r.Context(), authUser.ID)
	if err != nil {
		h.logger.Error("Failed to get user", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Verify current password
	if !tokens.CheckPasswordHash(req.CurrentPassword, user.Password) {
		h.logger.Error("Current password is incorrect", nil)
		utils.RespondWithError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	// Hash new password
	hashedPassword, err := tokens.HashPassword(req.NewPassword)
	if err != nil {
		h.logger.Error("Failed to hash password", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	_, err = h.userStore.PatchUser(r.Context(), db.PatchUserParams{
		ID:       authUser.ID,
		Password: sql.NullString{String: hashedPassword, Valid: true},
	})
	if err != nil {
		h.logger.Error("Failed to update password", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "password updated successfully",
	})
}

// UpdateEmailRequest represents an email update request
type UpdateEmailRequest struct {
	Email string `json:"email"`
}

// PatchEmail updates user email - self only
func (h *UserHandler) PatchEmail(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authenticated user", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateEmailRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		h.logger.Error("Failed to decode request body", err)
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Only check for conflicts when the email is actually changing
	if req.Email != authUser.Email {
		exists, err := h.userStore.EmailExists(r.Context(), req.Email)
		if err != nil {
			h.logger.Error("Failed to check email", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to check email")
			return
		}
		if exists {
			h.logger.Error("Email already in use", nil)
			utils.RespondWithError(w, http.StatusConflict, "email already in use")
			return
		}
	}

	_, err = h.userStore.PatchUser(r.Context(), db.PatchUserParams{
		ID:    authUser.ID,
		Email: sql.NullString{String: req.Email, Valid: true},
	})
	if err != nil {
		h.logger.Error("Failed to update email", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update email")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "email updated successfully",
	})
}

// DeleteUser deletes a user - self only
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authenticated user", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.userStore.DeleteUser(r.Context(), authUser.ID); err != nil {
		h.logger.Error("Failed to delete user", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}

// ListUsers returns a list of all users (public info only)
// http://localhost:8080/users?page=1&page_size=20
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page := int64(1)
	pageSize := int64(20)

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 64); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.ParseInt(ps, 10, 64); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	offset := (page - 1) * pageSize

	users, err := h.userStore.ListUsers(r.Context(), pageSize, offset)
	if err != nil {
		h.logger.Error("Failed to list users", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	// Admins see full user info (email + is_admin) so the admin panel can manage roles.
	caller, ok := middleware.GetUserFromContext(r.Context())
	isAdminCaller := ok && caller.IsAdmin

	publicUsers := make([]map[string]any, len(users))
	for i, u := range users {
		row := map[string]any{
			"id":         u.ID,
			"username":   u.Username,
			"bio":        u.Bio,
			"created_at": u.CreatedAt,
		}
		if isAdminCaller {
			row["email"] = u.Email
			row["is_admin"] = u.IsAdmin
			row["updated_at"] = u.UpdatedAt
		}
		publicUsers[i] = row
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"users": publicUsers,
		"pagination": map[string]interface{}{
			"page":      page,
			"page_size": pageSize,
			"count":     len(users),
		},
	})
}

// SearchUsers searches for users by username or email
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.logger.Error("search query required", nil)
		utils.RespondWithError(w, http.StatusBadRequest, "search query required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int64(10)
	offset := int64(0)

	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			offset = o
		}
	}

	users, err := h.userStore.SearchUsers(r.Context(), query, limit, offset)
	if err != nil {
		h.logger.Error("Failed to search users", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to search users")
		return
	}

	// Admins see full user info so the admin panel search works correctly.
	caller, ok := middleware.GetUserFromContext(r.Context())
	isAdminCaller := ok && caller.IsAdmin

	publicUsers := make([]map[string]any, len(users))
	for i, u := range users {
		row := map[string]any{
			"id":         u.ID,
			"username":   u.Username,
			"bio":        u.Bio,
			"created_at": u.CreatedAt,
		}
		if isAdminCaller {
			row["email"] = u.Email
			row["is_admin"] = u.IsAdmin
			row["updated_at"] = u.UpdatedAt
		}
		publicUsers[i] = row
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"users": publicUsers,
	})
}

// GetUserEvents returns events created by the authenticated user
func (h *UserHandler) GetUserEvents(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	events, err := h.eventStore.GetUserEvents(r.Context(), authUser.ID)
	if err != nil {
		h.logger.Error("Failed to get user events", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get user events")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"events": events,
	})
}

// GetUserEventsSummary returns the authenticated user's events with aggregate
// stats: tickets sold and number of unique buyers per event.
// For admins, it returns ALL events with aggregate stats.
func (h *UserHandler) GetUserEventsSummary(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var summaries []map[string]any
	var totalTickets int64
	var totalUniqueUsers int64
	var eventsCount int

	if authUser.IsAdmin {
		rows, err := h.eventStore.ListEventsWithTicketStats(r.Context())
		if err != nil {
			h.logger.Error("Failed to get all events summary for admin", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to get events summary")
			return
		}
		eventsCount = len(rows)
		summaries = make([]map[string]any, 0, eventsCount)
		for _, row := range rows {
			totalTickets += row.TicketsCount
			totalUniqueUsers += row.UniqueUsersCount
			summaries = append(summaries, map[string]any{
				"id":                 row.ID,
				"user_id":            row.UserID,
				"title":              row.Title,
				"owner_name":         row.OwnerName,
				"description":        row.Description,
				"location":           row.Location,
				"start_time":         row.StartTime,
				"image_url":          row.ImageUrl,
				"price":              row.Price,
				"created_at":         row.CreatedAt,
				"updated_at":         row.UpdatedAt,
				"tickets_count":      row.TicketsCount,
				"unique_users_count": row.UniqueUsersCount,
				"revenue":            row.Revenue,
			})
		}
	} else {
		rows, err := h.eventStore.GetUserEventsWithStats(r.Context(), authUser.ID)
		if err != nil {
			h.logger.Error("Failed to get user events summary", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to get user events summary")
			return
		}
		eventsCount = len(rows)
		summaries = make([]map[string]any, 0, eventsCount)
		for _, row := range rows {
			totalTickets += row.TicketsCount
			totalUniqueUsers += row.UniqueUsersCount
			summaries = append(summaries, map[string]any{
				"id":                 row.ID,
				"user_id":            row.UserID,
				"title":              row.Title,
				"owner_name":         row.OwnerName,
				"description":        row.Description,
				"location":           row.Location,
				"start_time":         row.StartTime,
				"image_url":          row.ImageUrl,
				"price":              row.Price,
				"created_at":         row.CreatedAt,
				"updated_at":         row.UpdatedAt,
				"tickets_count":      row.TicketsCount,
				"unique_users_count": row.UniqueUsersCount,
				"revenue":            row.Revenue,
			})
		}
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"events": summaries,
		"totals": map[string]any{
			"events_count":       eventsCount,
			"tickets_count":      totalTickets,
			"unique_users_count": totalUniqueUsers,
		},
	})
}

// GetUserTickets returns tickets belonging to the authenticated user
func (h *UserHandler) GetUserTickets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser, err := middleware.RequireUser(ctx)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	
	// Generate a unique cache key
	key := fmt.Sprintf("userTickets:userId=%d", authUser.ID)

	// Try to return cached response first
	if cached, err := h.cache.Get(ctx, key); err == nil && cached != "" {
		var response any
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.logger.Info("Returning cached tickets") // cache hit
			utils.RespondWithJSON(w, http.StatusOK, response)
			return
		}
	}
	
	h.logger.Info("Fetching tickets from DB")
	// Fetch tickets from DB
	tickets, err := h.ticketStore.GetUserTicketsWithEvent(ctx, authUser.ID)
	if err != nil {
		h.logger.Error("Failed to get user tickets", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get user tickets")
		return
	}
	
	res := map[string]any{
		"tickets": tickets,
	}
	
	// Cache the response for 60 seconds
	data, _ := json.Marshal(res)
	if err := h.cache.Set(ctx, key, data, 60*time.Second); err != nil {
		h.logger.Error("Failed to cache tickets", err)
	}
	
	// Return fresh response
	utils.RespondWithJSON(w, http.StatusOK, res)
}

package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"femProjectSqlc/internal/db"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/middleware"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/utils"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	userStore   store.UserStoreInterface
	eventStore  store.EventStoreInterface
	ticketStore store.TicketStoreInterface
	logger      *logger.Logger
}

func NewAdminHandler(userStore store.UserStoreInterface, eventStore store.EventStoreInterface, ticketStore store.TicketStoreInterface, log *logger.Logger) *AdminHandler {
	return &AdminHandler{
		userStore:   userStore,
		eventStore:  eventStore,
		ticketStore: ticketStore,
		logger:      log,
	}
}

type SetAdminRequest struct {
	IsAdmin bool `json:"is_admin"`
}

// SetUserAdmin promotes or demotes a user to/from admin
func (h *AdminHandler) SetUserAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req SetAdminRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Prevent admin from demoting themselves
	authUser, _ := middleware.RequireAdmin(r.Context())
	if authUser.ID == id && !req.IsAdmin {
		utils.RespondWithError(w, http.StatusBadRequest, "cannot remove your own admin status")
		return
	}

	if err := h.userStore.SetUserAdmin(r.Context(), req.IsAdmin, id); err != nil {
		h.logger.Error("Failed to set admin status", err)
		utils.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	action := "granted"
	if !req.IsAdmin {
		action = "revoked"
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "admin access " + action,
		"user_id": id,
	})
}

// AdminDeleteUser deletes any user by ID (admin only)
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Prevent admin from deleting themselves
	authUser, _ := middleware.RequireAdmin(r.Context())
	if authUser.ID == id {
		utils.RespondWithError(w, http.StatusBadRequest, "cannot delete yourself via admin endpoint")
		return
	}

	if err := h.userStore.DeleteUser(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete user (admin)", err)
		utils.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}

// AdminDeleteEvent deletes any event by ID (admin only)
func (h *AdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	if err := h.eventStore.AdminDeleteEvent(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete event (admin)", err)
		utils.RespondWithError(w, http.StatusNotFound, "event not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "event deleted successfully",
	})
}

// AdminDeleteTicket deletes any ticket by ID (admin only)
func (h *AdminHandler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	if err := h.ticketStore.AdminDeleteTicket(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete ticket (admin)", err)
		utils.RespondWithError(w, http.StatusNotFound, "ticket not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "ticket deleted successfully",
	})
}

// AdminCreateUserRequest is the request body for creating a user
type AdminCreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// AdminCreateUser creates a new user via admin
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req AdminCreateUserRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	// Check email uniqueness
	exists, err := h.userStore.EmailExists(r.Context(), req.Email)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to check email")
		return
	}
	if exists {
		utils.RespondWithError(w, http.StatusConflict, "email already in use")
		return
	}

	user, err := h.userStore.CreateUser(r.Context(), store.CreateUserParams{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// If admin, set it
	if req.IsAdmin {
		_ = h.userStore.SetUserAdmin(r.Context(), true, user.ID)
		user.IsAdmin = true
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"is_admin":   user.IsAdmin,
			"created_at": user.CreatedAt,
		},
	})
}

// AdminUpdateUserRequest is the request body for updating a user
type AdminUpdateUserRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	Bio      *string `json:"bio,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
}

// AdminPatchUser updates any user by ID (admin only)
func (h *AdminHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req AdminUpdateUserRequest
	if err := utils.DecodeJSON(r.Body, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get current user to check if email is changing or if role is changing
	existingUser, err := h.userStore.GetUserByID(r.Context(), id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	if req.Email != nil && *req.Email != "" && *req.Email != existingUser.Email {
		exists, err := h.userStore.EmailExists(r.Context(), *req.Email)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to check email")
			return
		}
		if exists {
			utils.RespondWithError(w, http.StatusConflict, "email already in use")
			return
		}
	}

	params := db.PatchUserParams{ID: id}
	if req.Username != nil && *req.Username != "" {
		params.Username = sql.NullString{String: *req.Username, Valid: true}
	}
	if req.Email != nil && *req.Email != "" {
		params.Email = sql.NullString{String: *req.Email, Valid: true}
	}
	if req.Bio != nil {
		params.Bio = sql.NullString{String: *req.Bio, Valid: true}
	}

	user, err := h.userStore.PatchUser(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to update user (admin)", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	// Handle role update if provided
	if req.IsAdmin != nil {
		authUser, _ := middleware.RequireAdmin(r.Context())
		if authUser.ID == id && !*req.IsAdmin {
			// Don't error out the whole request, but log that we couldn't demote self
			h.logger.Warn("Admin attempted to demote themselves via patch, ignoring role change")
		} else {
			if err := h.userStore.SetUserAdmin(r.Context(), *req.IsAdmin, id); err != nil {
				h.logger.Error("Failed to update role during patch", err)
			} else {
				user.IsAdmin = *req.IsAdmin
			}
		}
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"bio":        user.Bio,
			"is_admin":   user.IsAdmin,
			"updated_at": user.UpdatedAt,
		},
	})
}

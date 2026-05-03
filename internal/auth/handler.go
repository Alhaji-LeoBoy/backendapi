package auth

import (
	"encoding/json"
	"errors"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/tokens"
	"femProjectSqlc/internal/utils"
	"femProjectSqlc/internal/validation"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Handler struct {
	userStore  store.UserStoreInterface
	tokenStore store.TokenStoreInterface
}

func NewHandler(userStore store.UserStoreInterface, tokenStore store.TokenStoreInterface) *Handler {
	return &Handler{
		userStore:  userStore,
		tokenStore: tokenStore,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request body: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Check for unknown/extra fields
	if ok, err := validation.ValidateJSONFields(bodyBytes, req); !ok {
		log.Printf("ValidateJSONFields error: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Decode into struct
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check all required fields are present and valid (including email format)
	if ok, err := validation.ValidateClientInput(&req); !ok {
		log.Printf("ValidateClientInput error: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	hashedPassword, err := tokens.HashPassword(req.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error processing password")
		return
	}

	user, err := h.userStore.CreateUser(r.Context(), store.CreateUserParams{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	})
	if err != nil {
		log.Printf("CreateUser error: %v", err)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			utils.RespondWithError(w, http.StatusConflict, "User already exists")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	tokenPair, err := tokens.GenerateTokenPair(r.Context(), user.ID, h.tokenStore)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User: User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request body: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Check for unknown/extra fields
	if ok, err := validation.ValidateJSONFields(bodyBytes, req); !ok {
		log.Printf("ValidateJSONFields error: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Decode into struct
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check all required fields are present and valid (including email format)
	if ok, err := validation.ValidateClientInput(&req); !ok {
		log.Printf("ValidateClientInput error: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userStore.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !tokens.CheckPasswordHash(req.Password, user.Password) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	tokenPair, err := tokens.GenerateTokenPair(r.Context(), user.ID, h.tokenStore)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User: User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenPlaintext := utils.ExtractBearerToken(r)
	if tokenPlaintext == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing token")
		return
	}

	tokenHash := tokens.HashToken(tokenPlaintext)
	if err := h.tokenStore.DeleteTokenByHash(r.Context(), tokenHash); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error logging out")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := utils.ExtractBearerToken(r)
	if refreshToken == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing refresh token")
		return
	}

	tokenHash := tokens.HashToken(refreshToken)
	token, err := h.tokenStore.GetTokenByHash(r.Context(), tokenHash)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	if time.Now().After(token.Expiry) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Expired refresh token")
		return
	}

	if token.Scope != tokens.ScopeRefresh {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token scope")
		return
	}

	h.tokenStore.DeleteTokenByHash(r.Context(), tokenHash)

	tokenPair, err := tokens.GenerateTokenPair(r.Context(), token.UserID, h.tokenStore)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

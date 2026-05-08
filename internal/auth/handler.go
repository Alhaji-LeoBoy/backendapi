package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/tokens"
	"femProjectSqlc/internal/utils"
	"femProjectSqlc/internal/validation"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Handler struct {
	userStore           store.UserStoreInterface
	tokenStore          store.TokenStoreInterface
	requestSessionStore store.RequestSessionStoreInterface
}

func NewHandler(
	userStore store.UserStoreInterface,
	tokenStore store.TokenStoreInterface,
	requestSessionStore store.RequestSessionStoreInterface,
) *Handler {
	return &Handler{
		userStore:           userStore,
		tokenStore:          tokenStore,
		requestSessionStore: requestSessionStore,
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
	if err := h.createRequestSession(r, user.ID, tokenPair.AccessToken); err != nil {
		log.Printf("createRequestSession error: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Error creating request session")
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
	if err := h.createRequestSession(r, user.ID, tokenPair.AccessToken); err != nil {
		log.Printf("createRequestSession error: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Error creating request session")
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
	if err := tokens.ValidateTokenPlaintext(tokenPlaintext); err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token format")
		return
	}

	tokenHash := tokens.HashToken(tokenPlaintext)
	if err := h.tokenStore.DeleteTokenByHash(r.Context(), tokenHash); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error logging out")
		return
	}
	if err := h.requestSessionStore.RevokeRequestSessionByTokenHash(r.Context(), tokenHash); err != nil {
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
	if err := tokens.ValidateTokenPlaintext(refreshToken); err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	tokenHash := tokens.HashToken(refreshToken)
	token, err := h.tokenStore.ConsumeValidRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	tokenPair, err := tokens.GenerateTokenPair(r.Context(), token.UserID, h.tokenStore)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}
	if err := h.createRequestSession(r, token.UserID, tokenPair.AccessToken); err != nil {
		log.Printf("createRequestSession error: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Error creating request session")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

func (h *Handler) createRequestSession(r *http.Request, userID int64, accessToken string) error {
	tokenHash := tokens.HashToken(accessToken)
	ipAddress := clientIPFromRequest(r)

	_, err := h.requestSessionStore.CreateRequestSession(r.Context(), store.CreateRequestSessionParams{
		UserID:           userID,
		SessionTokenHash: tokenHash,
		IPAddress:        toNullString(ipAddress),
		UserAgent:        toNullString(r.UserAgent()),
		ExpiresAt:        time.Now().Add(tokens.AccessTokenExpiry),
	})
	return err
}

func clientIPFromRequest(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func toNullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

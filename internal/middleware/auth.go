package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/tokens"
	"net/http"
	"strings"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

type contextKey string

const UserContextKey contextKey = "user"

type Middleware struct {
	tokenStore store.TokenStoreInterface
	logger     *logger.Logger
}

func New(tokenStore store.TokenStoreInterface, log *logger.Logger) *Middleware {
	return &Middleware{
		tokenStore: tokenStore,
		logger:     log,
	}
}

type AuthenticatedUser struct {
	ID        int64          `json:"id"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	Bio       sql.NullString `json:"bio"`
	IsAdmin   bool           `json:"is_admin"`
	CreatedAt sql.NullTime   `json:"created_at"`
	UpdatedAt sql.NullTime   `json:"updated_at"`
}

// ChiAuthenticate returns chi-compatible middleware func(http.Handler) http.Handler
func (m *Middleware) ChiAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenPlaintext := extractBearerToken(r)
		if tokenPlaintext == "" {
			m.logger.Error("Missing authorization token", nil)
			respondWithError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		if err := tokens.ValidateTokenPlaintext(tokenPlaintext); err != nil {
			m.logger.Error("Invalid token format", err)
			respondWithError(w, http.StatusUnauthorized, "invalid token format")
			return
		}

		tokenHash := tokens.HashToken(tokenPlaintext)

		row, err := m.tokenStore.GetUserByTokenHashWithToken(r.Context(), tokenHash)
		if err != nil {
			m.logger.Error("Invalid or unknown token", err)
			respondWithError(w, http.StatusUnauthorized, "invalid or unknown token")
			return
		}

		if time.Now().After(row.Expiry) {
			m.logger.Error("Token expired", nil)
			respondWithError(w, http.StatusUnauthorized, "token expired")
			return
		}

		if row.Scope != tokens.ScopeAccess {
			m.logger.Error("Invalid token scope", nil)
			respondWithError(w, http.StatusUnauthorized, "invalid token scope - access token required")
			return
		}

		ctx := SetUserInContext(r.Context(), AuthenticatedUser{
			ID:        row.ID,
			Username:  row.Username,
			Email:     row.Email,
			Bio:       row.Bio,
			IsAdmin:   row.IsAdmin,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SetUserInContext(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

func GetUserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(UserContextKey).(AuthenticatedUser)
	return user, ok
}

func RequireUser(ctx context.Context) (AuthenticatedUser, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return AuthenticatedUser{}, ErrUnauthorized
	}
	return user, nil
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return after
	}
	return ""
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		// optional: log this instead of writing again
	}
}

// ChiRequireAdmin is Chi middleware that rejects non-admin users with 403.
func (m *Middleware) ChiRequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUserFromContext(r.Context())
		if !ok || !user.IsAdmin {
			respondWithError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin returns an error if the user from context is not an admin.
func RequireAdmin(ctx context.Context) (AuthenticatedUser, error) {
	user, err := RequireUser(ctx)
	if err != nil {
		return AuthenticatedUser{}, err
	}
	if !user.IsAdmin {
		return AuthenticatedUser{}, errors.New("admin access required")
	}
	return user, nil
}

// Authenticate returns standard http.HandlerFunc middleware
func (m *Middleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenPlaintext := extractBearerToken(r)
		if tokenPlaintext == "" {
			m.logger.Error("Missing authorization token", nil)
			respondWithError(w, http.StatusUnauthorized, "Missing authorization token")
			return
		}

		if err := tokens.ValidateTokenPlaintext(tokenPlaintext); err != nil {
			m.logger.Error("Invalid token format", err)
			respondWithError(w, http.StatusUnauthorized, "Invalid token format")
			return
		}

		tokenHash := tokens.HashToken(tokenPlaintext)

		row, err := m.tokenStore.GetUserByTokenHashWithToken(r.Context(), tokenHash)
		if err != nil {
			m.logger.Error("Invalid or unknown token", err)
			respondWithError(w, http.StatusUnauthorized, "Invalid or unknown token")
			return
		}

		if time.Now().After(row.Expiry) {
			m.logger.Error("Token expired", nil)
			respondWithError(w, http.StatusUnauthorized, "Token expired")
			return
		}

		if row.Scope != tokens.ScopeAccess {
			m.logger.Error("Invalid token scope - access token required", nil)
			respondWithError(w, http.StatusUnauthorized, "Invalid token scope - access token required")
			return
		}

		ctx := SetUserInContext(r.Context(), AuthenticatedUser{
			ID:        row.ID,
			Username:  row.Username,
			Email:     row.Email,
			Bio:       row.Bio,
			IsAdmin:   row.IsAdmin,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *Middleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenPlaintext := extractBearerToken(r)
		if tokenPlaintext == "" {
			next.ServeHTTP(w, r)
			return
		}

		if err := tokens.ValidateTokenPlaintext(tokenPlaintext); err != nil {
			next.ServeHTTP(w, r)
			return
		}

		tokenHash := tokens.HashToken(tokenPlaintext)

		row, err := m.tokenStore.GetUserByTokenHashWithToken(r.Context(), tokenHash)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if time.Now().After(row.Expiry) {
			next.ServeHTTP(w, r)
			return
		}

		if row.Scope != tokens.ScopeAccess {
			next.ServeHTTP(w, r)
			return
		}

		ctx := SetUserInContext(r.Context(), AuthenticatedUser{
			ID:        row.ID,
			Username:  row.Username,
			Email:     row.Email,
			Bio:       row.Bio,
			IsAdmin:   row.IsAdmin,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

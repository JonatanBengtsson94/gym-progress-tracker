package auth

import (
	"context"
	"net/http"
	"strings"
	"uuid"
)

type contextKey int

const userIdContextKey contextKey = iota

type SessionValidator interface {
	ValidateSession(uuid.UUID) (uint32, error)
}

type AuthMiddleware struct {
	service SessionValidator
}

func NewAuthMiddleware(service SessionValidator) *AuthMiddleware {
	return &AuthMiddleware{service: service}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sessionId, err := uuid.Parse(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userId, err := m.service.ValidateSession(sessionId)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(ContextWithUserId(r.Context(), userId)))
	})
}

func ContextWithUserId(ctx context.Context, userId uint32) context.Context {
	return context.WithValue(ctx, userIdContextKey, userId)
}

func UserIdFromContext(ctx context.Context) (uint32, bool) {
	userId, ok := ctx.Value(userIdContextKey).(uint32)
	return userId, ok
}

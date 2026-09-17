// Package identity carries the authenticated caller's user id on a request
// context. Establishing that identity is package auth's job; reading it is
// every handler's, including handlers in packages that auth itself depends
// on, so it lives apart from the credential and session handling.
package identity

import (
	"context"
	"net/http"
)

type contextKey int

const userIdContextKey contextKey = iota

func ContextWithUserId(ctx context.Context, userId uint32) context.Context {
	return context.WithValue(ctx, userIdContextKey, userId)
}

func UserIdFromContext(ctx context.Context) (uint32, bool) {
	userId, ok := ctx.Value(userIdContextKey).(uint32)
	return userId, ok
}

// RequireUserId extracts the authenticated user id from r's context. If
// there isn't one, it writes an Unauthorized response and returns ok=false;
// callers should return immediately in that case.
func RequireUserId(w http.ResponseWriter, r *http.Request) (userId uint32, ok bool) {
	userId, ok = UserIdFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	return userId, ok
}

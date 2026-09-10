package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type LoginService interface {
	Login(context.Context, string, string) (Session, error)
}

type AuthService interface {
	LoginService
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, err := h.service.Login(r.Context(), req.Username, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

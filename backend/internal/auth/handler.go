package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
)

type AuthService interface {
	Login(context.Context, string, string) (Session, error)
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

type LoginResponse struct {
	SessionId string `json:"session_id"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	session, err := h.service.Login(r.Context(), req.Username, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err != nil {
		httpx.InternalError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, LoginResponse{SessionId: session.SessionId.String()})
}

package user

import (
	"context"
	"errors"
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
)

type UserService interface {
	GetUser(context.Context, uint32) (User, error)
}

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

type UserResponse struct {
	UserId    uint32 `json:"user_id"`
	UserName  string `json:"user_name"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (h *UserHandler) GetUserFromSession(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	user, err := h.service.GetUser(r.Context(), userId)
	if errors.Is(err, ErrUserNotFound) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		httpx.InternalError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UserResponse{UserId: user.UserId, UserName: user.UserName, FirstName: user.FirstName, LastName: user.LastName})
}

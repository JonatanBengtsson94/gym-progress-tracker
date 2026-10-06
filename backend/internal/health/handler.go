package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/httpx"
)

const pingTimeout = 2 * time.Second

type Pinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	db Pinger
}

func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

type HealthResponse struct {
	Status string `json:"status"`
}

// GetHealth responds with 200 when the server can reach its database, and 503 when it can't.
// It needs no session.
func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		slog.Error("health check failed", "error", err)
		httpx.WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "unavailable"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

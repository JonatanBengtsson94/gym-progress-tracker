package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/health"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(context.Context) error {
	return m.err
}

func TestHealthHandler_GetHealth(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   string
	}{
		{name: "database reachable", pingErr: nil, wantStatus: http.StatusOK, wantBody: "ok"},
		{name: "database unreachable", pingErr: errors.New("connection refused"), wantStatus: http.StatusServiceUnavailable, wantBody: "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := health.NewHealthHandler(&mockPinger{err: tt.pingErr})

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			handler.GetHealth(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			var body health.HealthResponse
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if body.Status != tt.wantBody {
				t.Errorf("expected status %q, got %q", tt.wantBody, body.Status)
			}
		})
	}
}

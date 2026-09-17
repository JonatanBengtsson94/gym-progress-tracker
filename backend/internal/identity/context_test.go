package identity_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
)

func TestRequireUserId_Present(t *testing.T) {
	wantUserId := uint32(42)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), wantUserId))
	rec := httptest.NewRecorder()

	gotUserId, ok := identity.RequireUserId(rec, req)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if gotUserId != wantUserId {
		t.Errorf("expected userId %d, got %d", wantUserId, gotUserId)
	}
	if rec.Result().StatusCode != http.StatusOK {
		t.Errorf("expected no response to be written, got status %d", rec.Result().StatusCode)
	}
}

func TestRequireUserId_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	rec := httptest.NewRecorder()

	_, ok := identity.RequireUserId(rec, req)
	if ok {
		t.Fatal("expected ok=false")
	}
	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

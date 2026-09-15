package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// WriteJSON writes payload to w as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// DecodeJSONBody decodes r's JSON body into dst, writing a BadRequest
// response and returning ok=false if decoding fails.
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

// InternalError logs err and writes a generic 500 response to w.
func InternalError(w http.ResponseWriter, err error) {
	slog.Error("internal server error", "error", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// LoggingMiddleware logs the method, path, status code and duration of every request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

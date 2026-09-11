package httpx

import (
	"encoding/json"
	"net/http"
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

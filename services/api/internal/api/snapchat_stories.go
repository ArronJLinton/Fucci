package api

import (
	"encoding/json"
	"net/http"

	"github.com/ArronJLinton/fucci-api/internal/snapchat"
)

// GET /api/snapchat/stories?username=psg — public proxy to RapidAPI host snapchat6.p.rapidapi.com.
// Upstream 403/429 is passed through; configure RAPID_API_KEY and subscribe the key to that API on RapidAPI.
func (c *Config) getSnapchatUserStories(w http.ResponseWriter, r *http.Request) {
	if c.RapidAPIKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Snapchat stories are not configured (missing RAPID_API_KEY).",
		})
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "query parameter `username` is required",
		})
		return
	}

	body, status, err := snapchat.FetchUserStories(r.Context(), c.RapidAPIKey, username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

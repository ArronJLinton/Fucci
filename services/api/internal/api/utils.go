package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// apiErrorBody is the standard JSON error shape for API clients.
type apiErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Println("Responding with 5XX error: ", msg)
	}
	respondWithJSON(w, code, apiErrorBody{Error: msg})
}

// respondWithErrorCode sends a stable client message plus a machine-readable code (omit err details).
func respondWithErrorCode(w http.ResponseWriter, httpCode int, publicMsg, errCode string) {
	if httpCode > 499 {
		log.Printf("Responding with 5XX: %s (code=%s)", publicMsg, errCode)
	}
	respondWithJSON(w, httpCode, apiErrorBody{Error: publicMsg, Code: errCode})
}

// logErrorAndRespond500 logs the underlying error server-side and returns a generic 500 + code.
func logErrorAndRespond500(w http.ResponseWriter, logCtx string, err error, errCode string) {
	log.Printf("api %s: %v", logCtx, err)
	respondWithJSON(w, http.StatusInternalServerError, apiErrorBody{
		Error: "Something went wrong. Please try again.",
		Code:  errCode,
	})
}

const errCodeJSONMarshal = "API_RESPONSE_ENCODE_FAILED"

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(apiErrorBody{
			Error: "Something went wrong. Please try again.",
			Code:  errCodeJSONMarshal,
		})
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

type HealthResponse struct {
	Message string `json:"message"`
}

func HandleReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, HealthResponse{Message: "OK. Server Ready."})
}

func (c *Config) HandleRedisHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := c.Cache.HealthCheck(ctx)
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("Redis health check failed: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, HealthResponse{Message: "Redis health check passed"})
}

func (c *Config) HandleCacheStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats, err := c.Cache.GetStats(ctx)
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("Failed to get cache stats: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func HandleError(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusInternalServerError, "Something went wrong.")
}

func HTTPRequest(method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

func handleClientRequest[T any](url string, method string, headers map[string]string) (*T, error) {
	resp, err := HTTPRequest(method, url, headers, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %s", err)
	}
	defer resp.Body.Close()

	// Read the raw response body
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}

	// Create a new reader from the raw body for JSON decoding
	var response T
	err = json.Unmarshal(rawBody, &response)
	if err != nil {
		// Only log raw response on error for debugging
		fmt.Printf("Error parsing response from %s:\n%s\n", url, string(rawBody))
		return nil, fmt.Errorf("failed to parse response from api service: %s", err)
	}

	return &response, nil
}

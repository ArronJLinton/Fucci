package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func authTestRequest(method, path string, body interface{}, userID int32) *http.Request {
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := context.WithValue(r.Context(), "user_id", userID)
	return r.WithContext(ctx)
}

func TestHandleUpdateProfile_AvatarURLRejectsBadHost(t *testing.T) {
	cfg := &Config{CloudinaryCloudName: "demo-cloud"}
	rec := httptest.NewRecorder()
	req := authTestRequest(http.MethodPut, "/users/profile", map[string]string{
		"avatar_url": "https://example.com/image/upload/v1/fucci/avatars/avatar-1.jpg",
	}, 99)

	cfg.handleUpdateProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateProfile_AvatarURLAcceptsCloudinaryHost(t *testing.T) {
	cfg := &Config{CloudinaryCloudName: "demo-cloud"}
	err := cfg.validateCloudinaryMediaURLForContext(
		"https://res.cloudinary.com/demo-cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
		"avatar",
	)
	if err != nil {
		t.Fatalf("expected valid cloudinary avatar url, got %v", err)
	}
}

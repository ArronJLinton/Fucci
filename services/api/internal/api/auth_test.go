package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/auth"
)

// fakeProfileUpdatePersistence records ExecUpdate and returns a canned UserResponse from LoadUserResponse.
type fakeProfileUpdatePersistence struct {
	execCalls int
	lastQuery string
	lastArgs  []interface{}
	avatarURL string
}

func (f *fakeProfileUpdatePersistence) ExecUpdate(ctx context.Context, query string, args ...interface{}) error {
	f.execCalls++
	f.lastQuery = query
	f.lastArgs = args
	return nil
}

func (f *fakeProfileUpdatePersistence) LoadUserResponse(ctx context.Context, userID int32) (UserResponse, error) {
	return UserResponse{
		ID:        userID,
		Firstname: "Test",
		Lastname:  "User",
		Email:     "test@example.com",
		AvatarURL: f.avatarURL,
		Role:      "user",
	}, nil
}

func authTestRequest(method, path string, body interface{}, userID int32) *http.Request {
	var r *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := auth.ContextWithClaims(r.Context(), &auth.JWTClaims{UserID: userID})
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

func TestHandleUpdateProfile_ValidAvatarURLReturns200AndPersists(t *testing.T) {
	const secureURL = "https://res.cloudinary.com/demo-cloud/image/upload/v1/fucci/avatars/avatar-99.jpg"
	fake := &fakeProfileUpdatePersistence{avatarURL: secureURL}
	cfg := &Config{
		CloudinaryCloudName: "demo-cloud",
		ProfileUpdateDB:     fake,
	}
	rec := httptest.NewRecorder()
	const userID int32 = 42
	req := authTestRequest(http.MethodPut, "/users/profile", map[string]string{
		"avatar_url": secureURL,
	}, userID)

	cfg.handleUpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if fake.execCalls != 1 {
		t.Fatalf("expected ExecUpdate once, got %d", fake.execCalls)
	}
	if !strings.Contains(fake.lastQuery, "avatar_url") {
		t.Fatalf("expected UPDATE to set avatar_url, query=%q", fake.lastQuery)
	}
	if len(fake.lastArgs) < 2 {
		t.Fatalf("expected args for value and user id, got %v", fake.lastArgs)
	}
	if got := fake.lastArgs[0]; got != secureURL {
		t.Fatalf("ExecUpdate avatar arg: got %v want %q", got, secureURL)
	}
	if got := fake.lastArgs[len(fake.lastArgs)-1]; got != userID {
		t.Fatalf("ExecUpdate user id arg: got %v want %d", got, userID)
	}
	var out UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.AvatarURL != secureURL {
		t.Fatalf("response avatar_url: got %q want %q", out.AvatarURL, secureURL)
	}
	if out.ID != userID {
		t.Fatalf("response id: got %d want %d", out.ID, userID)
	}
}

func TestHandleUpdateProfile_AvatarURLReturns500WhenCloudinaryCloudNameUnset(t *testing.T) {
	cfg := &Config{}
	rec := httptest.NewRecorder()
	req := authTestRequest(http.MethodPut, "/users/profile", map[string]string{
		"avatar_url": "https://res.cloudinary.com/demo-cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
	}, 99)

	cfg.handleUpdateProfile(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "cloud name is not configured") {
		t.Fatalf("expected configuration error in body, got %q", body)
	}
}

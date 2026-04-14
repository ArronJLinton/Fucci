package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/auth"
)

func TestCloudinarySign_Deterministic(t *testing.T) {
	got := cloudinarySign(map[string]string{
		"timestamp": "1700000000",
		"folder":    "fucci/avatars",
		"public_id": "avatar-1-1700000000",
	}, "secret123")
	const expected = "e6a985499eb48ef1d8337a6de63fc5cf7d9f9385"
	if got != expected {
		t.Fatalf("signature mismatch: got %q want %q", got, expected)
	}
}

func TestPostCloudinarySignature_RequiresUploadAuthMethod(t *testing.T) {
	cfg := &Config{
		CloudinaryCloudName: "demo-cloud",
		CloudinaryAPIKey:    "api-key",
	}
	body := strings.NewReader(`{"context":"avatar"}`)
	req := httptest.NewRequest(http.MethodPost, "/upload/cloudinary/signature", body)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: 1}))
	rr := httptest.NewRecorder()
	cfg.postCloudinarySignature(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	var out struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Error == "" || !strings.Contains(out.Error, "CLOUDINARY_API_SECRET") {
		t.Fatalf("unexpected error body: %q", out.Error)
	}
}

func TestPostCloudinarySignature_OKWithAPISecret(t *testing.T) {
	cfg := &Config{
		CloudinaryCloudName: "demo-cloud",
		CloudinaryAPIKey:    "api-key",
		CloudinaryAPISecret: "secret",
	}
	body := strings.NewReader(`{"context":"avatar"}`)
	req := httptest.NewRequest(http.MethodPost, "/upload/cloudinary/signature", body)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: 1}))
	rr := httptest.NewRecorder()
	cfg.postCloudinarySignature(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d, want %d", rr.Code, http.StatusOK)
	}
	var out cloudinarySignatureResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Signature == "" {
		t.Fatal("expected non-empty signature")
	}
}

func TestValidateCloudinaryMediaURLForContext(t *testing.T) {
	cfg := Config{CloudinaryCloudName: "demo-cloud"}

	t.Run("accepts valid avatar URL", func(t *testing.T) {
		err := cfg.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/demo-cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
			"avatar",
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("accepts avatar URL when cloud segment casing differs from config", func(t *testing.T) {
		mixedCase := Config{CloudinaryCloudName: "demo-cloud"}
		err := mixedCase.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/Demo-Cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
			"avatar",
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects wrong host", func(t *testing.T) {
		err := cfg.validateCloudinaryMediaURLForContext(
			"https://example.com/demo-cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
			"avatar",
		)
		if err == nil {
			t.Fatal("expected error for invalid host")
		}
	})

	t.Run("rejects wrong folder for context", func(t *testing.T) {
		err := cfg.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/demo-cloud/image/upload/v1/fucci/player-profiles/p1.jpg",
			"avatar",
		)
		if err == nil {
			t.Fatal("expected error for invalid folder")
		}
	})

	t.Run("rejects wrong cloud name", func(t *testing.T) {
		err := cfg.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/other-cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
			"avatar",
		)
		if err == nil {
			t.Fatal("expected error for invalid cloud name")
		}
	})

	t.Run("rejects folder segment only later in public id path", func(t *testing.T) {
		err := cfg.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/demo-cloud/image/upload/v1/other/fucci/avatars/not-really-ours.jpg",
			"avatar",
		)
		if err == nil {
			t.Fatal("expected error when allowed folder is not the public id prefix")
		}
	})

	t.Run("accepts URL with transformation and version before public id", func(t *testing.T) {
		err := cfg.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/demo-cloud/image/upload/c_scale,w_400/v1699123456/fucci/avatars/avatar-1.jpg",
			"avatar",
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects when cloud name not configured", func(t *testing.T) {
		loose := Config{}
		err := loose.validateCloudinaryMediaURLForContext(
			"https://res.cloudinary.com/demo-cloud/image/upload/v1/fucci/avatars/avatar-1.jpg",
			"avatar",
		)
		if !errors.Is(err, ErrCloudinaryURLValidationNotConfigured) {
			t.Fatalf("expected ErrCloudinaryURLValidationNotConfigured, got %v", err)
		}
	})
}

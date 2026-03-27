package api

import "testing"

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
}

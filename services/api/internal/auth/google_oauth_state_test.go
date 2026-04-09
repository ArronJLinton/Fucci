package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestAllowedGoogleAppReturnURI_ProductionDefault(t *testing.T) {
	cases := []struct {
		raw string
		ok  bool
	}{
		{"fucci://auth", true},
		{"https://auth.expo.io/@someone/slug", true},
		{"exp://127.0.0.1:8081/--/auth", false},
		{"http://localhost:8081/auth", false},
		{"http://10.0.2.2:8080/cb", false},
		{"http://auth.expo.io/cb", false},
		{"https://localhost:8081/auth", false},
		{"com.magistridev.fucci:/oauth2redirect", false},
		{"https://evil.com/phish", false},
		{"https://auth.expo.io.evil.com/x", false},
		{"javascript:alert(1)", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := AllowedGoogleAppReturnURI(tc.raw, false); got != tc.ok {
			t.Fatalf("AllowedGoogleAppReturnURI(%q, false) = %v, want %v", tc.raw, got, tc.ok)
		}
	}
}

func TestAllowedGoogleAppReturnURI_DevFlag(t *testing.T) {
	cases := []struct {
		raw string
		ok  bool
	}{
		{"fucci://auth", true},
		{"https://auth.expo.io/@someone/slug", true},
		{"exp://127.0.0.1:8081/--/auth", true},
		{"http://localhost:8081/auth", true},
		{"http://10.0.2.2:8080/cb", true},
		{"http://192.168.1.1/x", true},
		{"https://localhost:8081/--/auth", true},
		{"https://evil.com/phish", false},
		{"http://evil.com/x", false},
	}
	for _, tc := range cases {
		if got := AllowedGoogleAppReturnURI(tc.raw, true); got != tc.ok {
			t.Fatalf("AllowedGoogleAppReturnURI(%q, true) = %v, want %v", tc.raw, got, tc.ok)
		}
	}
}

func TestSignAndParseGoogleOAuthState(t *testing.T) {
	_ = InitJWTAuth("test-secret-for-state")
	tok, err := SignGoogleOAuthState("fucci://auth")
	if err != nil {
		t.Fatal(err)
	}
	ret, err := ParseGoogleOAuthState(tok)
	if err != nil || ret != "fucci://auth" {
		t.Fatalf("parse: %v %q", err, ret)
	}
}

func TestParseGoogleOAuthState_ErrorsWhenJWTNotInitialized(t *testing.T) {
	prev := jwtSecret
	jwtSecret = nil
	t.Cleanup(func() { jwtSecret = prev })

	_, err := ParseGoogleOAuthState("any-token")
	if !errors.Is(err, ErrJWTNotInitialized) {
		t.Fatalf("expected ErrJWTNotInitialized, got %v", err)
	}
}

func TestParseGoogleOAuthState_RejectsUnexpectedSigningMethod(t *testing.T) {
	_ = InitJWTAuth("test-secret-for-state")

	claims := &GoogleOAuthStateClaims{
		Purpose:   googleOAuthStatePurpose,
		ReturnURL: "fucci://auth",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	raw, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseGoogleOAuthState(raw)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unexpected signing method") {
		t.Fatalf("expected unexpected signing method error, got %v", err)
	}
}

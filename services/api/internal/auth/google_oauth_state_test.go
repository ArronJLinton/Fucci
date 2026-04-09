package auth

import "testing"

func TestAllowedGoogleAppReturnURI(t *testing.T) {
	cases := []struct {
		raw string
		ok  bool
	}{
		{"fucci://auth", true},
		{"exp://127.0.0.1:8081/--/auth", true},
		{"https://auth.expo.io/@someone/slug", true},
		{"http://localhost:8081/auth", true},
		{"http://10.0.2.2:8080/cb", true},
		{"com.magistridev.fucci:/oauth2redirect", true},
		{"https://evil.com/phish", false},
		{"javascript:alert(1)", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := AllowedGoogleAppReturnURI(tc.raw); got != tc.ok {
			t.Fatalf("AllowedGoogleAppReturnURI(%q) = %v, want %v", tc.raw, got, tc.ok)
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

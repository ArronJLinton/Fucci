package auth

import "testing"

func TestGoogleOAuthVerifier_ValidateRedirectURI(t *testing.T) {
	verifier := NewGoogleOAuthVerifier("client-id", "secret", []string{
		"fucci://auth",
		"com.fucci.app:/oauth2redirect",
	})

	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{name: "ios allowed", uri: "fucci://auth"},
		{name: "android allowed", uri: "com.fucci.app:/oauth2redirect"},
		{name: "unknown uri rejected", uri: "https://example.com/callback", wantErr: true},
		{name: "empty uri rejected", uri: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := verifier.ValidateRedirectURI(tc.uri)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for uri %q", tc.uri)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for uri %q, got %v", tc.uri, err)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	if !toBool(true) {
		t.Fatal("expected true for bool true")
	}
	if !toBool("true") {
		t.Fatal("expected true for string true")
	}
	if toBool("false") {
		t.Fatal("expected false for string false")
	}
}


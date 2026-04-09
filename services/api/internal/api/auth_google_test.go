package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/DATA-DOG/go-sqlmock"
)

type fakeGoogleVerifier struct {
	exchangeFn func(ctx context.Context, code, redirectURI string) (string, error)
	verifyFn   func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error)
}

func (f *fakeGoogleVerifier) ExchangeCodeForIDToken(ctx context.Context, code, redirectURI string) (string, error) {
	return f.exchangeFn(ctx, code, redirectURI)
}

func (f *fakeGoogleVerifier) VerifyIDToken(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
	return f.verifyFn(ctx, token)
}

func TestHandleGoogleAuth_NewUserReturnsIsNewTrue(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-123",
					Email:         "newuser@example.com",
					EmailVerified: true,
					GivenName:     "New",
					FamilyName:    "User",
					Picture:       "https://cdn.example/avatar.jpg",
					Locale:        "en",
				}, nil
			},
		},
	}

	mock.ExpectQuery("SELECT id, COALESCE\\(role, 'fan'\\) FROM users WHERE google_id = \\$1 LIMIT 1").
		WithArgs("sub-123").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, auth_provider FROM users WHERE lower\\(email\\) = lower\\(\\$1\\) LIMIT 1").
		WithArgs("newuser@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO users").
		WithArgs("New", "User", "newuser@example.com", "sub-123", "https://cdn.example/avatar.jpg", "en").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int32(101)))
	mock.ExpectQuery("SELECT id, firstname, lastname, email, COALESCE\\(display_name, ''\\), COALESCE\\(avatar_url, ''\\), COALESCE\\(is_verified, false\\), COALESCE\\(is_active, true\\), COALESCE\\(role, 'fan'\\), created_at::text, updated_at::text FROM users WHERE id = \\$1").
		WithArgs(int32(101)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "firstname", "lastname", "email", "display_name", "avatar_url", "is_verified", "is_active", "role", "created_at", "updated_at",
		}).AddRow(101, "New", "User", "newuser@example.com", "", "https://cdn.example/avatar.jpg", true, true, "fan", "2026-01-01", "2026-01-01"))

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	cfg.handleGoogleAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out GoogleAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !out.IsNew {
		t.Fatalf("expected is_new=true")
	}
	if out.Token == "" {
		t.Fatalf("expected token")
	}
	if out.User.Email != "newuser@example.com" {
		t.Fatalf("expected user email, got %q", out.User.Email)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleGoogleAuth_EmailNotVerifiedReturns400(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-123",
					Email:         "newuser@example.com",
					EmailVerified: false,
				}, nil
			},
		},
	}

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out googleAuthErrorPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthEmailNotVerified {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthEmailNotVerified, out.Code)
	}
}

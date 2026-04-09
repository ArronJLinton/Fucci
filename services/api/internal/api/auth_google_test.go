package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/DATA-DOG/go-sqlmock"
)

// Regexes match sqlc-generated queries used by googleAuthFromCode (substring match).
var (
	rxSQLGoogleGetByGoogleID = `SELECT id, firstname, lastname, email, created_at, updated_at, is_admin, display_name, avatar_url, google_id, auth_provider, locale, last_login_at, is_verified, is_active, role FROM users WHERE google_id = \$1::text`
	rxSQLGoogleGetByEmailLower = `SELECT id, firstname, lastname, email, created_at, updated_at, is_admin, display_name, avatar_url, google_id, auth_provider, locale, last_login_at, is_verified, is_active, role FROM users WHERE lower\(email\) = lower\(\$1\) LIMIT 1`
	rxSQLGoogleCreateUser = `INSERT INTO users \(firstname, lastname, email, google_id, auth_provider, avatar_url, locale, is_admin, is_active, is_verified, last_login_at\)`
	rxSQLGoogleGetUserByID = `SELECT id, firstname, lastname, email, created_at, updated_at, is_admin, display_name, avatar_url, google_id, auth_provider, locale, last_login_at, is_verified, is_active, role FROM users WHERE id = \$1`
	rxSQLGoogleUpdateLogin = `avatar_url = CASE WHEN \$1::text <> '' THEN \$1 ELSE avatar_url END`
	rxSQLGoogleLink = `COALESCE\(NULLIF\(google_id::text, ''\), \$1::text\)::varchar\(255\)`
)

var sqlGoogleAuthUserColumns = []string{
	"id", "firstname", "lastname", "email", "created_at", "updated_at", "is_admin",
	"display_name", "avatar_url", "google_id", "auth_provider", "locale", "last_login_at",
	"is_verified", "is_active", "role",
}

func sqlMockGoogleUserFullRow(id int32, firstname, lastname, email, avatarURL, googleSub, authProv string, ts time.Time) *sqlmock.Rows {
	return sqlmock.NewRows(sqlGoogleAuthUserColumns).AddRow(
		id, firstname, lastname, email, ts, ts, false,
		sql.NullString{},
		sql.NullString{String: avatarURL, Valid: avatarURL != ""},
		sql.NullString{String: googleSub, Valid: googleSub != ""},
		authProv,
		sql.NullString{},
		sql.NullTime{},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullString{String: "fan", Valid: true},
	)
}

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

func TestHandleGoogleAuth_NotConfiguredReturns503(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		// Intentionally unset GoogleOAuthClientID/GoogleOAuthClientSecret.
	}

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthNotConfigured {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthNotConfigured, out.Code)
	}
}

func TestGoogleAuthFromCode_NotConfiguredBeforeExchange(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				t.Fatal("exchange must not be called when OAuth client credentials are missing")
				return "", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				t.Fatal("verify must not be called")
				return auth.GoogleIDTokenClaims{}, nil
			},
		},
	}

	_, procErr := cfg.googleAuthFromCode(context.Background(), "any-code", "https://example/callback")
	if procErr == nil {
		t.Fatal("expected procErr")
	}
	if procErr.status != http.StatusServiceUnavailable {
		t.Fatalf("status %d want %d", procErr.status, http.StatusServiceUnavailable)
	}
	if procErr.code != auth.GoogleAuthNotConfigured {
		t.Fatalf("code %q want %q", procErr.code, auth.GoogleAuthNotConfigured)
	}
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
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
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

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-123").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleGetByEmailLower).
		WithArgs("newuser@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleCreateUser).
		WithArgs("New", "User", "newuser@example.com", sql.NullString{String: "sub-123", Valid: true}, sql.NullString{String: "https://cdn.example/avatar.jpg", Valid: true}, sql.NullString{String: "en", Valid: true}).
		WillReturnRows(sqlMockGoogleUserFullRow(101, "New", "User", "newuser@example.com", "https://cdn.example/avatar.jpg", "sub-123", "google", ts))
	mock.ExpectQuery(rxSQLGoogleGetUserByID).
		WithArgs(int32(101)).
		WillReturnRows(sqlMockGoogleUserFullRow(101, "New", "User", "newuser@example.com", "https://cdn.example/avatar.jpg", "sub-123", "google", ts))

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
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
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
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthEmailNotVerified {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthEmailNotVerified, out.Code)
	}
}

func TestHandleGoogleAuth_ExistingGoogleUserReturnsIsNewFalse(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-existing",
					Email:         "existing@example.com",
					EmailVerified: true,
					Picture:       "https://cdn.example/new-avatar.jpg",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-existing").
		WillReturnRows(sqlMockGoogleUserFullRow(42, "Existing", "User", "existing@example.com", "https://cdn.example/old.jpg", "sub-existing", "google", ts))
	mock.ExpectQuery(rxSQLGoogleUpdateLogin).
		WithArgs("https://cdn.example/new-avatar.jpg", int32(42)).
		WillReturnRows(sqlMockGoogleUserFullRow(42, "Existing", "User", "existing@example.com", "https://cdn.example/new-avatar.jpg", "sub-existing", "google", ts2))
	mock.ExpectQuery(rxSQLGoogleGetUserByID).
		WithArgs(int32(42)).
		WillReturnRows(sqlMockGoogleUserFullRow(42, "Existing", "User", "existing@example.com", "https://cdn.example/new-avatar.jpg", "sub-existing", "google", ts2))

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
	if out.IsNew {
		t.Fatalf("expected is_new=false")
	}
	if out.User.Email != "existing@example.com" {
		t.Fatalf("expected existing user email, got %q", out.User.Email)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleGoogleAuth_InvalidCodeReturns400(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "", auth.ErrGoogleInvalidCode
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{}, nil
			},
		},
	}

	body := map[string]string{"code": "bad-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthCodeInvalid {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthCodeInvalid, out.Code)
	}
}

func TestHandleGoogleAuth_GoogleExchangeFailedReturns500(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "", auth.ErrGoogleExchangeFailed
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{}, nil
			},
		},
	}

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthUpstreamAPIError {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthUpstreamAPIError, out.Code)
	}
}

func TestHandleGoogleAuth_TokenVerifyFailedReturns401(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn: db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{}, errors.New("verify failed")
			},
		},
	}

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthTokenVerifyFailed {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthTokenVerifyFailed, out.Code)
	}
}

func TestHandleGoogleAuth_ExistingGoogleUserUpdateFailureReturns500(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-existing",
					Email:         "existing@example.com",
					EmailVerified: true,
					Picture:       "https://cdn.example/new-avatar.jpg",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-existing").
		WillReturnRows(sqlMockGoogleUserFullRow(42, "Existing", "User", "existing@example.com", "https://cdn.example/old.jpg", "sub-existing", "google", ts))
	mock.ExpectQuery(rxSQLGoogleUpdateLogin).
		WithArgs("https://cdn.example/new-avatar.jpg", int32(42)).
		WillReturnError(errors.New("db write failed"))

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthUpstreamAPIError {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthUpstreamAPIError, out.Code)
	}
}

func TestHandleGoogleAuth_EmailFallbackUpdateFailureReturns500(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-fallback",
					Email:         "existing-social@example.com",
					EmailVerified: true,
					Picture:       "https://cdn.example/new-avatar.jpg",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-fallback").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleGetByEmailLower).
		WithArgs("existing-social@example.com").
		WillReturnRows(sqlmock.NewRows(sqlGoogleAuthUserColumns).AddRow(
			int32(77), "", "", "existing-social@example.com", ts, ts, false,
			sql.NullString{}, sql.NullString{},
			sql.NullString{}, "google",
			sql.NullString{}, sql.NullTime{},
			sql.NullBool{}, sql.NullBool{},
			sql.NullString{String: "fan", Valid: true},
		))
	mock.ExpectQuery(rxSQLGoogleLink).
		WithArgs("sub-fallback", "https://cdn.example/new-avatar.jpg", int32(77)).
		WillReturnError(errors.New("db write failed"))

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthUpstreamAPIError {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthUpstreamAPIError, out.Code)
	}
}

func TestHandleGoogleAuth_EmailMatchedDifferentGoogleIDReturns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "incoming-subject",
					Email:         "existing-social@example.com",
					EmailVerified: true,
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("incoming-subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleGetByEmailLower).
		WithArgs("existing-social@example.com").
		WillReturnRows(sqlmock.NewRows(sqlGoogleAuthUserColumns).AddRow(
			int32(77), "", "", "existing-social@example.com", ts, ts, false,
			sql.NullString{}, sql.NullString{},
			sql.NullString{String: "different-subject", Valid: true}, "google",
			sql.NullString{}, sql.NullTime{},
			sql.NullBool{}, sql.NullBool{},
			sql.NullString{String: "fan", Valid: true},
		))

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthAccountExistsEmail {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthAccountExistsEmail, out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleGoogleAuth_EmailFallbackNonGoogleProviderReturns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "incoming-subject",
					Email:         "apple-user@example.com",
					EmailVerified: true,
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("incoming-subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleGetByEmailLower).
		WithArgs("apple-user@example.com").
		WillReturnRows(sqlmock.NewRows(sqlGoogleAuthUserColumns).AddRow(
			int32(88), "", "", "apple-user@example.com", ts, ts, false,
			sql.NullString{}, sql.NullString{},
			sql.NullString{}, "apple",
			sql.NullString{}, sql.NullTime{},
			sql.NullBool{}, sql.NullBool{},
			sql.NullString{String: "fan", Valid: true},
		))

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthAccountExistsEmail {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthAccountExistsEmail, out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestIsGoogleProviderCancellation(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    bool
	}{
		{name: "access denied", code: "access_denied", want: true},
		{name: "user cancelled", code: "user_cancelled", want: true},
		{name: "user canceled", code: "user_canceled", want: true},
		{name: "trim and case", code: "  Access_Denied ", want: true},
		{name: "non cancel error", code: "invalid_request", want: false},
		{name: "empty", code: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isGoogleProviderCancellation(tc.code)
			if got != tc.want {
				t.Fatalf("isGoogleProviderCancellation(%q) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestGoogleOAuthExchangeCode_SingleUse(t *testing.T) {
	in := GoogleAuthResponse{
		Token: "jwt-token",
		User: UserResponse{
			ID:    99,
			Email: "user@example.com",
			Role:  "fan",
		},
		IsNew: true,
	}

	cfg := &Config{}
	code, err := cfg.issueGoogleOAuthExchangeCode(context.Background(), in)
	if err != nil {
		t.Fatalf("issueGoogleOAuthExchangeCode error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected non-empty code")
	}

	out, ok := cfg.consumeGoogleOAuthExchangeCode(context.Background(), code)
	if !ok {
		t.Fatalf("expected first consume to succeed")
	}
	if out.Token != in.Token || out.User.ID != in.User.ID || out.IsNew != in.IsNew {
		t.Fatalf("unexpected consumed payload: %#v", out)
	}

	_, ok = cfg.consumeGoogleOAuthExchangeCode(context.Background(), code)
	if ok {
		t.Fatalf("expected second consume to fail (single use)")
	}
}

func TestGoogleOAuthExchangeCode_SingleUse_SharedCache(t *testing.T) {
	store := make(map[string][]byte)
	var mu sync.Mutex
	mc := &MockCache{
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			b, err := json.Marshal(value)
			if err != nil {
				return err
			}
			store[key] = b
			return nil
		},
		getDelFunc: func(ctx context.Context, key string, value interface{}) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			b, ok := store[key]
			if !ok {
				return false, nil
			}
			delete(store, key)
			if err := json.Unmarshal(b, value); err != nil {
				return false, err
			}
			return true, nil
		},
	}

	in := GoogleAuthResponse{
		Token: "jwt-shared",
		User:  UserResponse{ID: 42, Email: "x@y.com"},
		IsNew: false,
	}
	cfg := &Config{Cache: mc}
	code, err := cfg.issueGoogleOAuthExchangeCode(context.Background(), in)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	out, ok := cfg.consumeGoogleOAuthExchangeCode(context.Background(), code)
	if !ok {
		t.Fatal("first consume should succeed")
	}
	if out.Token != in.Token {
		t.Fatalf("token mismatch")
	}
	if _, ok := cfg.consumeGoogleOAuthExchangeCode(context.Background(), code); ok {
		t.Fatal("second consume should fail")
	}
}

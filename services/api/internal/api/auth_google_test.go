package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/DATA-DOG/go-sqlmock"
)

// Regexes match sqlc-generated queries used by googleAuthFromCode (substring match).
var (
	rxSQLGoogleGetByGoogleID   = `SELECT id, firstname, lastname, email, created_at, updated_at, is_admin, display_name, avatar_url, google_id, auth_provider, locale, last_login_at, is_verified, is_active, role FROM users WHERE google_id = \$1::varchar\(255\)`
	rxSQLGoogleGetByEmailLower = `SELECT id, firstname, lastname, email, created_at, updated_at, is_admin, display_name, avatar_url, google_id, auth_provider, locale, last_login_at, is_verified, is_active, role FROM users WHERE lower\(email\) = lower\(\$1\) LIMIT 1`
	rxSQLGoogleCreateUser      = `INSERT INTO users \(firstname, lastname, email, google_id, auth_provider, avatar_url, locale, is_admin, is_active, is_verified, last_login_at\)`
	rxSQLGoogleUpdateLogin     = `avatar_url = CASE WHEN \$1::text <> '' THEN \$1 ELSE avatar_url END`
	rxSQLGoogleLink            = `COALESCE\(NULLIF\(google_id::text, ''\), \$1::text\)::varchar\(255\)`
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

func sqlMockGoogleUserInactiveRow(id int32, firstname, lastname, email, avatarURL, googleSub, authProv string, ts time.Time) *sqlmock.Rows {
	return sqlmock.NewRows(sqlGoogleAuthUserColumns).AddRow(
		id, firstname, lastname, email, ts, ts, false,
		sql.NullString{},
		sql.NullString{String: avatarURL, Valid: avatarURL != ""},
		sql.NullString{String: googleSub, Valid: googleSub != ""},
		authProv,
		sql.NullString{},
		sql.NullTime{},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullBool{Bool: false, Valid: true},
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

func TestPublicGoogleOAuthAppErrorDescription_OmitsInternalDetail(t *testing.T) {
	// 5xx: description must not echo internal DB-style messages (shown in URL to app).
	internal := &googleAuthProcError{
		status: http.StatusInternalServerError,
		code:   auth.GoogleAuthUpstreamAPIError,
		msg:    "failed to update google login fields",
	}
	got := publicGoogleOAuthAppErrorDescription(internal)
	if got == internal.msg || strings.Contains(got, "failed to") {
		t.Fatalf("expected generic public text, got %q", got)
	}
	email := &googleAuthProcError{
		status: http.StatusBadRequest,
		code:   auth.GoogleAuthEmailNotVerified,
		msg:    "Google email is not verified",
	}
	if !strings.Contains(publicGoogleOAuthAppErrorDescription(email), "Verify") {
		t.Fatalf("expected user-facing hint for email, got %q", publicGoogleOAuthAppErrorDescription(email))
	}
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

func TestGoogleAuthFromCode_MissingSubjectOrEmailUnauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "cid",
		GoogleOAuthClientSecret: "sec",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "",
					Email:         "a@b.com",
					EmailVerified: true,
				}, nil
			},
		},
	}

	_, procErr := cfg.googleAuthFromCode(context.Background(), "code", "https://cb")
	if procErr == nil {
		t.Fatal("expected procErr for empty subject")
	}
	if procErr.status != http.StatusUnauthorized || procErr.code != auth.GoogleAuthTokenVerifyFailed {
		t.Fatalf("got status=%d code=%q", procErr.status, procErr.code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("db should not be queried: %v", err)
	}

	cfg2 := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "cid",
		GoogleOAuthClientSecret: "sec",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-x",
					Email:         "   ",
					EmailVerified: true,
				}, nil
			},
		},
	}
	_, procErr = cfg2.googleAuthFromCode(context.Background(), "code", "https://cb")
	if procErr == nil {
		t.Fatal("expected procErr for empty email")
	}
	if procErr.status != http.StatusUnauthorized || procErr.code != auth.GoogleAuthTokenVerifyFailed {
		t.Fatalf("got status=%d code=%q", procErr.status, procErr.code)
	}
}

func TestGoogleAuthFromCode_InvalidEmailClaimUnauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "cid",
		GoogleOAuthClientSecret: "sec",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-1",
					Email:         "not-a-valid-email",
					EmailVerified: true,
				}, nil
			},
		},
	}

	_, procErr := cfg.googleAuthFromCode(context.Background(), "code", "https://cb")
	if procErr == nil {
		t.Fatal("expected procErr for invalid email")
	}
	if procErr.status != http.StatusUnauthorized || procErr.code != auth.GoogleAuthTokenVerifyFailed {
		t.Fatalf("got status=%d code=%q msg=%q", procErr.status, procErr.code, procErr.msg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("db should not be queried: %v", err)
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
		DBConn:                  db,
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
		DBConn:                  db,
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
	ts2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-existing").
		WillReturnRows(sqlMockGoogleUserFullRow(42, "Existing", "User", "existing@example.com", "https://cdn.example/old.jpg", "sub-existing", "google", ts))
	mock.ExpectQuery(rxSQLGoogleUpdateLogin).
		WithArgs("https://cdn.example/new-avatar.jpg", int32(42)).
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

func TestHandleGoogleAuth_InactiveUserReturns403(t *testing.T) {
	_ = InitJWT("test-secret")
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
					Subject:       "sub-inactive",
					Email:         "inactive@example.com",
					EmailVerified: true,
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-inactive").
		WillReturnRows(sqlMockGoogleUserInactiveRow(42, "In", "Active", "inactive@example.com", "", "sub-inactive", "google", ts))

	body := map[string]string{"code": "auth-code", "redirect_uri": "fucci://auth"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleAuth(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if out.Code != auth.GoogleAuthAccountInactive {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthAccountInactive, out.Code)
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
		DBConn:                  db,
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

func TestHandleGoogleAuth_MissingRedirectURIReturnsInvalidRedirectURI(t *testing.T) {
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
				t.Fatal("exchange must not be called when redirect_uri is missing")
				return "", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				t.Fatal("verify must not be called")
				return auth.GoogleIDTokenClaims{}, nil
			},
		},
	}

	body := map[string]string{"code": "auth-code", "redirect_uri": ""}
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
	if out.Code != auth.GoogleAuthInvalidRedirectURI {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthInvalidRedirectURI, out.Code)
	}
}

func TestHandleGoogleAuth_InvalidRedirectURIReturns400(t *testing.T) {
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
				return "", auth.ErrGoogleInvalidRedirectURI
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{}, errors.New("verify must not be called when exchange fails")
			},
		},
	}

	body := map[string]string{"code": "auth-code", "redirect_uri": "https://evil.example/cb"}
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
	if out.Code != auth.GoogleAuthInvalidRedirectURI {
		t.Fatalf("expected code %s, got %s", auth.GoogleAuthInvalidRedirectURI, out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleGoogleAuth_EmailPasswordAccountReturns409(t *testing.T) {
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
					Subject:       "google-sub-new",
					Email:         "password-only@example.com",
					EmailVerified: true,
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("google-sub-new").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleGetByEmailLower).
		WithArgs("password-only@example.com").
		WillReturnRows(sqlmock.NewRows(sqlGoogleAuthUserColumns).AddRow(
			int32(55), "Pass", "User", "password-only@example.com", ts, ts, false,
			sql.NullString{}, sql.NullString{},
			sql.NullString{}, "email",
			sql.NullString{}, sql.NullTime{},
			sql.NullBool{Bool: true, Valid: true},
			sql.NullBool{Bool: true, Valid: true},
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
		DBConn:                  db,
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
		name string
		code string
		want bool
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

func TestHandleGoogleOAuthCallback_SuccessRedirectsWithExchangeCode(t *testing.T) {
	_ = InitJWT("test-secret")
	state, err := auth.SignGoogleOAuthState("fucci://auth")
	if err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cbURL := "https://example.com/v1/api/auth/google/callback"
	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "test-google-client-id",
		GoogleOAuthClientSecret: "test-google-client-secret",
		GoogleOAuthCallbackURL:  cbURL,
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				if code != "google-auth-code" || redirectURI != cbURL {
					t.Fatalf("unexpected exchange args: code=%q redirectURI=%q", code, redirectURI)
				}
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-cb-1",
					Email:         "cbnew@example.com",
					EmailVerified: true,
					GivenName:     "Cb",
					FamilyName:    "New",
					Picture:       "https://cdn.example/a.jpg",
					Locale:        "en",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-cb-1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleGetByEmailLower).
		WithArgs("cbnew@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLGoogleCreateUser).
		WithArgs("Cb", "New", "cbnew@example.com", sql.NullString{String: "sub-cb-1", Valid: true}, sql.NullString{String: "https://cdn.example/a.jpg", Valid: true}, sql.NullString{String: "en", Valid: true}).
		WillReturnRows(sqlMockGoogleUserFullRow(201, "Cb", "New", "cbnew@example.com", "https://cdn.example/a.jpg", "sub-cb-1", "google", ts))

	q := url.Values{}
	q.Set("code", "google-auth-code")
	q.Set("state", state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if parsed.Scheme != "fucci" || parsed.Host != "auth" {
		t.Fatalf("unexpected redirect base: %q", loc)
	}
	exc := parsed.Query().Get("code")
	if exc == "" {
		t.Fatalf("expected exchange code in redirect, got %q", loc)
	}
	if parsed.Query().Get("is_new") != "1" {
		t.Fatalf("expected is_new=1 for new user, got %q", parsed.RawQuery)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}

	// Exchange endpoint accepts the one-time code exactly once.
	body := map[string]string{"code": exc}
	raw, _ := json.Marshal(body)
	reqEx := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", bytes.NewReader(raw))
	recEx := httptest.NewRecorder()
	cfg.handleGoogleOAuthExchange(recEx, reqEx)
	if recEx.Code != http.StatusOK {
		t.Fatalf("exchange: expected 200, got %d body=%s", recEx.Code, recEx.Body.String())
	}
	var gOut GoogleAuthResponse
	if err := json.Unmarshal(recEx.Body.Bytes(), &gOut); err != nil {
		t.Fatalf("exchange unmarshal: %v", err)
	}
	if gOut.User.Email != "cbnew@example.com" || !gOut.IsNew {
		t.Fatalf("unexpected exchange body: %+v", gOut)
	}
}

func TestHandleGoogleOAuthCallback_SuccessExistingUser_IsNewZero(t *testing.T) {
	_ = InitJWT("test-secret")
	state, err := auth.SignGoogleOAuthState("fucci://oauth/cb")
	if err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cbURL := "https://example.com/cb"
	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "cid",
		GoogleOAuthClientSecret: "sec",
		GoogleOAuthCallbackURL:  cbURL,
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub-ex",
					Email:         "ex@example.com",
					EmailVerified: true,
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLGoogleGetByGoogleID).
		WithArgs("sub-ex").
		WillReturnRows(sqlMockGoogleUserFullRow(77, "Ex", "User", "ex@example.com", "", "sub-ex", "google", ts))
	mock.ExpectQuery(rxSQLGoogleUpdateLogin).
		WithArgs("", int32(77)).
		WillReturnRows(sqlMockGoogleUserFullRow(77, "Ex", "User", "ex@example.com", "", "sub-ex", "google", ts2))

	q := url.Values{}
	q.Set("code", "c1")
	q.Set("state", state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("is_new") != "0" {
		t.Fatalf("expected is_new=0, got %q", parsed.Query().Get("is_new"))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql: %v", err)
	}
}

func TestHandleGoogleOAuthCallback_ProviderErrorRedirectsWithQueryParams(t *testing.T) {
	_ = InitJWT("test-secret")
	state, err := auth.SignGoogleOAuthState("fucci://my-return")
	if err != nil {
		t.Fatal(err)
	}

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := &Config{DBConn: db}

	q := url.Values{}
	q.Set("error", "invalid_request")
	q.Set("error_description", "Consent was revoked")
	q.Set("state", state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("google_error") != "invalid_request" {
		t.Fatalf("google_error=%q", parsed.Query().Get("google_error"))
	}
	gotDesc := parsed.Query().Get("google_error_description")
	if gotDesc != "Consent was revoked" {
		t.Fatalf("google_error_description=%q", gotDesc)
	}
}

func TestHandleGoogleOAuthCallback_AccessDeniedRedirectsCancel(t *testing.T) {
	_ = InitJWT("test-secret")
	state, err := auth.SignGoogleOAuthState("fucci://arena")
	if err != nil {
		t.Fatal(err)
	}

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := &Config{DBConn: db}

	q := url.Values{}
	q.Set("error", "access_denied")
	q.Set("state", state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("cancel") != "1" {
		t.Fatalf("expected cancel=1, got %q", parsed.RawQuery)
	}
	if parsed.Scheme != "fucci" || parsed.Host != "arena" {
		t.Fatalf("unexpected redirect: %s", rec.Header().Get("Location"))
	}
}

func TestHandleGoogleOAuthCallback_AccessDeniedWithoutValidStateUsesDefaultReturn(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := &Config{DBConn: db}

	q := url.Values{}
	q.Set("error", "access_denied")
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.String() != "fucci://auth?cancel=1" && !(parsed.Scheme == "fucci" && parsed.Host == "auth" && parsed.Query().Get("cancel") == "1") {
		t.Fatalf("expected default fucci://auth?cancel=1, got %s", rec.Header().Get("Location"))
	}
}

func TestHandleGoogleOAuthCallback_MissingCodeOrState400(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := &Config{DBConn: db}

	for _, path := range []string{
		"/auth/google/callback?code=only",
		"/auth/google/callback?state=only",
		"/auth/google/callback?",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		cfg.handleGoogleOAuthCallback(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", path, rec.Code)
		}
	}
}

func TestHandleGoogleOAuthCallback_InvalidState400(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := &Config{DBConn: db}

	q := url.Values{}
	q.Set("code", "c")
	q.Set("state", "not-a-valid-jwt")
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGoogleOAuthCallback_MissingCallbackURL503(t *testing.T) {
	_ = InitJWT("test-secret")
	state, err := auth.SignGoogleOAuthState("fucci://auth")
	if err != nil {
		t.Fatal(err)
	}

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "x",
		GoogleOAuthClientSecret: "y",
		// GoogleOAuthCallbackURL intentionally empty
	}

	q := url.Values{}
	q.Set("code", "c")
	q.Set("state", state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGoogleOAuthCallback_ProcErrorRedirectsToApp(t *testing.T) {
	_ = InitJWT("test-secret")
	state, err := auth.SignGoogleOAuthState("fucci://auth")
	if err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:                  db,
		GoogleOAuthClientID:     "cid",
		GoogleOAuthClientSecret: "sec",
		GoogleOAuthCallbackURL:  "https://example/cb",
		GoogleVerifier: &fakeGoogleVerifier{
			exchangeFn: func(ctx context.Context, code, redirectURI string) (string, error) {
				return "id-token", nil
			},
			verifyFn: func(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error) {
				return auth.GoogleIDTokenClaims{
					Subject:       "sub",
					Email:         "u@example.com",
					EmailVerified: false,
				}, nil
			},
		},
	}

	q := url.Values{}
	q.Set("code", "c")
	q.Set("state", state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("google_error") != auth.GoogleAuthEmailNotVerified {
		t.Fatalf("google_error=%q", parsed.Query().Get("google_error"))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql: %v", err)
	}
}

func TestHandleGoogleOAuthExchange_InvalidOrExpiredCode400(t *testing.T) {
	cfg := &Config{}
	body := map[string]string{"code": "definitely-not-issued"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthExchange(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != auth.GoogleAuthCodeInvalid {
		t.Fatalf("code %q want %q", out.Code, auth.GoogleAuthCodeInvalid)
	}
}

func TestHandleGoogleOAuthExchange_MalformedBody400(t *testing.T) {
	cfg := &Config{}
	req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", bytes.NewReader([]byte(`{`)))
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthExchange(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != auth.GoogleAuthCodeInvalid {
		t.Fatalf("expected INVALID_CODE, got %+v", out)
	}
}

func TestHandleGoogleOAuthExchange_ReuseCodeReturns400(t *testing.T) {
	cfg := &Config{}
	in := GoogleAuthResponse{
		Token: "tok",
		User:  UserResponse{ID: 1, Email: "a@b.com"},
		IsNew: false,
	}
	code, err := cfg.issueGoogleOAuthExchangeCode(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}

	body := map[string]string{"code": code}
	raw, _ := json.Marshal(body)
	req1 := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", bytes.NewReader(raw))
	rec1 := httptest.NewRecorder()
	cfg.handleGoogleOAuthExchange(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first exchange: %d %s", rec1.Code, rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", bytes.NewReader(raw))
	rec2 := httptest.NewRecorder()
	cfg.handleGoogleOAuthExchange(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("second exchange: expected 400, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec2.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != auth.GoogleAuthCodeInvalid {
		t.Fatalf("expected INVALID_CODE on reuse, got %+v", out)
	}
}

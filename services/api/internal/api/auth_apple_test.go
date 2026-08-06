package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/DATA-DOG/go-sqlmock"
)

// Regexes match sqlc-generated queries used by appleAuthFromIdentityToken (substring match).
var (
	rxSQLAppleGetByAppleID   = `FROM users WHERE apple_id = \$1::varchar\(255\)`
	rxSQLAppleGetByEmailLower = `FROM users WHERE email = \$1 LIMIT 1`
	rxSQLAppleCreateUser     = `INSERT INTO users \(firstname, lastname, email, apple_id, auth_provider`
	rxSQLAppleUpdateLogin    = `apple_refresh_token = COALESCE\(\$1, apple_refresh_token\)`
	rxSQLAppleLink           = `COALESCE\(NULLIF\(apple_id::text, ''\), \$1::text\)::varchar\(255\)`
)

var sqlAppleAuthUserColumns = []string{
	"id", "firstname", "lastname", "email", "created_at", "updated_at", "is_admin",
	"display_name", "avatar_url", "google_id", "auth_provider", "locale", "last_login_at",
	"is_verified", "is_active", "role", "apple_id", "apple_refresh_token",
}

func sqlMockAppleUserFullRow(id int32, firstname, lastname, email, appleSub, authProv string, ts time.Time) *sqlmock.Rows {
	return sqlmock.NewRows(sqlAppleAuthUserColumns).AddRow(
		id, firstname, lastname, email, ts, ts, false,
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		authProv,
		sql.NullString{},
		sql.NullTime{},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullString{String: "fan", Valid: true},
		sql.NullString{String: appleSub, Valid: appleSub != ""},
		sql.NullString{},
	)
}

func sqlMockAppleUserInactiveRow(id int32, firstname, lastname, email, appleSub, authProv string, ts time.Time) *sqlmock.Rows {
	return sqlmock.NewRows(sqlAppleAuthUserColumns).AddRow(
		id, firstname, lastname, email, ts, ts, false,
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		authProv,
		sql.NullString{},
		sql.NullTime{},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullBool{Bool: false, Valid: true},
		sql.NullString{String: "fan", Valid: true},
		sql.NullString{String: appleSub, Valid: appleSub != ""},
		sql.NullString{},
	)
}

type fakeAppleVerifier struct {
	verifyFn func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error)
}

func (f *fakeAppleVerifier) VerifyIdentityToken(ctx context.Context, identityToken string) (auth.AppleIDTokenClaims, error) {
	return f.verifyFn(ctx, identityToken)
}

func TestHandleAppleAuth_NotConfigured(t *testing.T) {
	cfg := &Config{}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"identity_token": "x"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))

	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "APPLE_NOT_CONFIGURED" {
		t.Fatalf("expected APPLE_NOT_CONFIGURED, got %q", out.Code)
	}
}

func TestHandleAppleAuth_MissingToken(t *testing.T) {
	cfg := &Config{AppleClientID: "com.magistridev.fucci"}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))

	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "INVALID_TOKEN" {
		t.Fatalf("expected INVALID_TOKEN, got %q", out.Code)
	}
}

func TestHandleAppleAuth_InvalidBody(t *testing.T) {
	cfg := &Config{AppleClientID: "com.magistridev.fucci"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader([]byte("{")))

	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "INVALID_BODY" {
		t.Fatalf("expected INVALID_BODY, got %q", out.Code)
	}
}

func TestHandleAppleAuth_InvalidTokenReturns401(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{}, errors.New("bad token")
			},
		},
	}

	body, _ := json.Marshal(map[string]string{"identity_token": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "INVALID_TOKEN" {
		t.Fatalf("expected INVALID_TOKEN, got %q", out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("db should not be queried: %v", err)
	}
}

func TestHandleAppleAuth_NewUserReturnsIsNewTrue(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.new",
					Email:   "newapple@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.new").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleGetByEmailLower).
		WithArgs("newapple@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleCreateUser).
		WithArgs("Ada", "Lovelace", "newapple@example.com", sql.NullString{String: "apple.sub.new", Valid: true}, sql.NullString{}).
		WillReturnRows(sqlMockAppleUserFullRow(201, "Ada", "Lovelace", "newapple@example.com", "apple.sub.new", "apple", ts))

	body, _ := json.Marshal(map[string]any{
		"identity_token": "id-token",
		"full_name": map[string]string{
			"given_name":  "Ada",
			"family_name": "Lovelace",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out AppleAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.IsNew {
		t.Fatal("expected is_new=true")
	}
	if out.Token == "" {
		t.Fatal("expected token")
	}
	if out.User.Email != "newapple@example.com" {
		t.Fatalf("expected email, got %q", out.User.Email)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_NewUserDefaultsNamesWhenMissing(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.default-name",
					Email:   "defaultname@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.default-name").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleGetByEmailLower).
		WithArgs("defaultname@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleCreateUser).
		WithArgs("Fucci", "Fan", "defaultname@example.com", sql.NullString{String: "apple.sub.default-name", Valid: true}, sql.NullString{}).
		WillReturnRows(sqlMockAppleUserFullRow(202, "Fucci", "Fan", "defaultname@example.com", "apple.sub.default-name", "apple", ts))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out AppleAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.IsNew || out.User.Firstname != "Fucci" || out.User.Lastname != "Fan" {
		t.Fatalf("expected default Fucci Fan new user, got is_new=%v name=%q %q", out.IsNew, out.User.Firstname, out.User.Lastname)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_ExistingAppleUserReturnsIsNewFalse(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.existing",
					Email:   "existing@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.existing").
		WillReturnRows(sqlMockAppleUserFullRow(42, "Existing", "User", "existing@example.com", "apple.sub.existing", "apple", ts))
	mock.ExpectQuery(rxSQLAppleUpdateLogin).
		WithArgs(sql.NullString{}, int32(42)).
		WillReturnRows(sqlMockAppleUserFullRow(42, "Existing", "User", "existing@example.com", "apple.sub.existing", "apple", ts2))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out AppleAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.IsNew {
		t.Fatal("expected is_new=false")
	}
	if out.User.Email != "existing@example.com" {
		t.Fatalf("expected existing email, got %q", out.User.Email)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_InactiveUserReturns403(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.inactive",
					Email:   "inactive@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.inactive").
		WillReturnRows(sqlMockAppleUserInactiveRow(42, "In", "Active", "inactive@example.com", "apple.sub.inactive", "apple", ts))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "ACCOUNT_INACTIVE" {
		t.Fatalf("expected ACCOUNT_INACTIVE, got %q", out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_NewUserWithoutEmailReturns400(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.no-email",
					Email:   "",
				}, nil
			},
		},
	}

	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.no-email").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "EMAIL_REQUIRED" {
		t.Fatalf("expected EMAIL_REQUIRED, got %q", out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_EmailPasswordAccountReturns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.new",
					Email:   "password-only@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.new").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleGetByEmailLower).
		WithArgs("password-only@example.com").
		WillReturnRows(sqlMockAppleUserFullRow(55, "Pass", "User", "password-only@example.com", "", "email", ts))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "ACCOUNT_EXISTS_EMAIL" {
		t.Fatalf("expected ACCOUNT_EXISTS_EMAIL, got %q", out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_EmailLinkedToOtherAppleReturns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "incoming-apple-sub",
					Email:   "taken@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("incoming-apple-sub").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleGetByEmailLower).
		WithArgs("taken@example.com").
		WillReturnRows(sqlMockAppleUserFullRow(77, "Other", "Apple", "taken@example.com", "different-apple-sub", "apple", ts))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "ACCOUNT_EXISTS_EMAIL" {
		t.Fatalf("expected ACCOUNT_EXISTS_EMAIL, got %q", out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_EmailFallbackNonAppleProviderReturns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "incoming-apple-sub",
					Email:   "google-user@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("incoming-apple-sub").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleGetByEmailLower).
		WithArgs("google-user@example.com").
		WillReturnRows(sqlmock.NewRows(sqlAppleAuthUserColumns).AddRow(
			int32(88), "", "", "google-user@example.com", ts, ts, false,
			sql.NullString{}, sql.NullString{},
			sql.NullString{String: "google-sub", Valid: true}, "google",
			sql.NullString{}, sql.NullTime{},
			sql.NullBool{Bool: true, Valid: true},
			sql.NullBool{Bool: true, Valid: true},
			sql.NullString{String: "fan", Valid: true},
			sql.NullString{}, sql.NullString{},
		))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out apiErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Code != "ACCOUNT_EXISTS_EMAIL" {
		t.Fatalf("expected ACCOUNT_EXISTS_EMAIL, got %q", out.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHandleAppleAuth_LinksExistingAppleProviderUser(t *testing.T) {
	_ = InitJWT("test-secret")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.link",
					Email:   "linkme@example.com",
				}, nil
			},
		},
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.link").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(rxSQLAppleGetByEmailLower).
		WithArgs("linkme@example.com").
		WillReturnRows(sqlMockAppleUserFullRow(99, "Link", "Me", "linkme@example.com", "", "apple", ts))
	mock.ExpectQuery(rxSQLAppleLink).
		WithArgs("apple.sub.link", sql.NullString{}, int32(99)).
		WillReturnRows(sqlMockAppleUserFullRow(99, "Link", "Me", "linkme@example.com", "apple.sub.link", "apple", ts))

	body, _ := json.Marshal(map[string]string{"identity_token": "id-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out AppleAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.IsNew {
		t.Fatal("expected is_new=false for linked account")
	}
	if out.User.ID != 99 {
		t.Fatalf("expected linked user id 99, got %d", out.User.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAppleAuthFromIdentityToken_LookupFailureReturns500(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		DBConn:        db,
		AppleClientID: "com.magistridev.fucci",
		AppleVerifier: &fakeAppleVerifier{
			verifyFn: func(ctx context.Context, token string) (auth.AppleIDTokenClaims, error) {
				return auth.AppleIDTokenClaims{
					Subject: "apple.sub.err",
					Email:   "err@example.com",
				}, nil
			},
		},
	}

	mock.ExpectQuery(rxSQLAppleGetByAppleID).
		WithArgs("apple.sub.err").
		WillReturnError(errors.New("db down"))

	_, procErr := cfg.appleAuthFromIdentityToken(context.Background(), AppleAuthRequest{IdentityToken: "tok"})
	if procErr == nil {
		t.Fatal("expected procErr")
	}
	if procErr.status != http.StatusInternalServerError || procErr.code != "UPSTREAM" {
		t.Fatalf("got status=%d code=%q", procErr.status, procErr.code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

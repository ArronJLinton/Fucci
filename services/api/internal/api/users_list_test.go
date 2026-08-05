package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleListAllUsers_OmitsAppleRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	secret := "apple-refresh-secret-should-never-leak"
	rows := sqlmock.NewRows(sqlAppleAuthUserColumns).AddRow(
		int32(7), "Ada", "Lovelace", "ada@example.com", ts, ts, false,
		sql.NullString{},
		sql.NullString{},
		sql.NullString{},
		"apple",
		sql.NullString{},
		sql.NullTime{},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullBool{Bool: true, Valid: true},
		sql.NullString{String: "fan", Valid: true},
		sql.NullString{String: "apple.sub.7", Valid: true},
		sql.NullString{String: secret, Valid: true},
	)
	mock.ExpectQuery(`FROM users ORDER BY created_at DESC`).WillReturnRows(rows)

	cfg := &Config{DB: database.New(db)}
	req := httptest.NewRequest(http.MethodGet, "/users/all", nil)
	rec := httptest.NewRecorder()
	cfg.handleListAllUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, secret) {
		t.Fatalf("response leaked apple refresh token: %s", body)
	}
	if strings.Contains(body, "apple_refresh_token") || strings.Contains(body, "AppleRefreshToken") {
		t.Fatalf("response included refresh-token field name: %s", body)
	}

	var out []UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, body)
	}
	if len(out) != 1 || out[0].Email != "ada@example.com" {
		t.Fatalf("unexpected list payload: %+v", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUsersModel_AppleRefreshTokenJSONOmitted(t *testing.T) {
	u := database.Users{
		ID:                1,
		Email:             "a@b.com",
		AppleRefreshToken: sql.NullString{String: "secret-token", Valid: true},
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "secret-token") {
		t.Fatalf("Users JSON leaked AppleRefreshToken: %s", raw)
	}
}

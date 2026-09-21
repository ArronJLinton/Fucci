package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestCommunityMutationsRequireAuthentication(t *testing.T) {
	h := New(&Config{})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create team", method: http.MethodPost, path: "/teams", body: `{"name":"X","league_id":"00000000-0000-0000-0000-000000000001"}`},
		{name: "update team", method: http.MethodPut, path: "/teams/00000000-0000-0000-0000-000000000001", body: `{}`},
		{name: "delete team", method: http.MethodDelete, path: "/teams/00000000-0000-0000-0000-000000000001"},
		{name: "create league", method: http.MethodPost, path: "/leagues", body: `{"name":"X"}`},
		{name: "update league", method: http.MethodPut, path: "/leagues/00000000-0000-0000-0000-000000000001", body: `{}`},
		{name: "delete league", method: http.MethodDelete, path: "/leagues/00000000-0000-0000-0000-000000000001"},
		{name: "create team manager", method: http.MethodPost, path: "/team-managers", body: `{"user_id":1,"league_id":"00000000-0000-0000-0000-000000000001"}`},
		{name: "update team manager", method: http.MethodPut, path: "/team-managers/00000000-0000-0000-0000-000000000001", body: `{}`},
		{name: "delete team manager", method: http.MethodDelete, path: "/team-managers/00000000-0000-0000-0000-000000000001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("code=%d body=%s, want 401", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCommunityMutationsRejectNonAdmins(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create team", method: http.MethodPost, path: "/teams", body: `{"name":"X","league_id":"00000000-0000-0000-0000-000000000001"}`},
		{name: "create league", method: http.MethodPost, path: "/leagues", body: `{"name":"X"}`},
		{name: "create team manager", method: http.MethodPost, path: "/team-managers", body: `{"user_id":1,"league_id":"00000000-0000-0000-0000-000000000001"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			const uid int32 = 42
			mock.ExpectQuery(testSQLGetUserForDebateAdmin).
				WithArgs(uid).
				WillReturnRows(debateAdminUserRows(uid, false, database.UserRoleFan))

			h := New(&Config{DB: database.New(db), DBConn: db})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+adminAuthzTestToken(t, uid, string(database.UserRoleFan)))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("code=%d body=%s, want 403", rec.Code, rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("db expectations: %v", err)
			}
		})
	}
}

func TestListAllUsersRequiresAdmin(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		h := New(&Config{})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users/all", nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("code=%d body=%s, want 401", rec.Code, rec.Body.String())
		}
	})

	t.Run("non-admin authenticated", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		const uid int32 = 99
		mock.ExpectQuery(testSQLGetUserForDebateAdmin).
			WithArgs(uid).
			WillReturnRows(debateAdminUserRows(uid, false, database.UserRoleFan))

		h := New(&Config{DB: database.New(db), DBConn: db})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users/all", nil)
		req.Header.Set("Authorization", "Bearer "+adminAuthzTestToken(t, uid, string(database.UserRoleFan)))
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("code=%d body=%s, want 403", rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("db expectations: %v", err)
		}
	})

	t.Run("admin authenticated", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		const uid int32 = 7
		mock.ExpectQuery(testSQLGetUserForDebateAdmin).
			WithArgs(uid).
			WillReturnRows(debateAdminUserRows(uid, true, database.UserRoleAdmin))
		mock.ExpectQuery(`-- name: ListUsers :many
SELECT id, firstname, lastname, email, created_at, updated_at, is_admin, display_name, avatar_url, google_id, auth_provider, locale, last_login_at, is_verified, is_active, role FROM users ORDER BY created_at DESC`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "firstname", "lastname", "email", "created_at", "updated_at", "is_admin",
				"display_name", "avatar_url", "google_id", "auth_provider", "locale", "last_login_at",
				"is_verified", "is_active", "role",
			}))

		h := New(&Config{DB: database.New(db), DBConn: db})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users/all", nil)
		req.Header.Set("Authorization", "Bearer "+adminAuthzTestToken(t, uid, string(database.UserRoleAdmin)))
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s, want 200", rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("db expectations: %v", err)
		}
	})
}

func adminAuthzTestToken(t *testing.T, userID int32, role string) string {
	t.Helper()
	if err := auth.InitJWTAuth("admin-authz-test-secret"); err != nil {
		t.Fatal(err)
	}
	token, err := auth.GenerateToken(userID, "admin-authz@example.com", role, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

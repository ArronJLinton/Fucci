package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/moderation"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAcceptedTerms(t *testing.T) {
	t.Parallel()
	require.Error(t, requireAcceptedTerms(false))
	require.NoError(t, requireAcceptedTerms(true))
}

func TestFilterStoriesByBlocked(t *testing.T) {
	t.Parallel()
	stories := []userStoryPayload{
		{ID: "a", UserID: 1},
		{ID: "b", UserID: 2},
		{ID: "c", UserID: 3},
	}
	got := filterStoriesByBlocked(stories, map[int32]struct{}{2: {}})
	require.Len(t, got, 2)
	assert.Equal(t, int32(1), got[0].UserID)
	assert.Equal(t, int32(3), got[1].UserID)

	assert.Equal(t, stories, filterStoriesByBlocked(stories, nil))
	assert.Equal(t, stories, filterStoriesByBlocked(stories, map[int32]struct{}{}))
}

func TestHandleLogin_TermsRequired(t *testing.T) {
	cfg := &Config{}
	body, _ := json.Marshal(map[string]string{
		"email":    "fan@example.com",
		"password": "Password1",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleLogin(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "TERMS_REQUIRED", out.Code)
}

func TestHandleLogin_DeactivatedAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1 AND COALESCE\(is_active, true\) = false`).
		WithArgs("banned@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int32(9)))

	cfg := &Config{DBConn: db}
	body, _ := json.Marshal(map[string]any{
		"email":          "banned@example.com",
		"password":       "Password1",
		"accepted_terms": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleLogin(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, auth.GoogleAuthAccountInactive, out.Code)
	assert.Contains(t, out.Error, "deactivated")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleCreateUser_TermsRequired(t *testing.T) {
	cfg := &Config{}
	body, _ := json.Marshal(map[string]string{
		"firstname": "Ada",
		"lastname":  "Lovelace",
		"email":     "ada@example.com",
		"password":  "Password1",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleCreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "TERMS_REQUIRED", out.Code)
}

func TestHandleAppleAuth_TermsRequired(t *testing.T) {
	cfg := &Config{AppleClientID: "com.magistridev.fucci"}
	body, _ := json.Marshal(map[string]string{"identity_token": "tok"})
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleAppleAuth(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "TERMS_REQUIRED", out.Code)
}

func TestHandleGoogleOAuthExchange_TermsRequired(t *testing.T) {
	cfg := &Config{}
	body, _ := json.Marshal(map[string]string{"code": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	cfg.handleGoogleOAuthExchange(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "TERMS_REQUIRED", out.Code)
}

func TestCreateDebateComment_ObjectionableContent(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CommentReader: &mockCommentReader{
			getDebateFunc: func(ctx context.Context, id int32) (database.Debates, error) {
				return database.Debates{ID: id}, nil
			},
		},
	}
	req := commentRequestWithChiParams(
		"POST",
		"/debates/1/comments",
		CreateDebateCommentRequest{Content: "what the fuck"},
		map[string]string{"debateId": "1"},
		&userID,
	)
	rec := httptest.NewRecorder()
	config.CreateDebateComment(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "OBJECTIONABLE_CONTENT", out.Code)
}

func TestPostContentReport_AvatarOK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{
		DB:                 database.New(db),
		ModerationNotifier: &moderation.Notifier{}, // SMTP unset → sync log path
	}
	const reporterID int32 = 99
	const targetID int32 = 10
	reportID := uuid.New()
	createdAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	ts := createdAt

	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(reporterID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))
	mock.ExpectQuery(`FROM users WHERE id = \$1`).
		WithArgs(targetID).
		WillReturnRows(sqlMockAppleUserFullRow(targetID, "T", "User", "t@example.com", "", "email", ts))
	mock.ExpectQuery(`(?s)-- name: CreateContentReport :one\s+INSERT INTO content_reports .*RETURNING id, reporter_id, reportable_type, reportable_id, reason, description, status, created_at, reported_user_id`).
		WithArgs(reporterID, "avatar", strconv.FormatInt(int64(targetID), 10), sql.NullInt32{Int32: targetID, Valid: true}, "inappropriate_content", sql.NullString{}).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "reportable_type", "reportable_id", "reason", "description", "status", "created_at", "reported_user_id",
		}).AddRow(
			reportID, reporterID, "avatar", strconv.FormatInt(int64(targetID), 10),
			"inappropriate_content", nil, "pending", createdAt, targetID,
		))

	body, _ := json.Marshal(map[string]string{
		"reportable_type": "avatar",
		"reportable_id":   strconv.FormatInt(int64(targetID), 10),
		"reason":          "inappropriate_content",
	})
	req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: reporterID}))
	rec := httptest.NewRecorder()
	cfg.postContentReport(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "avatar", out["reportable_type"])
	assert.EqualValues(t, targetID, out["reported_user_id"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostContentReport_PlayerProfileOK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{
		DB:                 database.New(db),
		ModerationNotifier: &moderation.Notifier{},
	}
	const reporterID int32 = 99
	const targetID int32 = 42
	reportID := uuid.New()
	createdAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(reporterID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))
	mock.ExpectQuery(`FROM users WHERE id = \$1`).
		WithArgs(targetID).
		WillReturnRows(sqlMockAppleUserFullRow(targetID, "P", "Pro", "p@example.com", "", "email", createdAt))
	mock.ExpectQuery(`(?s)-- name: CreateContentReport :one\s+INSERT INTO content_reports .*RETURNING id, reporter_id, reportable_type, reportable_id, reason, description, status, created_at, reported_user_id`).
		WithArgs(reporterID, "player_profile", strconv.FormatInt(int64(targetID), 10), sql.NullInt32{Int32: targetID, Valid: true}, "harassment", sql.NullString{}).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "reportable_type", "reportable_id", "reason", "description", "status", "created_at", "reported_user_id",
		}).AddRow(
			reportID, reporterID, "player_profile", strconv.FormatInt(int64(targetID), 10),
			"harassment", nil, "pending", createdAt, targetID,
		))

	body, _ := json.Marshal(map[string]string{
		"reportable_type": "player_profile",
		"reportable_id":   strconv.FormatInt(int64(targetID), 10),
		"reason":          "harassment",
	})
	req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: reporterID}))
	rec := httptest.NewRecorder()
	cfg.postContentReport(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "player_profile", out["reportable_type"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostUserBlock_CreatesReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{
		DB:                 database.New(db),
		ModerationNotifier: &moderation.Notifier{},
	}
	const blockerID int32 = 7
	const blockedID int32 = 42
	blockID := uuid.New()
	reportID := uuid.New()
	createdAt := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(blockerID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))
	mock.ExpectQuery(`FROM users WHERE id = \$1`).
		WithArgs(blockedID).
		WillReturnRows(sqlMockAppleUserFullRow(blockedID, "B", "Locked", "b@example.com", "", "email", createdAt))
	mock.ExpectQuery(`(?s)-- name: CreateUserBlock :one\s+INSERT INTO user_blocks .*RETURNING id, blocker_id, blocked_user_id, created_at`).
		WithArgs(blockerID, blockedID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "blocker_id", "blocked_user_id", "created_at"}).
			AddRow(blockID, blockerID, blockedID, createdAt))
	mock.ExpectQuery(`(?s)-- name: CreateContentReport :one\s+INSERT INTO content_reports .*RETURNING id, reporter_id, reportable_type, reportable_id, reason, description, status, created_at, reported_user_id`).
		WithArgs(
			blockerID,
			"user",
			strconv.FormatInt(int64(blockedID), 10),
			sql.NullInt32{Int32: blockedID, Valid: true},
			"harassment",
			sql.NullString{String: "Auto-created when user was blocked", Valid: true},
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "reportable_type", "reportable_id", "reason", "description", "status", "created_at", "reported_user_id",
		}).AddRow(
			reportID, blockerID, "user", strconv.FormatInt(int64(blockedID), 10), "harassment",
			"Auto-created when user was blocked", "pending", createdAt, blockedID,
		))

	body, _ := json.Marshal(map[string]any{
		"blocked_user_id": blockedID,
		"reason":          "harassment",
	})
	req := httptest.NewRequest(http.MethodPost, "/users/blocks", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: blockerID}))
	rec := httptest.NewRecorder()
	cfg.postUserBlock(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, blockID.String(), out["id"])
	assert.Equal(t, reportID.String(), out["report_id"])
	assert.EqualValues(t, blockedID, out["blocked_user_id"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostUserBlock_CannotBlockSelf(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const uid int32 = 7
	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

	body, _ := json.Marshal(map[string]any{"blocked_user_id": uid})
	req := httptest.NewRequest(http.MethodPost, "/users/blocks", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: uid}))
	rec := httptest.NewRecorder()
	cfg.postUserBlock(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostUserBlock_RejectsInjectedReportableType(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const blockerID int32 = 7
	const blockedID int32 = 42

	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(blockerID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

	// CR/LF in reportable_type must be rejected by allowlist before any DB writes.
	body, _ := json.Marshal(map[string]any{
		"blocked_user_id": blockedID,
		"reason":          "harassment",
		"reportable_type": "user\r\nContent-Type: text/html\r\n\r\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/users/blocks", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: blockerID}))
	rec := httptest.NewRecorder()
	cfg.postUserBlock(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out.Error, "reportable_type")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsAllowedReportableType(t *testing.T) {
	t.Parallel()
	assert.True(t, isAllowedReportableType("user"))
	assert.True(t, isAllowedReportableType("story"))
	assert.False(t, isAllowedReportableType("user\r\nBcc: evil@x.test"))
	assert.False(t, isAllowedReportableType("not-a-type"))
}

func TestDeleteUserBlock_OK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const blockerID int32 = 7
	const blockedID int32 = 42

	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(blockerID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))
	mock.ExpectExec(`(?s)-- name: DeleteUserBlock :exec\s+DELETE FROM user_blocks`).
		WithArgs(blockerID, blockedID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/users/blocks/"+strconv.Itoa(int(blockedID)), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("blockedUserId", strconv.Itoa(int(blockedID)))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: blockerID}))
	rec := httptest.NewRecorder()
	cfg.deleteUserBlock(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

type stubContentChecker struct {
	result moderation.ContentCheckResult
	err    error
	calls  int
}

func (s *stubContentChecker) Check(ctx context.Context, input moderation.ContentCheckInput) (moderation.ContentCheckResult, error) {
	s.calls++
	return s.result, s.err
}

func TestSoftFlagUGC_CreatesReportAndNotifies(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	checker := &stubContentChecker{
		result: moderation.ContentCheckResult{
			Flagged:    true,
			Categories: []string{"hate"},
			Model:      "omni-moderation-latest",
		},
	}
	cfg := &Config{
		DB:                 database.New(db),
		ContentChecker:     checker,
		SystemUserEmail:    "contact@magistri.dev",
		ModerationNotifier: &moderation.Notifier{},
	}
	const authorID int32 = 42
	const systemID int32 = 1
	reportID := uuid.New()
	createdAt := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs("contact@magistri.dev").
		WillReturnRows(systemUserMockRows(systemID, "contact@magistri.dev"))
	mock.ExpectQuery(`(?s)-- name: CreateContentReport :one\s+INSERT INTO content_reports .*RETURNING id, reporter_id, reportable_type, reportable_id, reason, description, status, created_at, reported_user_id`).
		WithArgs(
			systemID,
			"debate_response",
			"99",
			sql.NullInt32{Int32: authorID, Valid: true},
			"inappropriate_content",
			sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "reportable_type", "reportable_id", "reason", "description", "status", "created_at", "reported_user_id",
		}).AddRow(
			reportID, systemID, "debate_response", "99", "inappropriate_content",
			"AI soft-flag", "pending", createdAt, authorID,
		))

	cfg.softFlagUGC(context.Background(), authorID, "debate_response", "99", []string{"bad text"}, "", "")
	assert.Equal(t, 1, checker.calls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSoftFlagUGC_NotFlaggedNoReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	checker := &stubContentChecker{result: moderation.ContentCheckResult{Flagged: false}}
	cfg := &Config{DB: database.New(db), ContentChecker: checker}
	cfg.softFlagUGC(context.Background(), 7, "debate_response", "1", []string{"hello"}, "", "")
	assert.Equal(t, 1, checker.calls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClampModerationDescription(t *testing.T) {
	t.Parallel()
	got, err := clampModerationDescription("  hello  ")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	_, err = clampModerationDescription(strings.Repeat("a", moderationReportDescMaxLen+1))
	require.Error(t, err)
}

func TestPostContentReport_DescriptionTooLong(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const reporterID int32 = 55
	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(reporterID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))

	long := strings.Repeat("x", moderationReportDescMaxLen+1)
	body, _ := json.Marshal(map[string]any{
		"reportable_type": "user",
		"reportable_id":   "10",
		"reason":          "spam",
		"description":     long,
	})
	req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: reporterID}))
	rec := httptest.NewRecorder()
	cfg.postContentReport(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostUserBlock_RejectsMismatchedReportableOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const blockerID int32 = 7
	const blockedID int32 = 42
	const otherUserID int32 = 99

	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(blockerID).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(true))
	mock.ExpectQuery(`FROM users WHERE id = \$1`).
		WithArgs(otherUserID).
		WillReturnRows(sqlMockAppleUserFullRow(otherUserID, "O", "Ther", "o@example.com", "", "email", time.Now().UTC()))

	body, _ := json.Marshal(map[string]any{
		"blocked_user_id": blockedID,
		"reportable_type": "user",
		"reportable_id":   strconv.FormatInt(int64(otherUserID), 10),
		"reason":          "harassment",
	})
	req := httptest.NewRequest(http.MethodPost, "/users/blocks", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: blockerID}))
	rec := httptest.NewRecorder()
	cfg.postUserBlock(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out.Error, "belong to the blocked user")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequireActiveAuthedUser_Deactivated(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const uid int32 = 12
	mock.ExpectQuery(`SELECT COALESCE\(is_active, TRUE\)::bool AS is_active`).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"is_active"}).AddRow(false))

	req := httptest.NewRequest(http.MethodPost, "/reports", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: uid}))
	rec := httptest.NewRecorder()
	_, ok := cfg.requireActiveAuthedUser(rec, req)
	assert.False(t, ok)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, auth.GoogleAuthAccountInactive, out.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRejectIfObjectionable_HateSpeech(t *testing.T) {
	cfg := &Config{}
	rec := httptest.NewRecorder()
	blocked := cfg.rejectIfObjectionable(rec, "heil hitler")
	assert.True(t, blocked)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out apiErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "OBJECTIONABLE_CONTENT", out.Code)
}

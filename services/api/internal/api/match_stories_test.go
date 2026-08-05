package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMatchStoryTeamLookup(t *testing.T) {
	match := &MatchInfo{HomeTeam: "Spain", AwayTeam: "Croatia"}
	homeKey := youtube.LookupKeyForTeamName("Spain")

	if err := validateMatchStoryTeamLookup("match", "123", homeKey, match); err != nil {
		t.Fatalf("expected valid home team key, got %v", err)
	}
	if err := validateMatchStoryTeamLookup("match", "123", "france", match); err == nil {
		t.Fatal("expected error for team not in match")
	}
	if err := validateMatchStoryTeamLookup("tournament", "123", homeKey, match); err == nil {
		t.Fatal("expected error for unsupported scope in v1")
	}
}

func TestCloudinaryConfigForMatchStoryContexts(t *testing.T) {
	photo, ok := cloudinaryConfigForContext("match_story_photo")
	if !ok || photo.ResourceType != "image" {
		t.Fatalf("unexpected photo config: %+v ok=%v", photo, ok)
	}
	video, ok := cloudinaryConfigForContext("match_story_video")
	if !ok || video.ResourceType != "video" || video.MaxBytes <= photo.MaxBytes {
		t.Fatalf("unexpected video config: %+v ok=%v", video, ok)
	}
}

type stubMatchStoryStore struct {
	GetMatchStoryByIDFn    func(ctx context.Context, id uuid.UUID) (database.MatchStories, error)
	DeactivateMatchStoryFn func(ctx context.Context, id uuid.UUID) (database.MatchStories, error)
}

func (s *stubMatchStoryStore) GetMatchStoryByID(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
	if s.GetMatchStoryByIDFn != nil {
		return s.GetMatchStoryByIDFn(ctx, id)
	}
	return database.MatchStories{}, assert.AnError
}

func (s *stubMatchStoryStore) DeactivateMatchStory(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
	if s.DeactivateMatchStoryFn != nil {
		return s.DeactivateMatchStoryFn(ctx, id)
	}
	return database.MatchStories{}, assert.AnError
}

func matchStoryDeleteTestRequest(storyID uuid.UUID, userID int32) *http.Request {
	r := httptest.NewRequest(http.MethodDelete, "/stories/"+storyID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", storyID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	if userID != 0 {
		r = r.WithContext(auth.ContextWithClaims(r.Context(), &auth.JWTClaims{UserID: userID}))
	}
	return r
}

func matchStoryReportTestRequest(storyID uuid.UUID, userID int32) *http.Request {
	body, _ := json.Marshal(map[string]string{
		"reportable_type": "story",
		"reportable_id":   storyID.String(),
		"reason":          "spam",
	})
	r := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewReader(body))
	if userID != 0 {
		r = r.WithContext(auth.ContextWithClaims(r.Context(), &auth.JWTClaims{UserID: userID}))
	}
	return r
}

func debateCommentReportTestRequest(commentID int32, userID int32) *http.Request {
	body, _ := json.Marshal(map[string]string{
		"reportable_type": "debate_response",
		"reportable_id":   strconv.FormatInt(int64(commentID), 10),
		"reason":          "spam",
	})
	r := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewReader(body))
	if userID != 0 {
		r = r.WithContext(auth.ContextWithClaims(r.Context(), &auth.JWTClaims{UserID: userID}))
	}
	return r
}

func sampleMatchStory(id uuid.UUID, userID int32, active bool) database.MatchStories {
	return database.MatchStories{
		ID:            id,
		UserID:        userID,
		ScopeType:     database.StoryScopeTypeMatch,
		ScopeID:       "12345",
		TeamLookupKey: "spain",
		ContentType:   database.StoryContentTypePhoto,
		MediaUrl:      "https://res.cloudinary.com/demo/image/upload/v1/story.jpg",
		IsActive:      active,
		CreatedAt:     time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC),
	}
}

func TestDeleteMatchStory_Unauthenticated(t *testing.T) {
	cfg := &Config{MatchStoryDB: &stubMatchStoryStore{}}
	rec := httptest.NewRecorder()
	cfg.deleteMatchStory(rec, matchStoryDeleteTestRequest(uuid.New(), 0))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteMatchStory_NotFound(t *testing.T) {
	storyID := uuid.New()
	stub := &stubMatchStoryStore{
		GetMatchStoryByIDFn: func(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
			assert.Equal(t, storyID, id)
			return database.MatchStories{}, sql.ErrNoRows
		},
	}
	cfg := &Config{MatchStoryDB: stub}
	rec := httptest.NewRecorder()
	cfg.deleteMatchStory(rec, matchStoryDeleteTestRequest(storyID, 1))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteMatchStory_Forbidden(t *testing.T) {
	storyID := uuid.New()
	ownerID := int32(10)
	stub := &stubMatchStoryStore{
		GetMatchStoryByIDFn: func(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
			return sampleMatchStory(id, ownerID, true), nil
		},
	}
	cfg := &Config{MatchStoryDB: stub}
	rec := httptest.NewRecorder()
	cfg.deleteMatchStory(rec, matchStoryDeleteTestRequest(storyID, 99))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteMatchStory_OK(t *testing.T) {
	storyID := uuid.New()
	ownerID := int32(10)
	var deactivatedID uuid.UUID
	stub := &stubMatchStoryStore{
		GetMatchStoryByIDFn: func(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
			return sampleMatchStory(id, ownerID, true), nil
		},
		DeactivateMatchStoryFn: func(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
			deactivatedID = id
			row := sampleMatchStory(id, ownerID, false)
			return row, nil
		},
	}
	cfg := &Config{MatchStoryDB: stub}
	rec := httptest.NewRecorder()
	cfg.deleteMatchStory(rec, matchStoryDeleteTestRequest(storyID, ownerID))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, storyID, deactivatedID)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "removed", out["status"])
}

func TestDeleteMatchStory_AlreadyRemoved(t *testing.T) {
	storyID := uuid.New()
	ownerID := int32(10)
	stub := &stubMatchStoryStore{
		GetMatchStoryByIDFn: func(ctx context.Context, id uuid.UUID) (database.MatchStories, error) {
			return sampleMatchStory(id, ownerID, false), nil
		},
	}
	cfg := &Config{MatchStoryDB: stub}
	rec := httptest.NewRecorder()
	cfg.deleteMatchStory(rec, matchStoryDeleteTestRequest(storyID, ownerID))

	assert.Equal(t, http.StatusOK, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, "already_removed", out["status"])
}

func TestPostContentReport_DoesNotDeactivateStory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	storyID := uuid.New()
	reportID := uuid.New()
	const reporterID int32 = 99
	const ownerID int32 = 10
	createdAt := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT id, user_id, scope_type, scope_id, team_lookup_key, content_type, media_url, caption, is_active, created_at FROM match_stories WHERE id = \$1`).
		WithArgs(storyID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "scope_type", "scope_id", "team_lookup_key", "content_type", "media_url", "caption", "is_active", "created_at",
		}).AddRow(
			storyID,
			ownerID,
			string(database.StoryScopeTypeMatch),
			"12345",
			"spain",
			string(database.StoryContentTypePhoto),
			"https://res.cloudinary.com/demo/image/upload/v1/story.jpg",
			nil,
			true,
			createdAt,
		))
	mock.ExpectQuery(`(?s)-- name: CreateContentReport :one\s+INSERT INTO content_reports .*RETURNING id, reporter_id, reportable_type, reportable_id, reason, description, status, created_at, reported_user_id`).
		WithArgs(reporterID, "story", storyID.String(), ownerID, "spam", sql.NullString{}).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "reportable_type", "reportable_id", "reason", "description", "status", "created_at", "reported_user_id",
		}).AddRow(
			reportID,
			reporterID,
			"story",
			storyID.String(),
			"spam",
			nil,
			"pending",
			createdAt,
			ownerID,
		))

	rec := httptest.NewRecorder()
	cfg.postContentReport(rec, matchStoryReportTestRequest(storyID, reporterID))

	assert.Equal(t, http.StatusOK, rec.Code)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, reportID.String(), out["id"])
	assert.Equal(t, false, out["story_deactivated"])
	assert.EqualValues(t, ownerID, out["reported_user_id"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostContentReport_DebateResponseOK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const commentID int32 = 42
	const reporterID int32 = 99
	const authorID int32 = 10
	reportID := uuid.New()
	createdAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT\s+c\.id, c\.debate_id, c\.parent_comment_id, c\.user_id, c\.content, c\.created_at, c\.updated_at, c\.seeded`).
		WithArgs(commentID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "debate_id", "parent_comment_id", "user_id", "content", "created_at", "updated_at", "seeded",
			"firstname", "lastname", "display_name", "avatar_url",
		}).AddRow(
			commentID,
			1,
			nil,
			authorID,
			"test comment",
			createdAt,
			createdAt,
			false,
			"Test",
			"User",
			nil,
			nil,
		))
	mock.ExpectQuery(`(?s)-- name: CreateContentReport :one\s+INSERT INTO content_reports .*RETURNING id, reporter_id, reportable_type, reportable_id, reason, description, status, created_at, reported_user_id`).
		WithArgs(reporterID, "debate_response", strconv.FormatInt(int64(commentID), 10), authorID, "spam", sql.NullString{}).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "reporter_id", "reportable_type", "reportable_id", "reason", "description", "status", "created_at", "reported_user_id",
		}).AddRow(
			reportID,
			reporterID,
			"debate_response",
			strconv.FormatInt(int64(commentID), 10),
			"spam",
			nil,
			"pending",
			createdAt,
			authorID,
		))

	rec := httptest.NewRecorder()
	cfg.postContentReport(rec, debateCommentReportTestRequest(commentID, reporterID))

	assert.Equal(t, http.StatusOK, rec.Code)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, reportID.String(), out["id"])
	assert.Equal(t, "debate_response", out["reportable_type"])
	assert.EqualValues(t, authorID, out["reported_user_id"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostContentReport_CannotReportOwnComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &Config{DB: database.New(db)}
	const commentID int32 = 42
	const userID int32 = 10
	createdAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT\s+c\.id, c\.debate_id, c\.parent_comment_id, c\.user_id, c\.content, c\.created_at, c\.updated_at, c\.seeded`).
		WithArgs(commentID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "debate_id", "parent_comment_id", "user_id", "content", "created_at", "updated_at", "seeded",
			"firstname", "lastname", "display_name", "avatar_url",
		}).AddRow(
			commentID,
			1,
			nil,
			userID,
			"my comment",
			createdAt,
			createdAt,
			false,
			"Test",
			"User",
			nil,
			nil,
		))

	rec := httptest.NewRecorder()
	cfg.postContentReport(rec, debateCommentReportTestRequest(commentID, userID))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

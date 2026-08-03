package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCardVoteReader implements CardVoteReader for tests.
type mockCardVoteReader struct {
	getUserFunc       func(ctx context.Context, id int32) (database.Users, error)
	getDebateCardFunc func(ctx context.Context, id int32) (database.DebateCards, error)
}

func (m *mockCardVoteReader) GetUser(ctx context.Context, id int32) (database.Users, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, id)
	}
	return database.Users{}, sql.ErrNoRows
}

func (m *mockCardVoteReader) GetDebateCard(ctx context.Context, id int32) (database.DebateCards, error) {
	if m.getDebateCardFunc != nil {
		return m.getDebateCardFunc(ctx, id)
	}
	return database.DebateCards{}, sql.ErrNoRows
}

// requestWithChiParams builds a request with chi URL params and optional JWT user id in context.
func requestWithChiParams(method, path string, body interface{}, urlParams map[string]string, userID *int32) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, nil)
	if len(bodyBytes) > 0 {
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	if len(urlParams) > 0 {
		rctx := chi.NewRouteContext()
		for k, v := range urlParams {
			rctx.URLParams.Add(k, v)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)
	}
	if userID != nil {
		ctx = auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: *userID})
		req = req.WithContext(ctx)
	}
	return req
}

func TestSetCardVote_Unauthorized(t *testing.T) {
	config := &Config{CardVoteReader: &mockCardVoteReader{}}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "1", "cardId": "2"}, nil)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Authentication required")
}

func TestSetCardVote_InvalidDebateID(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) { return database.Users{ID: id}, nil },
		},
	}
	req := requestWithChiParams("PUT", "/debates/foo/cards/2/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "foo", "cardId": "2"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid debate ID")
}

func TestSetCardVote_InvalidCardID(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) { return database.Users{ID: id}, nil },
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/bar/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "1", "cardId": "bar"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid card ID")
}

func TestSetCardVote_InvalidBody(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) { return database.Users{ID: id}, nil },
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", nil, map[string]string{"debateId": "1", "cardId": "2"}, &userID)
	req.Body = http.NoBody
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid request body")
}

func TestSetCardVote_InvalidVoteType(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) {
				return database.Users{ID: id}, nil
			},
			// GetDebateCard not called; handler returns 400 on vote_type before fetching card
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", SetCardVoteRequest{VoteType: "invalid"}, map[string]string{"debateId": "1", "cardId": "2"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "upvote")
}

func TestSetCardVote_UserNotFound(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) {
				return database.Users{}, sql.ErrNoRows
			},
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "1", "cardId": "2"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Account not found")
}

func TestSetCardVote_GetUserError(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) {
				return database.Users{}, assert.AnError
			},
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "1", "cardId": "2"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Failed to verify user")
}

func TestSetCardVote_CardNotFound(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) {
				return database.Users{ID: id}, nil
			},
			getDebateCardFunc: func(ctx context.Context, id int32) (database.DebateCards, error) {
				return database.DebateCards{}, sql.ErrNoRows
			},
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "1", "cardId": "2"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Card not found")
}

func TestSetCardVote_CardWrongDebate(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) {
				return database.Users{ID: id}, nil
			},
			getDebateCardFunc: func(ctx context.Context, id int32) (database.DebateCards, error) {
				// card belongs to debate 99, request is for debate 1
				return database.DebateCards{ID: 2, DebateID: sql.NullInt32{Int32: 99, Valid: true}}, nil
			},
		},
	}
	req := requestWithChiParams("PUT", "/debates/1/cards/2/vote", SetCardVoteRequest{VoteType: "upvote"}, map[string]string{"debateId": "1", "cardId": "2"}, &userID)
	rec := httptest.NewRecorder()

	config.setCardVote(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "does not belong to this debate")
}

func TestCardVoteCounts_JSON(t *testing.T) {
	c := CardVoteCounts{CardID: 1, YesCount: 10, NoCount: 5, TotalYes: 30, TotalNo: 15}
	data, err := json.Marshal(c)
	require.NoError(t, err)
	var out CardVoteCounts
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, c.CardID, out.CardID)
	assert.Equal(t, c.YesCount, out.YesCount)
	assert.Equal(t, c.NoCount, out.NoCount)
	assert.Equal(t, c.TotalYes, out.TotalYes)
	assert.Equal(t, c.TotalNo, out.TotalNo)
}

// Exact sqlc strings (must match internal/database/debates.sql.go).
const (
	testSQLDeleteDebateSwipeVotes = `-- name: DeleteDebateSwipeVotes :exec
DELETE FROM votes
WHERE user_id = $1
  AND vote_type IN ('upvote', 'downvote')
  AND emoji IS NULL
  AND debate_card_id IN (
    SELECT id FROM debate_cards
    WHERE debate_id = $2
      AND stance IN ('agree', 'disagree')
  )
`
	testSQLCreateVote = `-- name: CreateVote :one
INSERT INTO votes (debate_card_id, user_id, vote_type, emoji)
VALUES ($1, $2, $3, $4)
ON CONFLICT (debate_card_id, user_id, vote_type, emoji) 
DO UPDATE SET emoji = $4, created_at = CURRENT_TIMESTAMP
RETURNING id, debate_card_id, user_id, vote_type, emoji, created_at
`
	testSQLGetDebateCard = `-- name: GetDebateCard :one
SELECT id, debate_id, stance, title, description, ai_generated, created_at, updated_at FROM debate_cards WHERE id = $1
`
	testSQLGetDebateCardsOrdered = `-- name: GetDebateCards :many
SELECT id, debate_id, stance, title, description, ai_generated, created_at, updated_at FROM debate_cards WHERE debate_id = $1 ORDER BY stance
`
	testSQLCardVoteGetVoteCounts = `-- name: GetVoteCounts :many
SELECT 
    debate_card_id,
    vote_type,
    emoji,
    COUNT(*) as count
FROM votes 
WHERE debate_card_id = ANY($1::int[])
GROUP BY debate_card_id, vote_type, emoji
`
)

func TestSetCardVote_ReplacesPriorVoteOnOtherBinaryCard(t *testing.T) {
	// Voting NO on the disagree card must clear any prior YES on the agree card
	// (one swipe vote per user per debate).
	const (
		debateID     int32 = 42
		agreeCardID  int32 = 100
		disagreeCard int32 = 101
		userID       int32 = 7
	)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT pg_advisory_xact_lock($1, $2)`).
		WithArgs(debateID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(testSQLDeleteDebateSwipeVotes).
		WithArgs(sql.NullInt32{Int32: userID, Valid: true}, sql.NullInt32{Int32: debateID, Valid: true}).
		WillReturnResult(sqlmock.NewResult(0, 1)) // prior agree-card vote removed
	ts := time.Now()
	mock.ExpectQuery(testSQLCreateVote).
		WithArgs(
			sql.NullInt32{Int32: disagreeCard, Valid: true},
			sql.NullInt32{Int32: userID, Valid: true},
			"downvote",
			sql.NullString{},
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "debate_card_id", "user_id", "vote_type", "emoji", "created_at"}).
			AddRow(int64(9), int64(disagreeCard), int64(userID), "downvote", nil, ts))
	mock.ExpectCommit()

	// Analytics + counts after commit (keep path successful / deterministic).
	mock.ExpectQuery(testSQLGetDebateCard).
		WithArgs(disagreeCard).
		WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "stance", "title", "description", "ai_generated", "created_at", "updated_at"}).
			AddRow(disagreeCard, debateID, "disagree", "No", "d", false, ts, ts))
	mock.ExpectQuery(testSQLGetDebateCardsOrdered).
		WithArgs(sql.NullInt32{Int32: debateID, Valid: true}).
		WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "stance", "title", "description", "ai_generated", "created_at", "updated_at"}).
			AddRow(agreeCardID, debateID, "agree", "Yes", "y", false, ts, ts).
			AddRow(disagreeCard, debateID, "disagree", "No", "n", false, ts, ts))
	mock.ExpectQuery(testSQLCardVoteGetVoteCounts).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"debate_card_id", "vote_type", "emoji", "count"}).
			AddRow(int64(disagreeCard), "downvote", nil, int64(1)))
	mock.ExpectQuery(`-- name: GetCommentCount :one
SELECT COUNT(*) FROM comments WHERE debate_id = $1
`).
		WithArgs(sql.NullInt32{Int32: debateID, Valid: true}).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`-- name: UpdateDebateAnalytics :one
UPDATE debate_analytics 
SET total_votes = $2, total_comments = $3, engagement_score = $4, updated_at = CURRENT_TIMESTAMP
WHERE debate_id = $1
RETURNING id, debate_id, total_votes, total_comments, engagement_score, created_at, updated_at
`).
		WithArgs(
			sql.NullInt32{Int32: debateID, Valid: true},
			sql.NullInt32{Int32: 1, Valid: true},
			sql.NullInt32{Int32: 0, Valid: true},
			sql.NullString{String: "1.00", Valid: true},
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "total_votes", "total_comments", "engagement_score", "created_at", "updated_at"}).
			AddRow(int64(1), debateID, int64(1), int64(0), "1.00", ts, ts))
	// buildCardVoteCounts re-reads cards + counts:
	mock.ExpectQuery(testSQLGetDebateCardsOrdered).
		WithArgs(sql.NullInt32{Int32: debateID, Valid: true}).
		WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "stance", "title", "description", "ai_generated", "created_at", "updated_at"}).
			AddRow(agreeCardID, debateID, "agree", "Yes", "y", false, ts, ts).
			AddRow(disagreeCard, debateID, "disagree", "No", "n", false, ts, ts))
	mock.ExpectQuery(testSQLCardVoteGetVoteCounts).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"debate_card_id", "vote_type", "emoji", "count"}).
			AddRow(int64(disagreeCard), "downvote", nil, int64(1)))

	config := &Config{
		DB:     database.New(db),
		DBConn: db,
		CardVoteReader: &mockCardVoteReader{
			getUserFunc: func(ctx context.Context, id int32) (database.Users, error) {
				return database.Users{ID: id}, nil
			},
			getDebateCardFunc: func(ctx context.Context, id int32) (database.DebateCards, error) {
				return database.DebateCards{
					ID:       disagreeCard,
					DebateID: sql.NullInt32{Int32: debateID, Valid: true},
					Stance:   "disagree",
				}, nil
			},
		},
	}

	uid := userID
	req := requestWithChiParams(
		"PUT",
		"/debates/42/cards/101/vote",
		SetCardVoteRequest{VoteType: "downvote"},
		map[string]string{"debateId": "42", "cardId": "101"},
		&uid,
	)
	rec := httptest.NewRecorder()
	config.setCardVote(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteDebateSwipeVotes_SQLIsDebateScoped(t *testing.T) {
	// Guard against regressing to card-only delete (which allows dual agree+disagree votes).
	require.Contains(t, testSQLDeleteDebateSwipeVotes, "debate_id = $2")
	require.Contains(t, testSQLDeleteDebateSwipeVotes, "stance IN ('agree', 'disagree')")
	require.NotContains(t, testSQLDeleteDebateSwipeVotes, "debate_card_id = $1")
}

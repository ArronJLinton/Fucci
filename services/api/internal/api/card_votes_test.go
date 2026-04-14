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
	"github.com/ArronJLinton/fucci-api/internal/database"
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

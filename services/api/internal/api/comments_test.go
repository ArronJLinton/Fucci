package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCommentReader implements CommentReader for tests.
type mockCommentReader struct {
	getDebateFunc   func(ctx context.Context, id int32) (database.Debates, error)
	getCommentsFunc func(ctx context.Context, debateID sql.NullInt32) ([]database.GetCommentsRow, error)
	getCommentFunc  func(ctx context.Context, id int32) (database.GetCommentRow, error)
}

func (m *mockCommentReader) GetDebate(ctx context.Context, id int32) (database.Debates, error) {
	if m.getDebateFunc != nil {
		return m.getDebateFunc(ctx, id)
	}
	return database.Debates{}, sql.ErrNoRows
}

func (m *mockCommentReader) GetComments(ctx context.Context, debateID sql.NullInt32) ([]database.GetCommentsRow, error) {
	if m.getCommentsFunc != nil {
		return m.getCommentsFunc(ctx, debateID)
	}
	return nil, nil
}

func (m *mockCommentReader) GetComment(ctx context.Context, id int32) (database.GetCommentRow, error) {
	if m.getCommentFunc != nil {
		return m.getCommentFunc(ctx, id)
	}
	return database.GetCommentRow{}, sql.ErrNoRows
}

// commentRequestWithChiParams builds a request with chi URL params and optional JWT user id in context.
func commentRequestWithChiParams(method, path string, body interface{}, urlParams map[string]string, userID *int32) *http.Request {
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

// ---- ListDebateComments ----

func TestListDebateComments_InvalidDebateID(t *testing.T) {
	config := &Config{CommentReader: &mockCommentReader{}}
	req := commentRequestWithChiParams("GET", "/debates/foo/comments", nil, map[string]string{"debateId": "foo"}, nil)
	rec := httptest.NewRecorder()

	config.ListDebateComments(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid debate ID")
}

func TestListDebateComments_DebateNotFound(t *testing.T) {
	config := &Config{
		CommentReader: &mockCommentReader{
			getDebateFunc: func(ctx context.Context, id int32) (database.Debates, error) {
				return database.Debates{}, sql.ErrNoRows
			},
		},
	}
	req := commentRequestWithChiParams("GET", "/debates/1/comments", nil, map[string]string{"debateId": "1"}, nil)
	rec := httptest.NewRecorder()

	config.ListDebateComments(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Debate not found")
}

func TestListDebateComments_SuccessEmpty(t *testing.T) {
	config := &Config{
		CommentReader: &mockCommentReader{
			getDebateFunc: func(ctx context.Context, id int32) (database.Debates, error) {
				return database.Debates{ID: id}, nil
			},
			getCommentsFunc: func(ctx context.Context, debateID sql.NullInt32) ([]database.GetCommentsRow, error) {
				return []database.GetCommentsRow{}, nil
			},
		},
	}
	req := commentRequestWithChiParams("GET", "/debates/1/comments", nil, map[string]string{"debateId": "1"}, nil)
	rec := httptest.NewRecorder()

	config.ListDebateComments(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var out []DebateComment
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Empty(t, out)
}

// ---- CreateDebateComment ----

func TestCreateDebateComment_Unauthorized(t *testing.T) {
	config := &Config{CommentReader: &mockCommentReader{}}
	req := commentRequestWithChiParams("POST", "/debates/1/comments", CreateDebateCommentRequest{Content: "hello"}, map[string]string{"debateId": "1"}, nil)
	rec := httptest.NewRecorder()

	config.CreateDebateComment(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Authentication required")
}

func TestCreateDebateComment_InvalidDebateID(t *testing.T) {
	userID := int32(1)
	config := &Config{CommentReader: &mockCommentReader{}}
	req := commentRequestWithChiParams("POST", "/debates/foo/comments", CreateDebateCommentRequest{Content: "hello"}, map[string]string{"debateId": "foo"}, &userID)
	rec := httptest.NewRecorder()

	config.CreateDebateComment(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid debate ID")
}

func TestCreateDebateComment_ContentEmpty(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CommentReader: &mockCommentReader{
			getDebateFunc: func(ctx context.Context, id int32) (database.Debates, error) { return database.Debates{ID: id}, nil },
		},
	}
	req := commentRequestWithChiParams("POST", "/debates/1/comments", CreateDebateCommentRequest{Content: "   "}, map[string]string{"debateId": "1"}, &userID)
	rec := httptest.NewRecorder()

	config.CreateDebateComment(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "content is required")
}

func TestCreateDebateComment_ContentTooLong(t *testing.T) {
	userID := int32(1)
	longContent := strings.Repeat("x", 501)
	config := &Config{
		CommentReader: &mockCommentReader{
			getDebateFunc: func(ctx context.Context, id int32) (database.Debates, error) { return database.Debates{ID: id}, nil },
		},
	}
	req := commentRequestWithChiParams("POST", "/debates/1/comments", CreateDebateCommentRequest{Content: longContent}, map[string]string{"debateId": "1"}, &userID)
	rec := httptest.NewRecorder()

	config.CreateDebateComment(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "500")
}

func TestCreateDebateComment_DebateNotFound(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CommentReader: &mockCommentReader{
			getDebateFunc: func(ctx context.Context, id int32) (database.Debates, error) {
				return database.Debates{}, sql.ErrNoRows
			},
		},
	}
	req := commentRequestWithChiParams("POST", "/debates/1/comments", CreateDebateCommentRequest{Content: "hello"}, map[string]string{"debateId": "1"}, &userID)
	rec := httptest.NewRecorder()

	config.CreateDebateComment(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Debate not found")
}

// ---- SetCommentVote ----

func TestSetCommentVote_Unauthorized(t *testing.T) {
	config := &Config{}
	vt := "upvote"
	req := commentRequestWithChiParams("PUT", "/comments/1/vote", map[string]interface{}{"vote_type": vt}, map[string]string{"commentId": "1"}, nil)
	rec := httptest.NewRecorder()

	config.SetCommentVote(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Authentication required")
}

func TestSetCommentVote_InvalidCommentID(t *testing.T) {
	userID := int32(1)
	config := &Config{}
	vt := "upvote"
	req := commentRequestWithChiParams("PUT", "/comments/foo/vote", map[string]interface{}{"vote_type": vt}, map[string]string{"commentId": "foo"}, &userID)
	rec := httptest.NewRecorder()

	config.SetCommentVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid comment ID")
}

func TestSetCommentVote_InvalidBody(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CommentReader: &mockCommentReader{
			getCommentFunc: func(ctx context.Context, id int32) (database.GetCommentRow, error) {
				return database.GetCommentRow{ID: id}, nil
			},
		},
	}
	req := commentRequestWithChiParams("PUT", "/comments/1/vote", nil, map[string]string{"commentId": "1"}, &userID)
	req.Body = http.NoBody
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	config.SetCommentVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid request body")
}

func TestSetCommentVote_InvalidVoteType(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CommentReader: &mockCommentReader{
			getCommentFunc: func(ctx context.Context, id int32) (database.GetCommentRow, error) {
				return database.GetCommentRow{ID: id}, nil
			},
		},
	}
	req := commentRequestWithChiParams("PUT", "/comments/1/vote", map[string]interface{}{"vote_type": "invalid"}, map[string]string{"commentId": "1"}, &userID)
	rec := httptest.NewRecorder()

	config.SetCommentVote(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "upvote or downvote")
}

func TestSetCommentVote_CommentNotFound(t *testing.T) {
	userID := int32(1)
	config := &Config{
		CommentReader: &mockCommentReader{
			getCommentFunc: func(ctx context.Context, id int32) (database.GetCommentRow, error) {
				return database.GetCommentRow{}, sql.ErrNoRows
			},
		},
	}
	req := commentRequestWithChiParams("PUT", "/comments/999/vote", map[string]interface{}{"vote_type": "upvote"}, map[string]string{"commentId": "999"}, &userID)
	rec := httptest.NewRecorder()

	config.SetCommentVote(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Comment not found")
}

// ---- AddCommentReaction ----

func TestAddCommentReaction_Unauthorized(t *testing.T) {
	config := &Config{}
	req := commentRequestWithChiParams("POST", "/comments/1/reactions", AddCommentReactionRequest{Emoji: "👍"}, map[string]string{"commentId": "1"}, nil)
	rec := httptest.NewRecorder()

	config.AddCommentReaction(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Authentication required")
}

func TestAddCommentReaction_InvalidCommentID(t *testing.T) {
	userID := int32(1)
	config := &Config{}
	req := commentRequestWithChiParams("POST", "/comments/foo/reactions", AddCommentReactionRequest{Emoji: "👍"}, map[string]string{"commentId": "foo"}, &userID)
	rec := httptest.NewRecorder()

	config.AddCommentReaction(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid comment ID")
}

func TestAddCommentReaction_EmojiRequired(t *testing.T) {
	userID := int32(1)
	config := &Config{}
	req := commentRequestWithChiParams("POST", "/comments/1/reactions", AddCommentReactionRequest{Emoji: "   "}, map[string]string{"commentId": "1"}, &userID)
	rec := httptest.NewRecorder()

	config.AddCommentReaction(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "emoji is required")
}

// ---- RemoveCommentReaction ----

func TestRemoveCommentReaction_Unauthorized(t *testing.T) {
	config := &Config{}
	req := commentRequestWithChiParams("DELETE", "/comments/1/reactions?emoji=👍", nil, map[string]string{"commentId": "1"}, nil)
	req.URL.RawQuery = "emoji=👍"
	rec := httptest.NewRecorder()

	config.RemoveCommentReaction(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Authentication required")
}

func TestRemoveCommentReaction_InvalidCommentID(t *testing.T) {
	userID := int32(1)
	config := &Config{}
	req := commentRequestWithChiParams("DELETE", "/comments/foo/reactions", nil, map[string]string{"commentId": "foo"}, &userID)
	req.URL.RawQuery = "emoji=👍"
	rec := httptest.NewRecorder()

	config.RemoveCommentReaction(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "Invalid comment ID")
}

func TestRemoveCommentReaction_EmojiQueryRequired(t *testing.T) {
	userID := int32(1)
	config := &Config{}
	req := commentRequestWithChiParams("DELETE", "/comments/1/reactions", nil, map[string]string{"commentId": "1"}, &userID)
	rec := httptest.NewRecorder()

	config.RemoveCommentReaction(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "emoji query parameter")
}

// ---- Struct / JSON ----

func TestDebateComment_JSON(t *testing.T) {
	c := DebateComment{
		ID: 1, DebateID: 1, UserID: 1, UserDisplayName: "Alice", Content: "Hi",
		NetScore: 2, Reactions: []ReactionCount{{Emoji: "👍", Count: 1}},
	}
	data, err := json.Marshal(c)
	require.NoError(t, err)
	var out DebateComment
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, c.ID, out.ID)
	assert.Equal(t, c.UserDisplayName, out.UserDisplayName)
	assert.Equal(t, c.NetScore, out.NetScore)
	assert.Len(t, out.Reactions, 1)
	assert.Equal(t, "👍", out.Reactions[0].Emoji)
}

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
)

const (
	commentMaxLength   = 500
	commentRateLimitN  = 10
	commentRateWindow  = time.Minute
)

type commentRateEntry struct {
	count       int
	windowStart time.Time
}

// commentRateLimiter is in-memory per-user rate limit for comment creation (e.g. N per minute).
type commentRateLimiter struct {
	mu     sync.Mutex
	byUser map[int32]commentRateEntry
}

func (r *commentRateLimiter) allow(userID int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byUser == nil {
		r.byUser = make(map[int32]commentRateEntry)
	}
	now := time.Now()
	entry := r.byUser[userID]
	if now.Sub(entry.windowStart) >= commentRateWindow {
		entry.count = 0
		entry.windowStart = now
	}
	entry.count++
	r.byUser[userID] = entry
	return entry.count <= commentRateLimitN
}

var defaultCommentRateLimiter commentRateLimiter

// DebateComment is the public API shape for a comment (006 spec). Seeded flag is not exposed.
type DebateComment struct {
	ID                int32              `json:"id"`
	DebateID          int32              `json:"debate_id"`
	ParentCommentID   *int32             `json:"parent_comment_id,omitempty"`
	UserID            int32              `json:"user_id"`
	UserDisplayName   string             `json:"user_display_name"`
	UserAvatarURL     *string            `json:"user_avatar_url,omitempty"`
	Content           string             `json:"content"`
	CreatedAt         time.Time          `json:"created_at"`
	NetScore          int32              `json:"net_score"`
	CurrentUserVote   *string            `json:"current_user_vote,omitempty"` // "upvote" | "downvote" | null when unauthenticated
	Reactions         []ReactionCount    `json:"reactions"`
	Subcomments       []DebateComment    `json:"subcomments,omitempty"`
}

// ReactionCount is emoji + count for a comment.
type ReactionCount struct {
	Emoji string `json:"emoji"`
	Count int32  `json:"count"`
}

// ListDebateComments handles GET /api/debates/{debate_id}/comments.
// Returns top-level comments with one level of subcomments, net_score and reaction counts per comment.
// Public (no auth). Seeded flag and stance are not exposed.
func (c *Config) ListDebateComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	debateIDStr := chi.URLParam(r, "debateId")
	debateID, err := strconv.ParseInt(debateIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid debate ID")
		return
	}

	// Optional: verify debate exists
	_, err = c.DB.GetDebate(ctx, int32(debateID))
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Debate not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get debate: %v", err))
		return
	}

	rows, err := c.DB.GetComments(ctx, sql.NullInt32{Int32: int32(debateID), Valid: true})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get comments: %v", err))
		return
	}

	// Split into top-level and subcomments (do not expose seeded)
	var topLevel []database.GetCommentsRow
	subByParent := make(map[int32][]database.GetCommentsRow)
	for _, row := range rows {
		if !row.ParentCommentID.Valid {
			topLevel = append(topLevel, row)
			continue
		}
		parentID := row.ParentCommentID.Int32
		subByParent[parentID] = append(subByParent[parentID], row)
	}

	// Build response: for each top-level comment, add net_score, reactions, subcomments
	out := make([]DebateComment, 0, len(topLevel))
	for _, row := range topLevel {
		comment, err := c.buildDebateComment(ctx, row, nil)
		if err != nil {
			continue
		}
		// Attach subcomments (one level)
		if subs, ok := subByParent[row.ID]; ok {
			comment.Subcomments = make([]DebateComment, 0, len(subs))
			for _, subRow := range subs {
				subComment, err := c.buildDebateComment(ctx, subRow, nil)
				if err != nil {
					continue
				}
				comment.Subcomments = append(comment.Subcomments, subComment)
			}
		}
		out = append(out, comment)
	}

	respondWithJSON(w, http.StatusOK, out)
}

// CreateDebateCommentRequest is the body for POST /api/debates/{debate_id}/comments.
type CreateDebateCommentRequest struct {
	Content          string `json:"content"`
	ParentCommentID  *int32 `json:"parent_comment_id,omitempty"`
}

// CreateDebateComment handles POST /api/debates/{debate_id}/comments.
// Requires auth. Content ≤ 500 chars; parent_comment_id must be top-level. Rate-limited.
func (c *Config) CreateDebateComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	debateIDStr := chi.URLParam(r, "debateId")
	debateID, err := strconv.ParseInt(debateIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid debate ID")
		return
	}

	var req CreateDebateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		respondWithError(w, http.StatusBadRequest, "content is required")
		return
	}
	if len(content) > commentMaxLength {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("content must be at most %d characters", commentMaxLength))
		return
	}

	if !defaultCommentRateLimiter.allow(userID) {
		respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded; try again later")
		return
	}

	_, err = c.DB.GetDebate(ctx, int32(debateID))
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Debate not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to get debate")
		return
	}

	var parentCommentID sql.NullInt32
	if req.ParentCommentID != nil && *req.ParentCommentID > 0 {
		parent, err := c.DB.GetComment(ctx, *req.ParentCommentID)
		if err != nil {
			if err == sql.ErrNoRows {
				respondWithError(w, http.StatusNotFound, "Parent comment not found")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Failed to get parent comment")
			return
		}
		if parent.DebateID.Int32 != int32(debateID) {
			respondWithError(w, http.StatusBadRequest, "Parent comment does not belong to this debate")
			return
		}
		if parent.ParentCommentID.Valid {
			respondWithError(w, http.StatusBadRequest, "Replies only allowed to top-level comments")
			return
		}
		parentCommentID = sql.NullInt32{Int32: *req.ParentCommentID, Valid: true}
	}

	comment, err := c.DB.CreateComment(ctx, database.CreateCommentParams{
		DebateID:        sql.NullInt32{Int32: int32(debateID), Valid: true},
		ParentCommentID: parentCommentID,
		UserID:          sql.NullInt32{Int32: userID, Valid: true},
		Content:         content,
		Seeded:          false,
	})
	if err != nil {
		log.Printf("[comments] CreateComment error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create comment")
		return
	}

	row, err := c.DB.GetComment(ctx, comment.ID)
	if err != nil {
		respondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"id": comment.ID, "debate_id": debateID, "content": content,
		})
		return
	}

	out, err := c.buildDebateCommentFromCommentRow(ctx, row, &userID)
	if err != nil {
		respondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"id": comment.ID, "debate_id": debateID, "content": content,
		})
		return
	}
	respondWithJSON(w, http.StatusCreated, out)
}

// buildDebateCommentFromCommentRow builds DebateComment from GetCommentRow (e.g. after create).
func (c *Config) buildDebateCommentFromCommentRow(ctx context.Context, row database.GetCommentRow, currentUserID *int32) (DebateComment, error) {
	r := database.GetCommentsRow{
		ID: row.ID, DebateID: row.DebateID, ParentCommentID: row.ParentCommentID, UserID: row.UserID,
		Content: row.Content, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Seeded: row.Seeded,
		Firstname: row.Firstname, Lastname: row.Lastname,
		DisplayName: row.DisplayName, AvatarUrl: row.AvatarUrl,
	}
	return c.buildDebateComment(ctx, r, currentUserID)
}

// SetCommentVoteRequest is the body for PUT /api/comments/{comment_id}/vote.
type SetCommentVoteRequest struct {
	VoteType *string `json:"vote_type"` // "upvote", "downvote", or null to clear
}

// SetCommentVote handles PUT /api/comments/{comment_id}/vote. Requires auth.
func (c *Config) SetCommentVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	commentIDStr := chi.URLParam(r, "commentId")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid comment ID")
		return
	}

	var req SetCommentVoteRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.VoteType == nil || *req.VoteType == "" || *req.VoteType == "null" {
		_ = c.DB.DeleteCommentVote(ctx, database.DeleteCommentVoteParams{CommentID: int32(commentID), UserID: userID})
		netScore, _ := c.DB.GetCommentVoteNetScore(ctx, int32(commentID))
		respondWithJSON(w, http.StatusOK, map[string]interface{}{"net_score": netScore, "vote_type": nil})
		return
	}

	if *req.VoteType != "upvote" && *req.VoteType != "downvote" {
		respondWithError(w, http.StatusBadRequest, "vote_type must be upvote or downvote")
		return
	}

	_, err = c.DB.UpsertCommentVote(ctx, database.UpsertCommentVoteParams{
		CommentID: int32(commentID),
		UserID:    userID,
		VoteType:  *req.VoteType,
	})
	if err != nil {
		log.Printf("[comments] UpsertCommentVote error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to set vote")
		return
	}

	netScore, _ := c.DB.GetCommentVoteNetScore(ctx, int32(commentID))
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"net_score": netScore, "vote_type": *req.VoteType})
}

// AddCommentReactionRequest is the body for POST /api/comments/{comment_id}/reactions.
type AddCommentReactionRequest struct {
	Emoji string `json:"emoji"`
}

// AddCommentReaction handles POST /api/comments/{comment_id}/reactions. Toggle: if exists remove, else add. Requires auth.
func (c *Config) AddCommentReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	commentIDStr := chi.URLParam(r, "commentId")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid comment ID")
		return
	}

	var req AddCommentReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Emoji) == "" {
		respondWithError(w, http.StatusBadRequest, "emoji is required")
		return
	}
	emoji := strings.TrimSpace(req.Emoji)

	_, err = c.DB.GetUserCommentReaction(ctx, database.GetUserCommentReactionParams{
		CommentID: int32(commentID), UserID: userID, Emoji: emoji,
	})
	if err == nil {
		_ = c.DB.RemoveCommentReaction(ctx, database.RemoveCommentReactionParams{
			CommentID: int32(commentID), UserID: userID, Emoji: emoji,
		})
	} else {
		_, err = c.DB.AddCommentReaction(ctx, database.AddCommentReactionParams{
			CommentID: int32(commentID), UserID: userID, Emoji: emoji,
		})
		if err != nil {
			log.Printf("[comments] AddCommentReaction error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to add reaction")
			return
		}
	}

	rows, _ := c.DB.GetCommentReactionsByCommentID(ctx, int32(commentID))
	reactions := make([]ReactionCount, 0, len(rows))
	for _, rr := range rows {
		reactions = append(reactions, ReactionCount{Emoji: rr.Emoji, Count: rr.Count})
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"reactions": reactions})
}

// RemoveCommentReaction handles DELETE /api/comments/{comment_id}/reactions?emoji=. Requires auth.
func (c *Config) RemoveCommentReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	commentIDStr := chi.URLParam(r, "commentId")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid comment ID")
		return
	}

	emoji := strings.TrimSpace(r.URL.Query().Get("emoji"))
	if emoji == "" {
		respondWithError(w, http.StatusBadRequest, "emoji query parameter is required")
		return
	}

	_ = c.DB.RemoveCommentReaction(ctx, database.RemoveCommentReactionParams{
		CommentID: int32(commentID), UserID: userID, Emoji: emoji,
	})

	rows, _ := c.DB.GetCommentReactionsByCommentID(ctx, int32(commentID))
	reactions := make([]ReactionCount, 0, len(rows))
	for _, rr := range rows {
		reactions = append(reactions, ReactionCount{Emoji: rr.Emoji, Count: rr.Count})
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"reactions": reactions})
}

// buildDebateComment converts a GetCommentsRow to DebateComment (no seeded, no stance).
// currentUserVote is nil for unauthenticated list.
// Uses users.display_name when set, else firstname+lastname; users.avatar_url for UserAvatarURL.
func (c *Config) buildDebateComment(ctx context.Context, row database.GetCommentsRow, currentUserID *int32) (DebateComment, error) {
	displayName := ""
	if row.DisplayName.Valid && strings.TrimSpace(row.DisplayName.String) != "" {
		displayName = strings.TrimSpace(row.DisplayName.String)
	} else {
		displayName = strings.TrimSpace(row.Firstname + " " + row.Lastname)
	}
	if displayName == "" {
		displayName = "User"
	}

	var avatarURL *string
	if row.AvatarUrl.Valid && strings.TrimSpace(row.AvatarUrl.String) != "" {
		s := strings.TrimSpace(row.AvatarUrl.String)
		avatarURL = &s
	}

	netScore, _ := c.DB.GetCommentVoteNetScore(ctx, row.ID)
	reactionRows, _ := c.DB.GetCommentReactionsByCommentID(ctx, row.ID)
	reactions := make([]ReactionCount, 0, len(reactionRows))
	for _, rr := range reactionRows {
		reactions = append(reactions, ReactionCount{Emoji: rr.Emoji, Count: rr.Count})
	}

	var createdAt time.Time
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}

	comment := DebateComment{
		ID:                row.ID,
		DebateID:          row.DebateID.Int32,
		UserID:            row.UserID.Int32,
		UserDisplayName:   displayName,
		UserAvatarURL:     avatarURL,
		Content:           row.Content,
		CreatedAt:         createdAt,
		NetScore:          netScore,
		Reactions:         reactions,
	}
	if row.ParentCommentID.Valid {
		comment.ParentCommentID = &row.ParentCommentID.Int32
	}
	// current_user_vote: when we have auth, look up vote for currentUserID; for now leave nil
	if currentUserID != nil {
		vote, err := c.DB.GetCommentVoteByUser(ctx, database.GetCommentVoteByUserParams{CommentID: row.ID, UserID: *currentUserID})
		if err == nil {
			comment.CurrentUserVote = &vote.VoteType
		}
	}
	return comment, nil
}

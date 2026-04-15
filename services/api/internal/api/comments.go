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

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
)

const (
	commentMaxLength  = 500
	commentRateLimitN = 10
	commentRateWindow = time.Minute
)

func sqlNullStringVal(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

type commentRateEntry struct {
	count       int
	windowStart time.Time
}

// commentRateLimiter is in-memory per-user rate limit for comment creation (fallback when Redis unavailable).
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

// checkCommentRateLimit returns true if the user is within the comment-creation rate limit (N per minute).
// Uses Redis-backed Cache (Incr/Expire/TTL) when available for consistent enforcement across API instances;
// falls back to in-memory per-process limiter when cache is nil or unavailable.
func checkCommentRateLimit(ctx context.Context, c *Config, userID int32) bool {
	if c.Cache != nil {
		key := fmt.Sprintf("comment_rate:%d", userID)
		n, err := c.Cache.Incr(ctx, key)
		if err == nil {
			if n == 1 {
				if err := c.Cache.Expire(ctx, key, commentRateWindow); err != nil {
					log.Printf("[comments] rate limit Redis Expire failed: %v", err)
				}
			} else {
				ttl, err := c.Cache.TTL(ctx, key)
				if err != nil {
					log.Printf("[comments] rate limit Redis TTL failed: %v", err)
				} else if ttl < 0 {
					if err := c.Cache.Expire(ctx, key, commentRateWindow); err != nil {
						log.Printf("[comments] rate limit Redis fallback Expire failed: %v", err)
					}
				}
			}
			return n <= int64(commentRateLimitN)
		}
		log.Printf("[comments] rate limit Redis Incr failed: %v; using in-memory fallback", err)
	}
	return defaultCommentRateLimiter.allow(userID)
}

// DebateComment is the public API shape for a comment (006 spec). Seeded flag is not exposed.
type DebateComment struct {
	ID              int32           `json:"id"`
	DebateID        int32           `json:"debate_id"`
	ParentCommentID *int32          `json:"parent_comment_id,omitempty"`
	UserID          int32           `json:"user_id"`
	UserDisplayName string          `json:"user_display_name"`
	UserAvatarURL   *string         `json:"user_avatar_url,omitempty"`
	Content         string          `json:"content"`
	CreatedAt       time.Time       `json:"created_at"`
	NetScore        int32           `json:"net_score"`
	CurrentUserVote *string         `json:"current_user_vote,omitempty"` // "upvote" | "downvote" | null when unauthenticated
	Reactions       []ReactionCount `json:"reactions"`
	Subcomments     []DebateComment `json:"subcomments,omitempty"`
}

// ReactionCount is emoji + count for a comment.
type ReactionCount struct {
	Emoji string `json:"emoji"`
	Count int32  `json:"count"`
}

func sqlNullString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
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
	debateReader := CommentReader(c.DB)
	if c.CommentReader != nil {
		debateReader = c.CommentReader
	}
	_, err = debateReader.GetDebate(ctx, int32(debateID))
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Debate not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get debate: %v", err))
		return
	}

	rows, err := debateReader.GetComments(ctx, sql.NullInt32{Int32: int32(debateID), Valid: true})
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

	// Collect all comment IDs for batch fetch (avoids N+1)
	commentIDs := make([]int32, 0, len(rows))
	for _, row := range topLevel {
		commentIDs = append(commentIDs, row.ID)
		if subs := subByParent[row.ID]; len(subs) > 0 {
			for _, subRow := range subs {
				commentIDs = append(commentIDs, subRow.ID)
			}
		}
	}

	var netScoreByComment map[int32]int32
	var reactionsByComment map[int32][]ReactionCount
	if len(commentIDs) > 0 {
		netScoreByComment = make(map[int32]int32)
		if batchScores, err := c.DB.GetCommentVoteNetScoresBatch(ctx, commentIDs); err == nil {
			for _, r := range batchScores {
				netScoreByComment[r.CommentID] = r.NetScore
			}
		}
		reactionsByComment = make(map[int32][]ReactionCount)
		if batchReactions, err := c.DB.GetCommentReactionsByCommentIDsBatch(ctx, commentIDs); err == nil {
			for _, r := range batchReactions {
				reactionsByComment[r.CommentID] = append(reactionsByComment[r.CommentID], ReactionCount{Emoji: r.Emoji, Count: r.Count})
			}
		}
	}

	// Build response: for each top-level comment, add net_score, reactions, subcomments (from preloaded maps)
	out := make([]DebateComment, 0, len(topLevel))
	for _, row := range topLevel {
		comment, err := c.buildDebateCommentFromRowWithPreloaded(row, netScoreByComment[row.ID], reactionsByComment[row.ID], nil)
		if err != nil {
			continue
		}
		if subs, ok := subByParent[row.ID]; ok {
			comment.Subcomments = make([]DebateComment, 0, len(subs))
			for _, subRow := range subs {
				subComment, err := c.buildDebateCommentFromRowWithPreloaded(subRow, netScoreByComment[subRow.ID], reactionsByComment[subRow.ID], nil)
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
	Content         string `json:"content"`
	ParentCommentID *int32 `json:"parent_comment_id,omitempty"`
}

// CreateDebateComment handles POST /api/debates/{debate_id}/comments.
// Requires auth. Content ≤ 500 chars; parent_comment_id must be top-level. Rate-limited.
func (c *Config) CreateDebateComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(r.Context())
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

	if !checkCommentRateLimit(ctx, c, userID) {
		respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded; try again later")
		return
	}

	commentReader := CommentReader(c.DB)
	if c.CommentReader != nil {
		commentReader = c.CommentReader
	}
	_, err = commentReader.GetDebate(ctx, int32(debateID))
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
		parent, err := commentReader.GetComment(ctx, *req.ParentCommentID)
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
// API-level toggle: sending the same vote again clears it (delete and return vote_type: null).
// Explicit clear: send vote_type null/empty to remove vote. Returns 404 if comment does not exist.
func (c *Config) SetCommentVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(r.Context())
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Verify comment exists so we return 404 instead of 200 with net_score: 0 for invalid comment_id
	commentGetter := CommentReader(c.DB)
	if c.CommentReader != nil {
		commentGetter = c.CommentReader
	}
	_, err = commentGetter.GetComment(ctx, int32(commentID))
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Comment not found")
			return
		}
		log.Printf("[comments] GetComment error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get comment")
		return
	}

	// Explicit clear: client sent null/empty/"null"
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

	// Toggle: same vote again clears it (API-level semantics; client can send vote_type and server clears if already set)
	existing, err := c.DB.GetCommentVoteByUser(ctx, database.GetCommentVoteByUserParams{CommentID: int32(commentID), UserID: userID})
	if err == nil && existing.VoteType == *req.VoteType {
		_ = c.DB.DeleteCommentVote(ctx, database.DeleteCommentVoteParams{CommentID: int32(commentID), UserID: userID})
		netScore, _ := c.DB.GetCommentVoteNetScore(ctx, int32(commentID))
		respondWithJSON(w, http.StatusOK, map[string]interface{}{"net_score": netScore, "vote_type": nil})
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

	userID, ok := auth.UserIDFromContext(r.Context())
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

	userID, ok := auth.UserIDFromContext(r.Context())
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

// buildDebateCommentFromRowWithPreloaded builds a DebateComment from a GetCommentsRow using
// preloaded net score, reactions, and optional current-user vote (no DB calls). Used by ListDebateComments
// after batch-fetching scores and reactions.
func (c *Config) buildDebateCommentFromRowWithPreloaded(
	row database.GetCommentsRow,
	netScore int32,
	reactions []ReactionCount,
	currentUserVote *string,
) (DebateComment, error) {
	displayName := ""
	if row.DisplayName.Valid && strings.TrimSpace(row.DisplayName.String) != "" {
		displayName = strings.TrimSpace(row.DisplayName.String)
	} else {
		displayName = strings.TrimSpace(
			sqlNullString(row.Firstname) + " " + sqlNullString(row.Lastname),
		)
	}
	if displayName == "" {
		if row.Seeded {
			displayName = "Fucci"
		} else {
			displayName = "User"
		}
	}
	var avatarURL *string
	if row.AvatarUrl.Valid && strings.TrimSpace(row.AvatarUrl.String) != "" {
		s := strings.TrimSpace(row.AvatarUrl.String)
		avatarURL = &s
	}
	var createdAt time.Time
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	if reactions == nil {
		reactions = []ReactionCount{}
	}
	comment := DebateComment{
		ID:              row.ID,
		DebateID:        row.DebateID.Int32,
		UserID:          row.UserID.Int32,
		UserDisplayName: displayName,
		UserAvatarURL:   avatarURL,
		Content:         row.Content,
		CreatedAt:       createdAt,
		NetScore:        netScore,
		Reactions:       reactions,
		CurrentUserVote: currentUserVote,
	}
	if row.ParentCommentID.Valid {
		comment.ParentCommentID = &row.ParentCommentID.Int32
	}
	return comment, nil
}

// buildDebateComment converts a GetCommentsRow to DebateComment (no seeded, no stance).
// Fetches net_score, reactions, and optionally current_user_vote from DB (use buildDebateCommentFromRowWithPreloaded when listing with batch data).
// User display name and avatar come from the row: GetComments/GetComment select u.display_name and u.avatar_url; the shared helper prefers display_name over firstname+lastname and sets user_avatar_url.
func (c *Config) buildDebateComment(ctx context.Context, row database.GetCommentsRow, currentUserID *int32) (DebateComment, error) {
	netScore, _ := c.DB.GetCommentVoteNetScore(ctx, row.ID)
	reactionRows, _ := c.DB.GetCommentReactionsByCommentID(ctx, row.ID)
	reactions := make([]ReactionCount, 0, len(reactionRows))
	for _, rr := range reactionRows {
		reactions = append(reactions, ReactionCount{Emoji: rr.Emoji, Count: rr.Count})
	}
	var currentUserVote *string
	if currentUserID != nil {
		vote, err := c.DB.GetCommentVoteByUser(ctx, database.GetCommentVoteByUserParams{CommentID: row.ID, UserID: *currentUserID})
		if err == nil {
			currentUserVote = &vote.VoteType
		}
	}
	return c.buildDebateCommentFromRowWithPreloaded(row, netScore, reactions, currentUserVote)
}

package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
)

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

// buildDebateComment converts a GetCommentsRow to DebateComment (no seeded, no stance).
// currentUserVote is nil for unauthenticated list.
func (c *Config) buildDebateComment(ctx context.Context, row database.GetCommentsRow, currentUserID *int32) (DebateComment, error) {
	displayName := strings.TrimSpace(row.Firstname + " " + row.Lastname)
	if displayName == "" {
		displayName = "User"
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
		ID:              row.ID,
		DebateID:        row.DebateID.Int32,
		UserID:          row.UserID.Int32,
		UserDisplayName: displayName,
		Content:         row.Content,
		CreatedAt:       createdAt,
		NetScore:        netScore,
		Reactions:       reactions,
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

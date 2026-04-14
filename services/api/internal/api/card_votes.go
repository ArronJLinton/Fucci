package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
	"github.com/lib/pq"
)

// CardVoteCounts is the response for PUT card vote (006 spec).
type CardVoteCounts struct {
	CardID   int32 `json:"card_id"`
	YesCount int   `json:"yes_count"`
	NoCount  int   `json:"no_count"`
	TotalYes int   `json:"total_yes,omitempty"`
	TotalNo  int   `json:"total_no,omitempty"`
}

// SetCardVoteRequest is the body for PUT /api/debates/{debate_id}/cards/{card_id}/vote.
type SetCardVoteRequest struct {
	VoteType string `json:"vote_type"` // "upvote" (yes) or "downvote" (no)
}

// setCardVote handles PUT /api/debates/{debate_id}/cards/{card_id}/vote.
// Requires auth. One vote per user per card (replaces existing). No rate limit.
func (c *Config) setCardVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Ensure user exists (e.g. DB was reset or token from another env) to avoid votes_user_id_fkey violation
	userReader := CardVoteReader(c.DB)
	if c.CardVoteReader != nil {
		userReader = c.CardVoteReader
	}
	if _, err := userReader.GetUser(ctx, userID); err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Account not found. Please log in again.")
			return
		}
		log.Printf("[card_votes] GetUser error: %v (user_id=%d)", err, userID)
		respondWithError(w, http.StatusInternalServerError, "Failed to verify user")
		return
	}

	debateIDStr := chi.URLParam(r, "debateId")
	cardIDStr := chi.URLParam(r, "cardId")
	debateID, err := strconv.ParseInt(debateIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid debate ID")
		return
	}
	cardID, err := strconv.ParseInt(cardIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid card ID")
		return
	}

	var req SetCardVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.VoteType != "upvote" && req.VoteType != "downvote" {
		respondWithError(w, http.StatusBadRequest, "vote_type must be 'upvote' or 'downvote'")
		return
	}

	cardReader := CardVoteReader(c.DB)
	if c.CardVoteReader != nil {
		cardReader = c.CardVoteReader
	}
	card, err := cardReader.GetDebateCard(ctx, int32(cardID))
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Card not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to get card")
		return
	}
	if !card.DebateID.Valid || card.DebateID.Int32 != int32(debateID) {
		respondWithError(w, http.StatusNotFound, "Card does not belong to this debate")
		return
	}

	// One vote per user per card: delete existing swipe vote then insert (atomic).
	// Partial unique index idx_votes_swipe_one_per_user_card enforces at most one row per (card, user) when emoji IS NULL.
	tx, err := c.DBConn.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[card_votes] BeginTx error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to save vote")
		return
	}
	defer func() { _ = tx.Rollback() }()

	q := c.DB.WithTx(tx)
	_ = q.DeleteCardSwipeVotes(ctx, database.DeleteCardSwipeVotesParams{
		DebateCardID: sql.NullInt32{Int32: int32(cardID), Valid: true},
		UserID:       sql.NullInt32{Int32: userID, Valid: true},
	})

	_, err = q.CreateVote(ctx, database.CreateVoteParams{
		DebateCardID: sql.NullInt32{Int32: int32(cardID), Valid: true},
		UserID:       sql.NullInt32{Int32: userID, Valid: true},
		VoteType:     req.VoteType,
		Emoji:        sql.NullString{},
	})
	if err != nil {
		log.Printf("[card_votes] CreateVote error: %v (debate_id=%d card_id=%d user_id=%d)", err, debateID, cardID, userID)
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23503": // foreign_key_violation
				respondWithError(w, http.StatusBadRequest, "User or card not found")
				return
			case "23505": // unique_violation — partial index or race
				respondWithError(w, http.StatusConflict, "Vote already recorded")
				return
			}
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to save vote")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[card_votes] Commit error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to save vote")
		return
	}

	c.updateDebateAnalytics(ctx, int32(cardID))

	counts, err := c.buildCardVoteCounts(ctx, int32(debateID), int32(cardID))
	if err != nil {
		respondWithJSON(w, http.StatusOK, CardVoteCounts{
			CardID:   int32(cardID),
			YesCount: 0,
			NoCount:  0,
		})
		return
	}
	respondWithJSON(w, http.StatusOK, counts)
}

// buildCardVoteCounts returns per-card and debate totals for swipe (upvote=yes, downvote=no, emoji null only).
func (c *Config) buildCardVoteCounts(ctx context.Context, debateID, cardID int32) (*CardVoteCounts, error) {
	cards, err := c.DB.GetDebateCards(ctx, sql.NullInt32{Int32: debateID, Valid: true})
	if err != nil {
		return nil, err
	}
	cardIDs := make([]int32, len(cards))
	for i := range cards {
		cardIDs[i] = cards[i].ID
	}
	rows, err := c.DB.GetVoteCounts(ctx, cardIDs)
	if err != nil {
		return nil, err
	}

	// Only count swipe votes: vote_type upvote/downvote with null emoji
	perCard := make(map[int32]struct{ Yes, No int })
	var totalYes, totalNo int
	for _, r := range rows {
		if !r.DebateCardID.Valid || (r.VoteType != "upvote" && r.VoteType != "downvote") || r.Emoji.Valid {
			continue
		}
		cid := r.DebateCardID.Int32
		n := int(r.Count)
		if r.VoteType == "upvote" {
			pc := perCard[cid]
			pc.Yes += n
			perCard[cid] = pc
			totalYes += n
		} else {
			pc := perCard[cid]
			pc.No += n
			perCard[cid] = pc
			totalNo += n
		}
	}

	pc := perCard[cardID]
	out := &CardVoteCounts{
		CardID:   cardID,
		YesCount: pc.Yes,
		NoCount:  pc.No,
		TotalYes: totalYes,
		TotalNo:  totalNo,
	}
	return out, nil
}

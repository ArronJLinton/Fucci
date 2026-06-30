package push

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// DebatePushStore loads the top unvoted debate for push.
type DebatePushStore interface {
	GetTopUnvotedDebateForPush(ctx context.Context, userID int32) (database.GetTopUnvotedDebateForPushRow, error)
}

// DebateDailySelector picks the top unvoted debate by engagement for 6pm local.
type DebateDailySelector struct {
	Store DebatePushStore
}

func (s *DebateDailySelector) Select(ctx context.Context, user UserCandidate) (SelectResult, error) {
	if s.Store == nil {
		return SelectResult{Skip: "debate_store_unconfigured"}, nil
	}
	debate, err := s.Store.GetTopUnvotedDebateForPush(ctx, user.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SelectResult{Skip: "all_debates_voted"}, nil
		}
		return SelectResult{}, fmt.Errorf("top unvoted debate: %w", err)
	}
	if strings.TrimSpace(debate.Headline) == "" {
		return SelectResult{Skip: "debate_missing_headline"}, nil
	}

	return SelectResult{
		Request: &SendRequest{
			Title: debate.Headline,
			Body:  "Tap to vote on today's top debate.",
			Data:  debatePushData(debate.ID, debate.MatchID),
		},
	}, nil
}

func debatePushData(debateID int32, matchID string) map[string]interface{} {
	params := map[string]interface{}{"debateId": debateID}
	if matchID != "" {
		params["matchId"] = matchID
	}
	return map[string]interface{}{
		"type":   "debate",
		"route":  "SingleDebate",
		"params": params,
	}
}

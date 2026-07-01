package push

import (
	"fmt"
	"strings"
	"time"
)

const (
	CampaignMatchHighlightsPrefix = "match:"
	CampaignMatchHighlightsSuffix = ":highlights"
	CampaignMatchDebatesSuffix    = ":debates_live"

	MaxMatchPushesPerDay = 2
	MarqueeMaxFIFARank   = 50

	DefaultDelayAfterFT = time.Hour
	// DefaultDebatesStaggerAfterHighlights is the gap between highlights and debates-live when both send.
	DefaultDebatesStaggerAfterHighlights = 30 * time.Minute
	MatchFTScanInterval = 10 * time.Minute
)

// CampaignMatchHighlights returns the ledger key for a fixture highlights push.
func CampaignMatchHighlights(fixtureID int) string {
	return fmt.Sprintf("match:%d:highlights", fixtureID)
}

// CampaignMatchDebatesLive returns the ledger key for a post-FT debates push.
func CampaignMatchDebatesLive(fixtureID int) string {
	return fmt.Sprintf("match:%d:debates_live", fixtureID)
}

var finishedMatchStatuses = map[string]struct{}{
	"FT": {}, "AET": {}, "PEN": {}, "FT_PEN": {}, "AET_PEN": {},
}

// IsFinishedMatchStatus reports whether API-Football short status is full-time.
func IsFinishedMatchStatus(status string) bool {
	_, ok := finishedMatchStatuses[strings.ToUpper(strings.TrimSpace(status))]
	return ok
}

// MatchFixture is a normalized finished fixture candidate for post-FT push.
type MatchFixture struct {
	ID           int
	HomeTeamID   int
	AwayTeamID   int
	HomeTeamName string
	AwayTeamName string
	HomeGoals    int
	AwayGoals    int
	Kickoff      time.Time
	EstimatedEnd time.Time
}

// EstimateMatchEnd approximates full-time from periods.second or kickoff + 105m.
func EstimateMatchEnd(kickoff time.Time, periodSecondUnix int) time.Time {
	if periodSecondUnix > 0 {
		return time.Unix(int64(periodSecondUnix), 0).Add(15 * time.Minute)
	}
	return kickoff.Add(105 * time.Minute)
}

func matchPushData(matchID string, shortVideoID string) map[string]interface{} {
	params := map[string]interface{}{"matchId": matchID}
	if shortVideoID != "" {
		params["shortVideoId"] = shortVideoID
	}
	return map[string]interface{}{
		"type":   "match",
		"route":  "MatchDetails",
		"params": params,
	}
}

func formatScoreline(home, away string, hg, ag int) string {
	return fmt.Sprintf("%s %d – %d %s", home, hg, ag, away)
}

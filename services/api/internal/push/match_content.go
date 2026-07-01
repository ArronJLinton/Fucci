package push

import (
	"strconv"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

// ShortCandidate is a media-outlet Short tied to a fixture.
type ShortCandidate struct {
	Title    string
	VideoID  string
	Outlet   string
	PostedAt time.Time
}

// FindMatchHighlightShort picks the newest Short mentioning either team after kickoff.
func FindMatchHighlightShort(
	outlets []youtube.MediaOutletShorts,
	fixture MatchFixture,
) *ShortCandidate {
	if len(outlets) == 0 {
		return nil
	}
	homeTokens := teamMatchTokens(fixture.HomeTeamName)
	awayTokens := teamMatchTokens(fixture.AwayTeamName)

	var best *ShortCandidate
	for _, outlet := range outlets {
		for _, s := range outlet.Shorts {
			posted := s.PublishedAt
			if !posted.IsZero() && posted.Before(fixture.Kickoff) {
				continue
			}
			if !shortMentionsTeams(s.Title, homeTokens, awayTokens) {
				continue
			}
			candidate := &ShortCandidate{
				Title:    s.Title,
				VideoID:  s.VideoID,
				Outlet:   outlet.DisplayName,
				PostedAt: posted,
			}
			if best == nil || posted.After(best.PostedAt) {
				best = candidate
			}
		}
	}
	return best
}

func teamMatchTokens(name string) []string {
	key := youtube.LookupKeyForTeamName(name)
	parts := strings.Fields(key)
	if len(parts) == 0 {
		return nil
	}
	// Prefer last token ("states" for united states) plus full key.
	tokens := []string{key}
	if len(parts) > 1 {
		tokens = append(tokens, parts[len(parts)-1])
	}
	return tokens
}

func shortMentionsTeams(title string, homeTokens, awayTokens []string) bool {
	haystack := " " + strings.ToLower(title) + " "
	for _, t := range append(homeTokens, awayTokens...) {
		if t == "" {
			continue
		}
		if strings.Contains(haystack, " "+t+" ") || strings.Contains(haystack, t) {
			return true
		}
	}
	return false
}

// BuildMatchHighlightsPushRequest builds a Shorts highlights notification.
func BuildMatchHighlightsPushRequest(fixture MatchFixture, short *ShortCandidate) SendRequest {
	matchID := itoa(fixture.ID)
	scoreline := formatScoreline(fixture.HomeTeamName, fixture.AwayTeamName, fixture.HomeGoals, fixture.AwayGoals)
	return SendRequest{
		CampaignKey: CampaignMatchHighlights(fixture.ID),
		Category:    CategoryMatches,
		Title:       short.Title,
		Body:        scoreline + " — watch the highlights",
		Data:        matchPushData(matchID, short.VideoID),
	}
}

// BuildMatchDebatesLivePushRequest builds a post-FT debates notification.
func BuildMatchDebatesLivePushRequest(fixture MatchFixture) SendRequest {
	matchID := itoa(fixture.ID)
	scoreline := formatScoreline(fixture.HomeTeamName, fixture.AwayTeamName, fixture.HomeGoals, fixture.AwayGoals)
	return SendRequest{
		CampaignKey: CampaignMatchDebatesLive(fixture.ID),
		Category:    CategoryMatches,
		Title:       "Match finished — debates live",
		Body:        scoreline + " — join the debate",
		Data:        matchPushData(matchID, ""),
	}
}

// BuildMatchPushRequest builds highlights or debates-live (legacy helper).
func BuildMatchPushRequest(fixture MatchFixture, short *ShortCandidate) SendRequest {
	if short != nil {
		return BuildMatchHighlightsPushRequest(fixture, short)
	}
	return BuildMatchDebatesLivePushRequest(fixture)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

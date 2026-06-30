package api

import (
	"context"
	"time"

	pushpkg "github.com/ArronJLinton/fucci-api/internal/push"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

// LeagueMatchFixtures returns finished fixtures for a league/date from the shared matches cache.
func (c *Config) LeagueMatchFixtures(ctx context.Context, date time.Time, leagueID int) ([]pushpkg.MatchFixture, error) {
	lid := leagueID
	resp, err := c.FetchMatchesCached(ctx, date, &lid, nil)
	if err != nil {
		return nil, err
	}
	out := make([]pushpkg.MatchFixture, 0, len(resp.Response))
	for _, row := range resp.Response {
		status := row.Fixture.Status.Short
		if !pushpkg.IsFinishedMatchStatus(status) {
			continue
		}
		kickoff := row.Fixture.Date
		if kickoff.IsZero() && row.Fixture.Timestamp > 0 {
			kickoff = time.Unix(int64(row.Fixture.Timestamp), 0)
		}
		out = append(out, pushpkg.MatchFixture{
			ID:           row.Fixture.ID,
			HomeTeamID:   row.Teams.Home.ID,
			AwayTeamID:   row.Teams.Away.ID,
			HomeTeamName: row.Teams.Home.Name,
			AwayTeamName: row.Teams.Away.Name,
			HomeGoals:    row.Goals.Home,
			AwayGoals:    row.Goals.Away,
			Kickoff:      kickoff,
			EstimatedEnd: pushpkg.EstimateMatchEnd(kickoff, row.Fixture.Periods.Second),
		})
	}
	return out, nil
}

// MediaOutletsShortsForPush loads FOX/ESPN/Telemundo Shorts for match highlight matching.
func (c *Config) MediaOutletsShortsForPush(ctx context.Context) []youtube.MediaOutletShorts {
	svc := c.youtubeShortsService()
	if svc == nil || c.DB == nil {
		return nil
	}
	return svc.GetMediaOutletsShorts(ctx, c.DB)
}

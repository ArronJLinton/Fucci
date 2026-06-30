package push

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

// MatchPushStore is the DB surface for post-FT match pushes.
type MatchPushStore interface {
	DispatchStore
	ListMatchPushCandidates(ctx context.Context) ([]database.ListMatchPushCandidatesRow, error)
	CountMatchPushSendsForUserOnDate(ctx context.Context, arg database.CountMatchPushSendsForUserOnDateParams) (int64, error)
	ListNationalTeamRankings(ctx context.Context) ([]database.NationalTeamRankings, error)
}

// MatchFetcher loads fixtures for a league/date (typically API-Football via Redis cache).
type MatchFetcher interface {
	FetchLeagueMatches(ctx context.Context, date time.Time, leagueID int) ([]MatchFixture, error)
}

// MediaShortsProvider loads FOX/ESPN/Telemundo-style outlet Shorts.
type MediaShortsProvider interface {
	MediaOutletsShorts(ctx context.Context) []youtube.MediaOutletShorts
}

// MatchFTJob scans finished marquee fixtures ~1h after FT and sends match pushes.
type MatchFTJob struct {
	Matches   MatchFetcher
	Shorts    MediaShortsProvider
	Service   *Service
	Store     MatchPushStore
	Lock      scanLocker
	LeagueIDs []int
	DelayAfterFT time.Duration
	Now       func() time.Time
}

func (j *MatchFTJob) Name() string { return "push-match-ft" }

func (j *MatchFTJob) Run(ctx context.Context) error {
	if j.Service == nil || j.Store == nil || j.Matches == nil {
		return fmt.Errorf("match FT job not configured")
	}
	now := j.now()
	delay := j.DelayAfterFT
	if delay <= 0 {
		delay = DefaultDelayAfterFT
	}

	marquee, err := j.loadMarquee(ctx)
	if err != nil {
		return err
	}

	shorts := []youtube.MediaOutletShorts{}
	if j.Shorts != nil {
		shorts = j.Shorts.MediaOutletsShorts(ctx)
	}

	candidates, err := j.Store.ListMatchPushCandidates(ctx)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	dates := []time.Time{now, now.Add(-24 * time.Hour)}
	sentFixtures := 0
	for _, leagueID := range j.leagueIDs() {
		for _, day := range dates {
			fixtures, err := j.Matches.FetchLeagueMatches(ctx, day, leagueID)
			if err != nil {
				log.Printf("[push-match-ft] league=%d date=%s fetch error: %v", leagueID, day.Format("2006-01-02"), err)
				continue
			}
			for _, fx := range fixtures {
				if !marquee.IsMarquee(fx.HomeTeamID, fx.AwayTeamID) {
					continue
				}
				if now.Before(fx.EstimatedEnd.Add(delay)) {
					continue
				}
				if !j.acquireFixtureLock(ctx, fx.ID) {
					continue
				}

				short := FindMatchHighlightShort(shorts, fx)
				req := BuildMatchPushRequest(fx, short)
				n := j.sendToCandidates(ctx, candidates, req)
				if n > 0 {
					sentFixtures++
					log.Printf("[push-match-ft] fixture=%d campaign=%s sent=%d", fx.ID, req.CampaignKey, n)
				}
			}
		}
	}
	if sentFixtures > 0 {
		log.Printf("[push-match-ft] tick sent for %d fixture(s)", sentFixtures)
	}
	return nil
}

func (j *MatchFTJob) leagueIDs() []int {
	if len(j.LeagueIDs) > 0 {
		return j.LeagueIDs
	}
	return []int{1} // FIFA World Cup
}

func (j *MatchFTJob) now() time.Time {
	if j.Now != nil {
		return j.Now()
	}
	return time.Now()
}

func (j *MatchFTJob) loadMarquee(ctx context.Context) (*Marquee, error) {
	rows, err := j.Store.ListNationalTeamRankings(ctx)
	if err != nil {
		return nil, err
	}
	rankRows := make([]RankRow, len(rows))
	for i, r := range rows {
		rankRows[i] = RankRow{TeamID: r.TeamID, FIFARank: r.FifaRank}
	}
	return NewMarquee(rankRows, MarqueeMaxFIFARank), nil
}

func (j *MatchFTJob) acquireFixtureLock(ctx context.Context, fixtureID int) bool {
	if j.Lock == nil {
		return true
	}
	slot := CurrentScanSlot(j.now())
	key := fmt.Sprintf("push:match:fixture:%d:%s", fixtureID, slot)
	ok, err := j.Lock.SetNX(ctx, key, ScanLockTTL)
	if err != nil {
		log.Printf("[push-match-ft] SetNX(%s) failed: %v — proceeding", key, err)
		return true
	}
	return ok
}

func (j *MatchFTJob) sendToCandidates(
	ctx context.Context,
	candidates []database.ListMatchPushCandidatesRow,
	base SendRequest,
) int {
	sent := 0
	for _, c := range candidates {
		localDate, err := localDateForTimezone(c.Timezone)
		if err != nil {
			localDate = j.now().UTC().Truncate(24 * time.Hour)
		}
		count, err := j.Store.CountMatchPushSendsForUserOnDate(ctx, database.CountMatchPushSendsForUserOnDateParams{
			UserID:    c.UserID,
			LocalDate: localDate,
		})
		if err != nil {
			log.Printf("[push-match-ft] count ledger user=%d: %v", c.UserID, err)
			continue
		}
		if count >= MaxMatchPushesPerDay {
			continue
		}

		req := base
		req.UserID = c.UserID
		req.Timezone = c.Timezone
		if err := j.Service.SendToUser(ctx, req); err != nil {
			log.Printf("[push-match-ft] send user=%d fixture campaign=%s: %v", c.UserID, req.CampaignKey, err)
			continue
		}
		sent++
	}
	return sent
}

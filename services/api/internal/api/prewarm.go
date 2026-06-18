package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/news"
)

// PrewarmJob generates pre-match debates and warms the per-match news cache
// for every fixture scheduled today in the configured leagues. Runs each
// morning so the first user opening the app skips on-demand AI/news latency.
//
// Implements scheduler.Job. Idempotent — every Redis write and downstream
// endpoint (POST /debates/generate-set, GetMatchNewsCached) already short-
// circuits on cached entries, so re-running mid-day costs at most a few
// metadata reads per fixture.
type PrewarmJob struct {
	Config *Config
	// LeagueIDs is the list of API-Football league ids to scan. Typically
	// sourced from config.PREWARM_LEAGUE_IDS; ParsePrewarmLeagueIDs builds it.
	LeagueIDs []int
	// LoopbackBaseURL is the prefix used to call our own HTTP endpoints
	// (debate generate-set). Defaults to http://127.0.0.1:{PORT} when empty.
	LoopbackBaseURL string
	// DebatesPerMatch is the count parameter passed to /debates/generate-set
	// (default 3 — matches the public handler's default).
	DebatesPerMatch int

	httpClient *http.Client
}

// ParsePrewarmLeagueIDs turns the comma-separated env value
// (e.g. "1,39,140") into a []int, skipping blanks and unparsable entries.
// Returns nil for an empty/whitespace input so callers can disable the
// scheduler by clearing the env var.
func ParsePrewarmLeagueIDs(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			log.Printf("[prewarm] ignoring non-numeric PREWARM_LEAGUE_IDS entry %q", p)
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NewPrewarmJob wires up the daily pre-match generator. port is the API's own
// HTTP port (used for the loopback POST to /debates/generate-set).
func NewPrewarmJob(c *Config, leagueIDs []int, port string) *PrewarmJob {
	loopback := ""
	if port != "" {
		loopback = "http://127.0.0.1:" + port
	}
	return &PrewarmJob{
		Config:          c,
		LeagueIDs:       leagueIDs,
		LoopbackBaseURL: loopback,
		DebatesPerMatch: 3,
		httpClient:      &http.Client{Timeout: 90 * time.Second},
	}
}

// Name implements scheduler.Job.
func (j *PrewarmJob) Name() string { return "prematch-debate-prewarm" }

// Run implements scheduler.Job: list today's fixtures, then for each one
// warm the news cache and POST to /debates/generate-set. A Redis SetNX lock
// keyed on the UTC date ensures only one API machine actually does the work
// per day (others log and exit).
func (j *PrewarmJob) Run(ctx context.Context) error {
	if j.Config == nil {
		return errors.New("prewarm: nil Config")
	}
	if len(j.LeagueIDs) == 0 {
		log.Printf("[prewarm] no leagues configured (PREWARM_LEAGUE_IDS empty); nothing to do")
		return nil
	}
	if j.Config.AIPromptGenerator == nil {
		log.Printf("[prewarm] AI prompt generator not configured (missing OPENAI_API_KEY); skipping run")
		return nil
	}
	if j.LoopbackBaseURL == "" {
		return errors.New("prewarm: loopback base URL not configured (PORT was empty)")
	}

	today := time.Now().UTC()
	dateKey := today.Format("2006-01-02")

	// Cross-machine dedup: first machine to SetNX wins; others noop for the day.
	if j.Config.Cache != nil {
		lockKey := "prewarm:debates:date:" + dateKey
		const lockTTL = 25 * time.Hour
		acquired, err := j.Config.Cache.SetNX(ctx, lockKey, lockTTL)
		if err != nil {
			log.Printf("[prewarm] SetNX(%s) failed: %v — proceeding without cross-machine lock", lockKey, err)
		} else if !acquired {
			log.Printf("[prewarm] another machine already ran today's pre-warm (lock %s held); skipping", lockKey)
			return nil
		} else {
			log.Printf("[prewarm] acquired daily lock %s (ttl=%s)", lockKey, lockTTL)
		}
	}

	totalFixtures := 0
	totalDebatesOK := 0
	totalNewsWarmed := 0

	for _, leagueID := range j.LeagueIDs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lid := leagueID
		matches, err := j.Config.FetchMatchesCached(ctx, today, &lid, nil)
		if err != nil {
			log.Printf("[prewarm] league=%d FetchMatchesCached error: %v", leagueID, err)
			continue
		}
		if matches == nil || len(matches.Response) == 0 {
			log.Printf("[prewarm] league=%d date=%s no fixtures", leagueID, dateKey)
			continue
		}
		log.Printf("[prewarm] league=%d date=%s fixtures=%d", leagueID, dateKey, len(matches.Response))

		for _, m := range matches.Response {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			totalFixtures++
			status := m.Fixture.Status.Short
			if !isPreMatchStatus(status) {
				log.Printf("[prewarm] league=%d match=%d status=%s — skip (not pre-match)", leagueID, m.Fixture.ID, status)
				continue
			}
			matchID := strconv.Itoa(m.Fixture.ID)
			homeTeam := m.Teams.Home.Name
			awayTeam := m.Teams.Away.Name

			if err := j.warmNewsCache(ctx, matchID, homeTeam, awayTeam, status); err != nil {
				log.Printf("[prewarm] news warm failed match=%s (%s vs %s): %v", matchID, homeTeam, awayTeam, err)
			} else {
				totalNewsWarmed++
			}

			if err := j.triggerDebateSet(ctx, matchID); err != nil {
				log.Printf("[prewarm] debate generate-set failed match=%s (%s vs %s): %v", matchID, homeTeam, awayTeam, err)
				continue
			}
			totalDebatesOK++
		}
	}

	log.Printf("[prewarm] done date=%s fixtures_seen=%d news_cache_warmed=%d debates_ok=%d",
		dateKey, totalFixtures, totalNewsWarmed, totalDebatesOK)
	return nil
}

// isPreMatchStatus returns true for fixtures that have not yet started and
// therefore make sense to generate pre_match debates for.
// API-Football status codes: NS (not started), TBD (time to be determined),
// PST (postponed), 1H (first half), HT (halftime), 2H, ET, BT, P, SUSP, INT,
// FT, AET, PEN, AWD, WO, CANC, ABD, LIVE.
func isPreMatchStatus(short string) bool {
	switch strings.ToUpper(strings.TrimSpace(short)) {
	case "NS", "TBD":
		return true
	default:
		return false
	}
}

// warmNewsCache pulls match news into the shared per-match cache key so a
// later debate generation OR a user opening the MatchNewsScreen is a hit.
func (j *PrewarmJob) warmNewsCache(ctx context.Context, matchID, homeTeam, awayTeam, status string) error {
	k := j.Config.newsXAPIKey()
	if strings.TrimSpace(k) == "" {
		return errors.New("news API key not configured")
	}
	client := news.NewClient(k)
	if j.Config.NewsBaseURL != "" {
		client = news.NewClientWithBaseURL(k, j.Config.NewsBaseURL)
	}
	const limit = 10
	_, fromCache, err := news.GetMatchNewsCached(
		ctx,
		j.Config.Cache,
		client,
		matchID, homeTeam, awayTeam, status, "",
		nil,
		limit,
		cache.NewsTTL,
	)
	if err != nil {
		return err
	}
	if fromCache {
		log.Printf("[prewarm] news cache HIT match=%s — left alone", matchID)
	} else {
		log.Printf("[prewarm] news cache WARMED match=%s", matchID)
	}
	return nil
}

// triggerDebateSet POSTs to our own /debates/generate-set so we re-use the
// handler's distributed/in-process dedup, rate-limit, and validation logic
// instead of reimplementing it here.
func (j *PrewarmJob) triggerDebateSet(ctx context.Context, matchID string) error {
	body, err := json.Marshal(map[string]any{
		"match_id":    matchID,
		"debate_type": "pre_match",
		"count":       j.DebatesPerMatch,
	})
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	url := j.LoopbackBaseURL + "/v1/api/debates/generate-set"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Source", "prewarm")

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("loopback request: %w", err)
	}
	defer resp.Body.Close()

	// Cap body read so a runaway response doesn't blow up logs.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	log.Printf("[prewarm] generate-set ok match=%s status=%d", matchID, resp.StatusCode)
	return nil
}

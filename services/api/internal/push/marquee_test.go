package push

import (
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

func TestMarquee_IsMarquee(t *testing.T) {
	t.Parallel()
	m := NewMarquee([]RankRow{
		{TeamID: 10, FIFARank: 6},
		{TeamID: 99, FIFARank: 80},
	}, 50)
	if !m.IsMarquee(10, 999) {
		t.Fatal("expected marquee when home team ranked")
	}
	if m.IsMarquee(888, 999) {
		t.Fatal("expected non-marquee when both unranked")
	}
}

func TestMarquee_EmptyRanksAllowsAll(t *testing.T) {
	t.Parallel()
	m := NewMarquee(nil, 50)
	if !m.IsMarquee(1, 2) {
		t.Fatal("empty ranks should allow all fixtures")
	}
}

func TestBuildMatchPushRequest(t *testing.T) {
	t.Parallel()
	fx := MatchFixture{
		ID: 123, HomeTeamName: "USA", AwayTeamName: "Mexico", HomeGoals: 2, AwayGoals: 1,
	}
	short := &ShortCandidate{Title: "USA stun Mexico", VideoID: "abc"}
	req := BuildMatchHighlightsPushRequest(fx, short)
	if req.CampaignKey != CampaignMatchHighlights(123) {
		t.Fatalf("got key %q", req.CampaignKey)
	}
	debates := BuildMatchDebatesLivePushRequest(fx)
	if debates.CampaignKey != CampaignMatchDebatesLive(123) {
		t.Fatalf("got debates key %q", debates.CampaignKey)
	}
}

func TestFindMatchHighlightShort(t *testing.T) {
	t.Parallel()
	kickoff := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	fx := MatchFixture{HomeTeamName: "England", AwayTeamName: "France", Kickoff: kickoff}
	outlets := []youtube.MediaOutletShorts{{
		DisplayName: "FOX",
		Shorts: []youtube.Short{{
			Title: "England beat France in thriller", VideoID: "v1",
			PublishedAt: kickoff.Add(2 * time.Hour),
		}},
	}}
	short := FindMatchHighlightShort(outlets, fx)
	if short == nil || short.VideoID != "v1" {
		t.Fatalf("expected highlight short, got %+v", short)
	}
}

func TestEstimateMatchEnd(t *testing.T) {
	t.Parallel()
	kickoff := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	end := EstimateMatchEnd(kickoff, 0)
	want := kickoff.Add(105 * time.Minute)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
}

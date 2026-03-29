package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/sqlc-dev/pqtype"
)

func TestParsePositiveInt32Query(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		raw    string
		def    int32
		max    int32
		expect int32
	}{
		{"empty uses default", "", 30, 50, 30},
		{"within range", "40", 30, 50, 40},
		{"clamped to max", "999", 30, 50, 50},
		{"invalid uses default", "x", 20, 50, 20},
		{"zero uses default", "0", 20, 50, 20},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			u := "/debates/public-feed"
			if tc.raw != "" {
				u += "?limit=" + tc.raw
			}
			r := httptest.NewRequest(http.MethodGet, u, nil)
			got := parsePositiveInt32Query(r, "limit", tc.def, tc.max)
			if got != tc.expect {
				t.Fatalf("got %d want %d", got, tc.expect)
			}
		})
	}
}

func TestPublicDebateFeedResponseJSONShape(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	pub := PublicDebateFeedResponse{
		Debates: []DebateSummary{
			{
				ID:         1,
				MatchID:    "1321727",
				Headline:   "Test",
				DebateType: "pre_match",
				CreatedAt:  ts,
				Analytics: &DebateAnalyticsSummary{
					TotalVotes:      10,
					TotalComments:   2,
					EngagementScore: 14,
				},
			},
		},
	}
	b, err := json.Marshal(pub)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["debates"]; !ok {
		t.Fatal("expected top-level debates key")
	}
}

func TestDebateFeedResponseJSONShape(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	feed := DebateFeedResponse{
		NewDebates: []DebateSummary{
			{ID: 1, MatchID: "1", Headline: "N", DebateType: "pre_match", CreatedAt: ts},
		},
		VotedDebates: []DebateSummary{
			{
				ID: 2, MatchID: "2", Headline: "V", DebateType: "post_match", CreatedAt: ts,
				LastVotedAt: &ts,
			},
		},
	}
	b, err := json.Marshal(feed)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["new_debates"]; !ok {
		t.Fatal("expected new_debates key")
	}
	if _, ok := raw["voted_debates"]; !ok {
		t.Fatal("expected voted_debates key")
	}
}

func TestDebateSummaryFromPublicFeedRowMapsAnalytics(t *testing.T) {
	row := database.ListDebatesPublicFeedRow{
		ID:              5,
		MatchID:         "99",
		DebateType:      "pre_match",
		Headline:        "H",
		Description:     sql.NullString{String: "D", Valid: true},
		AiGenerated:     sql.NullBool{Bool: true, Valid: true},
		CreatedAt:       sql.NullTime{Time: time.Unix(100, 0).UTC(), Valid: true},
		UpdatedAt:       sql.NullTime{Time: time.Unix(200, 0).UTC(), Valid: true},
		TotalVotes:      sql.NullInt32{Int32: 3, Valid: true},
		TotalComments:   sql.NullInt32{Int32: 1, Valid: true},
		EngagementScore: sql.NullString{String: "5.50", Valid: true},
		BinaryAgreeUpvotes:    int64(12),
		BinaryDisagreeUpvotes: int64(8),
	}
	s, err := debateSummaryFromPublicFeedRow(row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Analytics == nil || s.Analytics.EngagementScore != 5.5 {
		t.Fatalf("analytics: %+v", s.Analytics)
	}
	if s.BinaryConsensus.AgreeUpvotes != 12 || s.BinaryConsensus.DisagreeUpvotes != 8 {
		t.Fatalf("binary_consensus: %+v", s.BinaryConsensus)
	}
	if s.Description != "D" || s.MatchID != "99" {
		t.Fatalf("unexpected summary: %+v", s)
	}
}

func TestLastVotedAtFromSQLCIface(t *testing.T) {
	ts := time.Unix(1000, 0).UTC()
	if p := lastVotedAtFromSQLCIface(nil); p != nil {
		t.Fatal("expected nil")
	}
	if p := lastVotedAtFromSQLCIface(ts); p == nil || !p.Equal(ts) {
		t.Fatalf("got %v", p)
	}
}

func TestDebateSummaryFromVotedFeedRowIncludesLastVotedAt(t *testing.T) {
	ts := time.Unix(500, 0).UTC()
	row := database.ListDebatesFeedVotedForUserRow{
		ID:                    1,
		MatchID:               "1",
		DebateType:            "pre_match",
		Headline:              "H",
		CreatedAt:             sql.NullTime{Time: time.Unix(1, 0).UTC(), Valid: true},
		TotalVotes:            sql.NullInt32{Int32: 1, Valid: true},
		EngagementScore:       sql.NullString{String: "1.00", Valid: true},
		BinaryAgreeUpvotes:    int64(3),
		BinaryDisagreeUpvotes: int64(1),
		LastVotedAt:           ts,
	}
	s, err := debateSummaryFromVotedFeedRow(row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.LastVotedAt == nil || !s.LastVotedAt.Equal(ts) {
		t.Fatalf("last_voted_at: %+v", s.LastVotedAt)
	}
	if s.BinaryConsensus.AgreeUpvotes != 3 || s.BinaryConsensus.DisagreeUpvotes != 1 {
		t.Fatalf("binary_consensus: %+v", s.BinaryConsensus)
	}
}


func TestDebateDataAggregator(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	config := &Config{
		Cache:          cache,
		FootballAPIKey: "mock-api-key",
		RapidAPIKey:    "mock-rapid-api-key",
	}

	aggregator := NewDebateDataAggregator(config)

	t.Run("test aggregator creation", func(t *testing.T) {
		if aggregator == nil {
			t.Error("DebateDataAggregator should not be nil")
		}
		if aggregator.Config != config {
			t.Error("Config should be set correctly")
		}
	})
}

func TestCreateDebateRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := CreateDebateRequest{
			MatchID:     "12345",
			DebateType:  "pre_match",
			Headline:    "Test Debate",
			Description: "Test Description",
			AIGenerated: true,
		}

		if req.MatchID != "12345" {
			t.Errorf("Expected MatchID to be '12345', got %s", req.MatchID)
		}
		if req.DebateType != "pre_match" {
			t.Errorf("Expected DebateType to be 'pre_match', got %s", req.DebateType)
		}
		if req.Headline != "Test Debate" {
			t.Errorf("Expected Headline to be 'Test Debate', got %s", req.Headline)
		}
		if !req.AIGenerated {
			t.Error("Expected AIGenerated to be true")
		}
	})
}

func TestDebateResponse(t *testing.T) {
	t.Run("debate response structure", func(t *testing.T) {
		response := DebateResponse{
			ID:          1,
			MatchID:     "12345",
			DebateType:  "pre_match",
			Headline:    "Test Debate",
			Description: "Test Description",
			AIGenerated: true,
		}

		if response.ID != 1 {
			t.Errorf("Expected ID to be 1, got %d", response.ID)
		}
		if response.MatchID != "12345" {
			t.Errorf("Expected MatchID to be '12345', got %s", response.MatchID)
		}
		if response.DebateType != "pre_match" {
			t.Errorf("Expected DebateType to be 'pre_match', got %s", response.DebateType)
		}
		if !response.AIGenerated {
			t.Error("Expected AIGenerated to be true")
		}
	})
}

func TestVoteCounts(t *testing.T) {
	t.Run("vote counts structure", func(t *testing.T) {
		counts := VoteCounts{
			Upvotes:   10,
			Downvotes: 5,
			Emojis: map[string]int{
				"👍": 15,
				"👎": 3,
			},
		}

		if counts.Upvotes != 10 {
			t.Errorf("Expected Upvotes to be 10, got %d", counts.Upvotes)
		}
		if counts.Downvotes != 5 {
			t.Errorf("Expected Downvotes to be 5, got %d", counts.Downvotes)
		}
		if counts.Emojis["👍"] != 15 {
			t.Errorf("Expected 👍 emoji count to be 15, got %d", counts.Emojis["👍"])
		}
		if counts.Emojis["👎"] != 3 {
			t.Errorf("Expected 👎 emoji count to be 3, got %d", counts.Emojis["👎"])
		}
	})
}

func TestDebateCardResponse(t *testing.T) {
	t.Run("debate card response structure", func(t *testing.T) {
		card := DebateCardResponse{
			ID:          1,
			DebateID:    1,
			Stance:      "agree",
			Title:       "Agree with the decision",
			Description: "This was the right call",
			AIGenerated: true,
			VoteCounts: VoteCounts{
				Upvotes:   25,
				Downvotes: 5,
			},
		}

		if card.Stance != "agree" {
			t.Errorf("Expected Stance to be 'agree', got %s", card.Stance)
		}
		if card.Title != "Agree with the decision" {
			t.Errorf("Expected Title to be 'Agree with the decision', got %s", card.Title)
		}
		if card.VoteCounts.Upvotes != 25 {
			t.Errorf("Expected Upvotes to be 25, got %d", card.VoteCounts.Upvotes)
		}
		if !card.AIGenerated {
			t.Error("Expected AIGenerated to be true")
		}
	})
}

func TestDebateAnalyticsResponse(t *testing.T) {
	t.Run("analytics response structure", func(t *testing.T) {
		analytics := DebateAnalyticsResponse{
			ID:              1,
			DebateID:        1,
			TotalVotes:      100,
			TotalComments:   25,
			EngagementScore: 150.5,
		}

		if analytics.TotalVotes != 100 {
			t.Errorf("Expected TotalVotes to be 100, got %d", analytics.TotalVotes)
		}
		if analytics.TotalComments != 25 {
			t.Errorf("Expected TotalComments to be 25, got %d", analytics.TotalComments)
		}
		if analytics.EngagementScore != 150.5 {
			t.Errorf("Expected EngagementScore to be 150.5, got %f", analytics.EngagementScore)
		}
	})
}

func TestMultipleEmojiVotesOnDebateCard(t *testing.T) {
	// Simulate voting with two different emojis
	emojiVotes := []struct {
		Emoji string
		Count int
	}{
		{"👍", 1},
		{"🔥", 1},
	}

	voteCounts := VoteCounts{
		Emojis: make(map[string]int),
	}

	for _, v := range emojiVotes {
		voteCounts.Emojis[v.Emoji] += v.Count
	}

	if voteCounts.Emojis["👍"] != 1 {
		t.Errorf("Expected 👍 emoji count to be 1, got %d", voteCounts.Emojis["👍"])
	}
	if voteCounts.Emojis["🔥"] != 1 {
		t.Errorf("Expected 🔥 emoji count to be 1, got %d", voteCounts.Emojis["🔥"])
	}

	// Simulate a second vote for 👍
	voteCounts.Emojis["👍"] += 1
	if voteCounts.Emojis["👍"] != 2 {
		t.Errorf("Expected 👍 emoji count to be 2 after second vote, got %d", voteCounts.Emojis["👍"])
	}
}

// mockDebatesFeedStore records calls for handler-level feed tests.
type mockDebatesFeedStore struct {
	publicLimits     []int32
	publicFeedRows   []database.ListDebatesPublicFeedRow
	newParams        []database.ListDebatesFeedNewForUserParams
	votedParams      []database.ListDebatesFeedVotedForUserParams
	publicErr        error
	newErr           error
	votedErr         error
}

func (m *mockDebatesFeedStore) ListDebatesPublicFeed(ctx context.Context, limit int32) ([]database.ListDebatesPublicFeedRow, error) {
	m.publicLimits = append(m.publicLimits, limit)
	if m.publicErr != nil {
		return nil, m.publicErr
	}
	if m.publicFeedRows != nil {
		return m.publicFeedRows, nil
	}
	return nil, nil
}

func (m *mockDebatesFeedStore) ListDebatesFeedNewForUser(ctx context.Context, arg database.ListDebatesFeedNewForUserParams) ([]database.ListDebatesFeedNewForUserRow, error) {
	m.newParams = append(m.newParams, arg)
	if m.newErr != nil {
		return nil, m.newErr
	}
	return nil, nil
}

func (m *mockDebatesFeedStore) ListDebatesFeedVotedForUser(ctx context.Context, arg database.ListDebatesFeedVotedForUserParams) ([]database.ListDebatesFeedVotedForUserRow, error) {
	m.votedParams = append(m.votedParams, arg)
	if m.votedErr != nil {
		return nil, m.votedErr
	}
	return nil, nil
}

func TestGetDebatesPublicFeed_DatabaseNotConfigured(t *testing.T) {
	c := &Config{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/public-feed", nil)
	c.getDebatesPublicFeed(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestGetDebatesPublicFeed_LimitUsesDefault(t *testing.T) {
	mock := &mockDebatesFeedStore{}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/public-feed", nil)
	c.getDebatesPublicFeed(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if len(mock.publicLimits) != 1 || mock.publicLimits[0] != 30 {
		t.Fatalf("public limit = %v, want [30]", mock.publicLimits)
	}
	var out PublicDebateFeedResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Debates == nil {
		t.Fatal("expected debates key")
	}
}

func TestGetDebatesPublicFeed_LimitClampedToMax(t *testing.T) {
	mock := &mockDebatesFeedStore{}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/public-feed?limit=999", nil)
	c.getDebatesPublicFeed(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if len(mock.publicLimits) != 1 || mock.publicLimits[0] != 50 {
		t.Fatalf("public limit = %v, want [50]", mock.publicLimits)
	}
}

func TestGetDebatesPublicFeed_ListError(t *testing.T) {
	mock := &mockDebatesFeedStore{publicErr: errors.New("db boom")}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/public-feed", nil)
	c.getDebatesPublicFeed(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", rec.Code)
	}
}

func TestGetDebatesFeed_Unauthorized(t *testing.T) {
	mock := &mockDebatesFeedStore{}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/feed", nil)
	c.getDebatesFeed(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if len(mock.newParams) != 0 || len(mock.votedParams) != 0 {
		t.Fatal("store should not be called without auth")
	}
}

func TestGetDebatesFeed_OKWithUserID(t *testing.T) {
	mock := &mockDebatesFeedStore{}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/feed", nil)
	ctx := context.WithValue(req.Context(), "user_id", int32(7))
	req = req.WithContext(ctx)
	c.getDebatesFeed(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if len(mock.newParams) != 1 || mock.newParams[0].UserID.Int32 != 7 || mock.newParams[0].Limit != 20 {
		t.Fatalf("new params = %+v", mock.newParams)
	}
	if len(mock.votedParams) != 1 || mock.votedParams[0].UserID.Int32 != 7 || mock.votedParams[0].Limit != 20 {
		t.Fatalf("voted params = %+v", mock.votedParams)
	}
	var out DebateFeedResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.NewDebates == nil || out.VotedDebates == nil {
		t.Fatalf("response: %+v", out)
	}
}

func TestGetDebatesFeed_LimitsClamped(t *testing.T) {
	mock := &mockDebatesFeedStore{}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/feed?new_limit=999&voted_limit=888", nil)
	ctx := context.WithValue(req.Context(), "user_id", int32(1))
	req = req.WithContext(ctx)
	c.getDebatesFeed(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if mock.newParams[0].Limit != 50 || mock.votedParams[0].Limit != 50 {
		t.Fatalf("new=%d voted=%d, want 50 both", mock.newParams[0].Limit, mock.votedParams[0].Limit)
	}
}

func TestGetDebatesFeed_NewListError(t *testing.T) {
	mock := &mockDebatesFeedStore{newErr: errors.New("fail new")}
	c := &Config{DebatesFeedDB: mock}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/feed", nil)
	ctx := context.WithValue(req.Context(), "user_id", int32(1))
	req = req.WithContext(ctx)
	c.getDebatesFeed(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", rec.Code)
	}
}

func TestTeamsFromMatchInfoJSON_ValidNullRawMessage(t *testing.T) {
	mi := MatchInfo{
		HomeTeam: "North FC", AwayTeam: "South United",
		HomeTeamLogo: "https://img/n.png", AwayTeamLogo: "https://img/s.png",
		Status: "FT", HomeScore: 2, AwayScore: 1,
	}
	raw, err := json.Marshal(&mi)
	if err != nil {
		t.Fatal(err)
	}
	nrm := pqtype.NullRawMessage{Valid: true, RawMessage: raw}
	teams := teamsFromMatchInfoJSON(nrm)
	if teams == nil {
		t.Fatal("expected non-nil Teams")
	}
	if teams.Home.Name != "North FC" || teams.Away.Name != "South United" {
		t.Fatalf("names: home=%q away=%q", teams.Home.Name, teams.Away.Name)
	}
	if teams.Home.Logo != "https://img/n.png" || teams.Away.Logo != "https://img/s.png" {
		t.Fatalf("logos: home=%q away=%q", teams.Home.Logo, teams.Away.Logo)
	}
	if teams.Home.Score == nil || teams.Away.Score == nil {
		t.Fatalf("expected scores: home=%v away=%v", teams.Home.Score, teams.Away.Score)
	}
	if *teams.Home.Score != 2 || *teams.Away.Score != 1 {
		t.Fatalf("scores: home=%v away=%v", *teams.Home.Score, *teams.Away.Score)
	}
}

func TestTeamsFromMatchInfoJSON_InProgressIncludesZeroOnBothSides(t *testing.T) {
	mi := MatchInfo{
		HomeTeam: "A", AwayTeam: "B",
		Status: "1H", HomeScore: 1, AwayScore: 0,
	}
	raw, err := json.Marshal(&mi)
	if err != nil {
		t.Fatal(err)
	}
	teams := teamsFromMatchInfoJSON(pqtype.NullRawMessage{Valid: true, RawMessage: raw})
	if teams == nil {
		t.Fatal("expected teams")
	}
	if teams.Home.Score == nil || teams.Away.Score == nil {
		t.Fatalf("want both scores set during 1H: home=%v away=%v", teams.Home.Score, teams.Away.Score)
	}
	if *teams.Home.Score != 1 || *teams.Away.Score != 0 {
		t.Fatalf("got home=%d away=%d", *teams.Home.Score, *teams.Away.Score)
	}
}

func TestTeamsFromMatchInfoJSON_NotStartedOmitsScores(t *testing.T) {
	mi := MatchInfo{
		HomeTeam: "A", AwayTeam: "B",
		Status: "NS", HomeScore: 0, AwayScore: 0,
	}
	raw, err := json.Marshal(&mi)
	if err != nil {
		t.Fatal(err)
	}
	teams := teamsFromMatchInfoJSON(pqtype.NullRawMessage{Valid: true, RawMessage: raw})
	if teams == nil {
		t.Fatal("expected teams")
	}
	if teams.Home.Score != nil || teams.Away.Score != nil {
		t.Fatalf("pre-match should omit scores: home=%v away=%v", teams.Home.Score, teams.Away.Score)
	}
}

func TestTeamsFromMatchInfoJSON_HalftimeZeroZeroShowsBothScores(t *testing.T) {
	mi := MatchInfo{
		HomeTeam: "A", AwayTeam: "B",
		Status: "HT", HomeScore: 0, AwayScore: 0,
	}
	raw, err := json.Marshal(&mi)
	if err != nil {
		t.Fatal(err)
	}
	teams := teamsFromMatchInfoJSON(pqtype.NullRawMessage{Valid: true, RawMessage: raw})
	if teams.Home.Score == nil || teams.Away.Score == nil {
		t.Fatal("HT 0-0 should still expose both scores")
	}
	if *teams.Home.Score != 0 || *teams.Away.Score != 0 {
		t.Fatalf("scores: %d–%d", *teams.Home.Score, *teams.Away.Score)
	}
}

func TestAttachTeamsToSummaries_NoDBQueryWhenTeamsFromJSON(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	c := &Config{DBConn: db}
	home, away := 3, 0
	summaries := []DebateSummary{{
		MatchID: "fixture-100",
		Teams: &DebateTeams{
			Home: DebateTeamSide{Name: "FromJSONHome", Logo: "L1", Score: &home},
			Away: DebateTeamSide{Name: "FromJSONAway", Logo: "L2", Score: &away},
		},
	}}
	c.attachTeamsToSummaries(context.Background(), summaries)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB interaction: %v", err)
	}
	if summaries[0].Teams.Home.Name != "FromJSONHome" {
		t.Fatalf("home name overwritten: %q", summaries[0].Teams.Home.Name)
	}
}

func TestAttachTeamsToSummaries_PreservesJSONTeamsWhenBackfillingNil(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	q := fmt.Sprintf(`
SELECT m.external_match_id, ht.name, ht.logo_url, m.home_score, at.name, at.logo_url, m.away_score
FROM matches m
LEFT JOIN teams ht ON m.home_team_id = ht.id
LEFT JOIN teams at ON m.away_team_id = at.id
WHERE m.external_match_id IN (%s)
`, "$1")
	rows := sqlmock.NewRows([]string{"external_match_id", "hname", "hlogo", "home_score", "aname", "alogo", "away_score"}).
		AddRow("needs-db", "DBHome", "http://db-h", int64(9), "DBAway", "http://db-a", int64(8))
	mock.ExpectQuery(q).WithArgs("needs-db").WillReturnRows(rows)

	c := &Config{DBConn: db}
	summaries := []DebateSummary{
		{
			MatchID: "json-only",
			Teams: &DebateTeams{
				Home: DebateTeamSide{Name: "JSONHome", Logo: "j1"},
				Away: DebateTeamSide{Name: "JSONAway", Logo: "j2"},
			},
		},
		{MatchID: "needs-db", Teams: nil},
	}
	c.attachTeamsToSummaries(context.Background(), summaries)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("DB expectations: %v", err)
	}
	if summaries[0].Teams.Home.Name != "JSONHome" || summaries[0].Teams.Away.Name != "JSONAway" {
		t.Fatalf("JSON debate teams mutated: %+v", summaries[0].Teams)
	}
	if summaries[1].Teams == nil {
		t.Fatal("expected DB backfill for nil Teams")
	}
	if summaries[1].Teams.Home.Name != "DBHome" || summaries[1].Teams.Away.Name != "DBAway" {
		t.Fatalf("DB backfill: %+v", summaries[1].Teams)
	}
	if summaries[1].Teams.Home.Score == nil || *summaries[1].Teams.Home.Score != 9 {
		t.Fatalf("home score: %+v", summaries[1].Teams.Home.Score)
	}
}

func TestGetDebatesPublicFeed_KeepsTeamsFromMatchInfoWithoutMatchesQuery(t *testing.T) {
	mi := MatchInfo{
		HomeTeam: "Stored Home", AwayTeam: "Stored Away",
		HomeTeamLogo: "https://h", AwayTeamLogo: "https://a",
		Status: "FT", HomeScore: 4, AwayScore: 4,
	}
	raw, err := json.Marshal(&mi)
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Unix(1700, 0).UTC()
	row := database.ListDebatesPublicFeedRow{
		ID:         42,
		MatchID:    "ext-match-1",
		DebateType: "post_match",
		Headline:   "Headline",
		MatchInfo:  pqtype.NullRawMessage{Valid: true, RawMessage: raw},
		CreatedAt:  sql.NullTime{Time: ts, Valid: true},
		TotalVotes: sql.NullInt32{Int32: 1, Valid: true},
	}
	mock := &mockDebatesFeedStore{publicFeedRows: []database.ListDebatesPublicFeedRow{row}}

	db, sqlMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	c := &Config{DebatesFeedDB: mock, DBConn: db}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debates/public-feed", nil)
	c.getDebatesPublicFeed(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("matches/teams query should not run when match_info supplies teams: %v", err)
	}
	var out PublicDebateFeedResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Debates) != 1 {
		t.Fatalf("debates len = %d", len(out.Debates))
	}
	teams := out.Debates[0].Teams
	if teams == nil {
		t.Fatal("expected teams from match_info")
	}
	if teams.Home.Name != "Stored Home" || teams.Away.Name != "Stored Away" {
		t.Fatalf("teams: %+v", teams)
	}
}

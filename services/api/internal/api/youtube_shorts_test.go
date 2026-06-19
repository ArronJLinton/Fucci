package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
	"github.com/go-chi/chi"
)

type stubChannel struct {
	id       string
	verified bool
}

type stubChannelStore struct {
	byKey map[string]stubChannel
}

func (s *stubChannelStore) GetTeamYouTubeChannelByLookupKey(_ context.Context, lookupKey string) (database.TeamYoutubeChannels, error) {
	ch, ok := s.byKey[lookupKey]
	if !ok {
		return database.TeamYoutubeChannels{}, errors.New("not found")
	}
	return database.TeamYoutubeChannels{
		ChannelID:  ch.id,
		IsVerified: ch.verified,
	}, nil
}

type stubShortsFetcher struct {
	byChannel map[string][]youtube.Short
}

func (s *stubShortsFetcher) FetchShortsForChannel(_ context.Context, channelID string) ([]youtube.Short, error) {
	return s.byChannel[channelID], nil
}

func TestGetMatchYouTubeShorts_ReturnsBothTeams(t *testing.T) {
	homeShort := youtube.Short{VideoID: "home1", Title: "Home", Duration: "PT30S"}
	awayShort := youtube.Short{VideoID: "away1", Title: "Away", Duration: "PT40S"}

	cfg := &Config{
		MatchInfoLookup: func(_ context.Context, matchID string) (*MatchInfo, error) {
			if matchID != "1489391" {
				t.Fatalf("unexpected match id %q", matchID)
			}
			return &MatchInfo{HomeTeam: "USA", AwayTeam: "Australia"}, nil
		},
		YouTubeShortsService: &youtube.Service{
			Channels: &stubChannelStore{
				byKey: map[string]stubChannel{
					"united states": {id: "UC-USA", verified: true},
					"australia":     {id: "UC-AUS", verified: true},
				},
			},
			Fetcher: &stubShortsFetcher{
				byChannel: map[string][]youtube.Short{
					"UC-USA": {homeShort},
					"UC-AUS": {awayShort},
				},
			},
		},
	}

	router := chi.NewRouter()
	router.Get("/matches/{matchId}/stories/shorts", cfg.getMatchYouTubeShorts)

	req := httptest.NewRequest(http.MethodGet, "/matches/1489391/stories/shorts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body matchShortsResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.MatchID != "1489391" {
		t.Fatalf("match_id = %q", body.MatchID)
	}
	if !body.Teams.Home.HasShorts || len(body.Teams.Home.Shorts) != 1 {
		t.Fatalf("home = %+v", body.Teams.Home)
	}
	if !body.Teams.Away.HasShorts || len(body.Teams.Away.Shorts) != 1 {
		t.Fatalf("away = %+v", body.Teams.Away)
	}
	if body.Teams.Home.LookupKey != "united states" {
		t.Fatalf("home lookup_key = %q", body.Teams.Home.LookupKey)
	}
}

func TestGetMatchYouTubeShorts_MatchNotFound(t *testing.T) {
	cfg := &Config{
		MatchInfoLookup: func(context.Context, string) (*MatchInfo, error) {
			return nil, errors.New("fixture not found")
		},
	}

	router := chi.NewRouter()
	router.Get("/matches/{matchId}/stories/shorts", cfg.getMatchYouTubeShorts)

	req := httptest.NewRequest(http.MethodGet, "/matches/bad/stories/shorts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

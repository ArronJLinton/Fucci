package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/ai"
	"github.com/ArronJLinton/fucci-api/internal/news"
)

type DebateDataAggregator struct {
	Config *Config
}

type MatchDataRequest struct {
	MatchID         string `json:"match_id"`
	HomeTeam        string `json:"home_team"`
	AwayTeam        string `json:"away_team"`
	Date            string `json:"date"`
	Status          string `json:"status"`
	HomeScore       int    `json:"home_score"`
	AwayScore       int    `json:"away_score"`
	HomeGoals       int    `json:"home_goals"`
	AwayGoals       int    `json:"away_goals"`
	HomeShots       int    `json:"home_shots"`
	AwayShots       int    `json:"away_shots"`
	HomePossession  int    `json:"home_possession"`
	AwayPossession  int    `json:"away_possession"`
	HomeFouls       int    `json:"home_fouls"`
	AwayFouls       int    `json:"away_fouls"`
	HomeYellowCards int    `json:"home_yellow_cards"`
	AwayYellowCards int    `json:"away_yellow_cards"`
	HomeRedCards    int    `json:"home_red_cards"`
	AwayRedCards    int    `json:"away_red_cards"`
	Venue           string `json:"venue"`
	League          string `json:"league"`
	Season          string `json:"season"`
	LeagueID        int    `json:"league_id"`
	SeasonYear      int    `json:"season_year"`
	HomeTeamID      int    `json:"home_team_id"`
	AwayTeamID      int    `json:"away_team_id"`
}

func NewDebateDataAggregator(config *Config) *DebateDataAggregator {
	return &DebateDataAggregator{
		Config: config,
	}
}

// AggregateMatchData fetches and combines data from multiple sources for debate generation
func (dda *DebateDataAggregator) AggregateMatchData(ctx context.Context, matchReq MatchDataRequest) (*ai.MatchData, error) {
	matchData := &ai.MatchData{
		MatchID:  matchReq.MatchID,
		HomeTeam: matchReq.HomeTeam,
		AwayTeam: matchReq.AwayTeam,
		Date:     matchReq.Date,
		Status:   matchReq.Status,
		Venue:    matchReq.Venue,
		League:   matchReq.League,
		Season:   matchReq.Season,
	}

	// Create enhanced stats with the data we already have
	enhancedStats := &ai.MatchStats{
		HomeScore:       matchReq.HomeScore,
		AwayScore:       matchReq.AwayScore,
		HomeGoals:       matchReq.HomeGoals,
		AwayGoals:       matchReq.AwayGoals,
		HomeShots:       matchReq.HomeShots,
		AwayShots:       matchReq.AwayShots,
		HomePossession:  matchReq.HomePossession,
		AwayPossession:  matchReq.AwayPossession,
		HomeFouls:       matchReq.HomeFouls,
		AwayFouls:       matchReq.AwayFouls,
		HomeYellowCards: matchReq.HomeYellowCards,
		AwayYellowCards: matchReq.AwayYellowCards,
		HomeRedCards:    matchReq.HomeRedCards,
		AwayRedCards:    matchReq.AwayRedCards,
	}

	// Fetch lineups if match is upcoming or in progress (optional: continue with other sources if unavailable)
	if matchReq.Status == "NS" || matchReq.Status == "1H" || matchReq.Status == "2H" || matchReq.Status == "HT" {
		lineups, err := dda.fetchLineups(ctx, matchReq.MatchID, matchReq.HomeTeamID, matchReq.AwayTeamID)
		if err != nil {
			log.Printf("Lineup data unavailable for match %s, continuing with other sources: %v", matchReq.MatchID, err)
		} else {
			matchData.Lineups = lineups
		}
	}

	// Fetch detailed match statistics if match is finished or in progress
	if matchReq.Status == "FT" || matchReq.Status == "AET" || matchReq.Status == "PEN" ||
		matchReq.Status == "1H" || matchReq.Status == "2H" || matchReq.Status == "HT" {
		detailedStats, err := dda.fetchMatchStats(ctx, matchReq.MatchID, matchReq.HomeTeamID, matchReq.AwayTeamID)
		if err != nil {
			log.Printf("Match stats unavailable for match %s, continuing with other sources: %v", matchReq.MatchID, err)
		} else {
			// Merge detailed stats with basic stats
			enhancedStats.HomeShots = detailedStats.HomeShots
			enhancedStats.AwayShots = detailedStats.AwayShots
			enhancedStats.HomePossession = detailedStats.HomePossession
			enhancedStats.AwayPossession = detailedStats.AwayPossession
			enhancedStats.HomeFouls = detailedStats.HomeFouls
			enhancedStats.AwayFouls = detailedStats.AwayFouls
			enhancedStats.HomeYellowCards = detailedStats.HomeYellowCards
			enhancedStats.AwayYellowCards = detailedStats.AwayYellowCards
			enhancedStats.HomeRedCards = detailedStats.HomeRedCards
			enhancedStats.AwayRedCards = detailedStats.AwayRedCards
		}
	}

	// Set the enhanced stats
	matchData.Stats = enhancedStats

	// Fetch news headlines via news module (optional: continue with other sources if rate-limited or unavailable)
	headlines, err := dda.fetchNewsHeadlines(ctx, matchReq.HomeTeam, matchReq.AwayTeam, matchReq.Status)
	if err != nil {
		log.Printf("News headlines unavailable, continuing with other sources: %v", err)
	} else {
		matchData.NewsHeadlines = headlines
	}

	// Fetch social media sentiment
	sentiment, err := dda.fetchSocialSentiment(ctx, matchReq.HomeTeam, matchReq.AwayTeam, matchReq.MatchID)
	if err != nil {
		log.Printf("[debate] social sentiment unavailable match_id=%s home=%s away=%s: %v", matchReq.MatchID, matchReq.HomeTeam, matchReq.AwayTeam, err)
	} else {
		matchData.SocialSentiment = sentiment
	}

	// Fetch head-to-head when both team IDs are available (uses API-Football via Config)
	if matchReq.HomeTeamID != 0 && matchReq.AwayTeamID != 0 {
		h2h, err := dda.Config.FetchHeadToHead(ctx, matchReq.HomeTeamID, matchReq.AwayTeamID)
		if err != nil {
			log.Printf("Failed to fetch head-to-head for %d vs %d: %v", matchReq.HomeTeamID, matchReq.AwayTeamID, err)
		} else {
			matchData.HeadToHeadSummary = h2h
		}
	}

	// Fetch league standings when league and season are available (uses API-Football via Config)
	if matchReq.LeagueID != 0 && matchReq.SeasonYear != 0 {
		data, err := dda.Config.GetLeagueStandingsData(ctx, fmt.Sprintf("%d", matchReq.LeagueID), fmt.Sprintf("%d", matchReq.SeasonYear))
		if err != nil {
			log.Printf("Failed to fetch league standings for league %d season %d: %v", matchReq.LeagueID, matchReq.SeasonYear, err)
		} else {
			matchData.LeagueTableSummary = FormatLeagueStandingsSummary(data)
		}
	}

	return matchData, nil
}

// fetchLineups gets lineup data for a match via Config.FetchLineupData (shared with futbol handler).
// Maps each response block to home/away by team.id when homeTeamID/awayTeamID are set; otherwise falls back to index order.
func (dda *DebateDataAggregator) fetchLineups(ctx context.Context, matchID string, homeTeamID, awayTeamID int) (*ai.LineupData, error) {
	raw, err := dda.Config.FetchLineupData(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if len(raw.Response) < 2 {
		return nil, fmt.Errorf("insufficient lineup data")
	}

	homeIdx, awayIdx := 0, 1
	for i := range raw.Response {
		tid := raw.Response[i].Team.ID
		if homeTeamID != 0 && tid == homeTeamID {
			homeIdx = i
		}
		if awayTeamID != 0 && tid == awayTeamID {
			awayIdx = i
		}
	}
	if homeIdx == awayIdx {
		// One ID matched or neither: assign the other slot to the remaining block
		if homeTeamID != 0 || awayTeamID != 0 {
			if homeTeamID != 0 {
				awayIdx = 1 - homeIdx
			} else {
				homeIdx = 1 - awayIdx
			}
		} else {
			homeIdx, awayIdx = 0, 1
		}
	}

	home := &raw.Response[homeIdx]
	away := &raw.Response[awayIdx]

	lineupData := &ai.LineupData{}
	for _, p := range home.StartXI {
		lineupData.HomeStarters = append(lineupData.HomeStarters, apiPlayerToAIPlayer(p.Player))
	}
	for _, p := range home.Substitutes {
		lineupData.HomeSubstitutes = append(lineupData.HomeSubstitutes, ai.Player{ID: p.Player.ID, Name: p.Player.Name, Number: p.Player.Number, Pos: p.Player.Pos, Photo: p.Player.Photo})
	}
	for _, p := range away.StartXI {
		lineupData.AwayStarters = append(lineupData.AwayStarters, apiPlayerToAIPlayer(p.Player))
	}
	for _, p := range away.Substitutes {
		lineupData.AwaySubstitutes = append(lineupData.AwaySubstitutes, ai.Player{ID: p.Player.ID, Name: p.Player.Name, Number: p.Player.Number, Pos: p.Player.Pos, Photo: p.Player.Photo})
	}

	return lineupData, nil
}

// apiPlayerToAIPlayer maps api.Player to ai.Player (ID, Name, Number, Pos, Photo).
func apiPlayerToAIPlayer(p Player) ai.Player {
	return ai.Player{ID: p.ID, Name: p.Name, Number: p.Number, Pos: p.Pos, Photo: p.Photo}
}

// fetchMatchStats gets match statistics via Config.FetchMatchStatsData (shared with API-Football).
// Maps response blocks to home/away by team.id when homeTeamID/awayTeamID are set; otherwise falls back to index order.
func (dda *DebateDataAggregator) fetchMatchStats(ctx context.Context, matchID string, homeTeamID, awayTeamID int) (*ai.MatchStats, error) {
	raw, err := dda.Config.FetchMatchStatsData(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if len(raw.Response) < 2 {
		return nil, fmt.Errorf("insufficient stats data")
	}

	homeIdx, awayIdx := 0, 1
	for i := range raw.Response {
		tid := raw.Response[i].Team.ID
		if homeTeamID != 0 && tid == homeTeamID {
			homeIdx = i
		}
		if awayTeamID != 0 && tid == awayTeamID {
			awayIdx = i
		}
	}
	if homeIdx == awayIdx {
		if homeTeamID != 0 || awayTeamID != 0 {
			if homeTeamID != 0 {
				awayIdx = 1 - homeIdx
			} else {
				homeIdx = 1 - awayIdx
			}
		} else {
			homeIdx, awayIdx = 0, 1
		}
	}

	homeStats := raw.Response[homeIdx].Statistics
	awayStats := raw.Response[awayIdx].Statistics

	stats := &ai.MatchStats{}
	stats.HomeGoals = getNumericStat(homeStats, "Goals")
	stats.HomeShots = getNumericStat(homeStats, "Total Shots")
	stats.HomePossession = getNumericStat(homeStats, "Ball Possession")
	stats.HomeFouls = getNumericStat(homeStats, "Fouls")
	stats.HomeYellowCards = getNumericStat(homeStats, "Yellow Cards")
	stats.HomeRedCards = getNumericStat(homeStats, "Red Cards")

	stats.AwayGoals = getNumericStat(awayStats, "Goals")
	stats.AwayShots = getNumericStat(awayStats, "Total Shots")
	stats.AwayPossession = getNumericStat(awayStats, "Ball Possession")
	stats.AwayFouls = getNumericStat(awayStats, "Fouls")
	stats.AwayYellowCards = getNumericStat(awayStats, "Yellow Cards")
	stats.AwayRedCards = getNumericStat(awayStats, "Red Cards")

	return stats, nil
}

func getNumericStat(statistics []FixtureStatEntry, statType string) int {
	for _, stat := range statistics {
		if stat.Type == statType {
			return valueToInt(stat.Value)
		}
	}
	return 0
}

// valueToInt converts API-Football stat value to int. Statistics often come as float64, int, string (e.g. "55%"), or json.Number.
func valueToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			if f, err2 := val.Float64(); err2 == nil {
				return int(f)
			}
			return 0
		}
		return int(n)
	case string:
		s := strings.TrimSpace(val)
		s = strings.TrimSuffix(s, "%")
		s = strings.TrimSpace(s)
		// Take leading numeric part in case of "55.5" or "55"
		if i := strings.IndexAny(s, " \t"); i >= 0 {
			s = s[:i]
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			if f, err2 := strconv.ParseFloat(s, 64); err2 == nil {
				return int(f)
			}
			return 0
		}
		return int(n)
	default:
		return 0
	}
}

// fetchNewsHeadlines gets match news via the news module's FetchMatchNews (Real-Time News Data API).
func (dda *DebateDataAggregator) fetchNewsHeadlines(ctx context.Context, homeTeam, awayTeam, matchStatus string) ([]string, error) {
	newsClient := news.NewClient(dda.Config.RapidAPIKey)
	if dda.Config.NewsBaseURL != "" {
		newsClient = news.NewClientWithBaseURL(dda.Config.RapidAPIKey, dda.Config.NewsBaseURL)
	}

	limit := 10
	resp, err := newsClient.FetchMatchNews(ctx, homeTeam, awayTeam, limit, matchStatus, nil)
	if err != nil {
		return nil, err
	}

	headlines := make([]string, 0, len(resp.Data))
	for _, article := range resp.Data {
		if article.Title != "" {
			headlines = append(headlines, article.Title)
		}
	}
	return headlines, nil
}

// fetchSocialSentiment gets social media sentiment data
func (dda *DebateDataAggregator) fetchSocialSentiment(ctx context.Context, homeTeam, awayTeam, matchID string) (*ai.SocialSentiment, error) {
	sentiment := &ai.SocialSentiment{
		TwitterSentiment:     0.0,
		RedditSentiment:      0.0,
		TopTopics:            []string{},
		ControversialMoments: []string{},
	}

	// Fetch Twitter sentiment for home team
	homeTwitterSentiment, err := dda.fetchTwitterSentiment(ctx, homeTeam)
	if err != nil {
		fmt.Printf("Failed to fetch home team Twitter sentiment: %v\n", err)
	} else {
		sentiment.TwitterSentiment = (sentiment.TwitterSentiment + homeTwitterSentiment) / 2
	}

	// Fetch Twitter sentiment for away team
	awayTwitterSentiment, err := dda.fetchTwitterSentiment(ctx, awayTeam)
	if err != nil {
		fmt.Printf("Failed to fetch away team Twitter sentiment: %v\n", err)
	} else {
		sentiment.TwitterSentiment = (sentiment.TwitterSentiment + awayTwitterSentiment) / 2
	}

	// For now, we'll use placeholder data for Reddit sentiment
	// You can implement Reddit API integration later
	sentiment.RedditSentiment = 0.0

	// Extract top topics from team names
	sentiment.TopTopics = []string{
		fmt.Sprintf("%s performance", homeTeam),
		fmt.Sprintf("%s performance", awayTeam),
		"match analysis",
		"team tactics",
	}

	return sentiment, nil
}

// fetchTwitterSentiment gets Twitter sentiment for a team
func (dda *DebateDataAggregator) fetchTwitterSentiment(ctx context.Context, teamName string) (float64, error) {
	// Use your existing Twitter API endpoint
	// For now, return a placeholder sentiment value
	// You can implement actual Twitter sentiment analysis later

	// Placeholder: return a neutral sentiment
	return 0.0, nil
}

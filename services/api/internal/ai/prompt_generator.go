package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PromptGenerator struct {
	OpenAIKey     string
	OpenAIBaseURL string
	Cache         CacheInterface
}

type CacheInterface interface {
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
}

type MatchData struct {
	MatchID            string           `json:"match_id"`
	HomeTeam           string           `json:"home_team"`
	AwayTeam           string           `json:"away_team"`
	Date               string           `json:"date"`
	Status             string           `json:"status"`
	Lineups            *LineupData      `json:"lineups,omitempty"`
	Stats              *MatchStats      `json:"stats,omitempty"`
	NewsHeadlines      []string         `json:"news_headlines,omitempty"`
	SocialSentiment    *SocialSentiment `json:"social_sentiment,omitempty"`
	Venue              string           `json:"venue,omitempty"`
	League             string           `json:"league,omitempty"`
	Season             string           `json:"season,omitempty"`
	HeadToHeadSummary  string           `json:"head_to_head_summary,omitempty"`
	LeagueTableSummary string           `json:"league_table_summary,omitempty"`
}

type LineupData struct {
	HomeStarters    []Player `json:"home_starters"`
	HomeSubstitutes []Player `json:"home_substitutes"`
	AwayStarters    []Player `json:"away_starters"`
	AwaySubstitutes []Player `json:"away_substitutes"`
}

type Player struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
	Pos    string `json:"pos"`
	Photo  string `json:"photo"`
}

type MatchStats struct {
	HomeScore       int `json:"home_score"`
	AwayScore       int `json:"away_score"`
	HomeGoals       int `json:"home_goals"`
	AwayGoals       int `json:"away_goals"`
	HomeShots       int `json:"home_shots"`
	AwayShots       int `json:"away_shots"`
	HomePossession  int `json:"home_possession"`
	AwayPossession  int `json:"away_possession"`
	HomeFouls       int `json:"home_fouls"`
	AwayFouls       int `json:"away_fouls"`
	HomeYellowCards int `json:"home_yellow_cards"`
	AwayYellowCards int `json:"away_yellow_cards"`
	HomeRedCards    int `json:"home_red_cards"`
	AwayRedCards    int `json:"away_red_cards"`
}

type SocialSentiment struct {
	TwitterSentiment     float64  `json:"twitter_sentiment"` // -1 to 1
	RedditSentiment      float64  `json:"reddit_sentiment"`  // -1 to 1
	TopTopics            []string `json:"top_topics"`
	ControversialMoments []string `json:"controversial_moments"`
}

type DebatePrompt struct {
	Headline    string       `json:"headline"`
	Description string       `json:"description"`
	Cards       []DebateCard `json:"cards"`
}

type DebateCard struct {
	Stance      string `json:"stance"` // "agree", "disagree", "wildcard"
	Title       string `json:"title"`
	Description string `json:"description"`
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewPromptGenerator(openAIKey, openAIBaseURL string, cache CacheInterface) *PromptGenerator {
	if openAIBaseURL == "" {
		openAIBaseURL = "https://api.openai.com/v1"
	}

	return &PromptGenerator{
		OpenAIKey:     openAIKey,
		OpenAIBaseURL: openAIBaseURL,
		Cache:         cache,
	}
}

func (pg *PromptGenerator) GeneratePreMatchPrompt(ctx context.Context, matchData MatchData) (*DebatePrompt, error) {
	cacheKey := fmt.Sprintf("pre_match_prompt:%s", matchData.MatchID)

	// Try cache first
	var cachedPrompt DebatePrompt
	exists, err := pg.Cache.Exists(ctx, cacheKey)
	if err == nil && exists {
		err = pg.Cache.Get(ctx, cacheKey, &cachedPrompt)
		if err == nil {
			return &cachedPrompt, nil
		}
	}

	// Generate new prompt
	prompt, err := pg.generatePrompt(ctx, matchData, "pre_match")
	if err != nil {
		return nil, err
	}

	// Cache the result
	err = pg.Cache.Set(ctx, cacheKey, prompt, 24*time.Hour)
	if err != nil {
		fmt.Printf("Failed to cache pre-match prompt: %v\n", err)
	}

	return prompt, nil
}

func (pg *PromptGenerator) GeneratePostMatchPrompt(ctx context.Context, matchData MatchData) (*DebatePrompt, error) {
	cacheKey := fmt.Sprintf("post_match_prompt:%s", matchData.MatchID)

	// Try cache first
	var cachedPrompt DebatePrompt
	exists, err := pg.Cache.Exists(ctx, cacheKey)
	if err == nil && exists {
		err = pg.Cache.Get(ctx, cacheKey, &cachedPrompt)
		if err == nil {
			return &cachedPrompt, nil
		}
	}

	// Generate new prompt
	prompt, err := pg.generatePrompt(ctx, matchData, "post_match")
	if err != nil {
		return nil, err
	}

	// Cache the result
	pg.Cache.Set(ctx, cacheKey, prompt, 24*time.Hour)

	return prompt, nil
}

func (pg *PromptGenerator) generatePrompt(ctx context.Context, matchData MatchData, promptType string) (*DebatePrompt, error) {
	systemPrompt := pg.buildSystemPrompt(promptType)
	userPrompt := pg.buildUserPrompt(matchData, promptType)

	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	response, err := pg.callOpenAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Parse the response (strip markdown code fences if present; models often wrap JSON in ```json ... ```)
	content := extractJSONFromContent(response.Choices[0].Message.Content)
	var prompt DebatePrompt
	err = json.Unmarshal([]byte(content), &prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	return &prompt, nil
}

// extractJSONFromContent returns the first JSON object from content, stripping optional markdown code fences.
func extractJSONFromContent(content string) string {
	s := strings.TrimSpace(content)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

func (pg *PromptGenerator) buildSystemPrompt(promptType string) string {
	if promptType == "pre_match" {
		return `You are a football debate prompt generator. Create engaging, controversial debate topics for PRE-MATCH discussions.

IMPORTANT: This is a PRE-MATCH debate. The match has NOT happened yet. Focus on predictions, expectations, and pre-match analysis.

Respond with ONLY a single JSON object (no markdown, no code fences, no extra text). Use this structure:
{
  "headline": "A compelling, controversial headline that will spark debate",
  "description": "A brief description providing context for the debate",
  "cards": [
    {
      "stance": "agree",
      "title": "Title for the agree stance",
      "description": "Brief description supporting this stance"
    },
    {
      "stance": "disagree", 
      "title": "Title for the disagree stance",
      "description": "Brief description supporting this stance"
    },
    {
      "stance": "wildcard",
      "title": "Title for a wildcard/unexpected stance",
      "description": "Brief description for an unexpected perspective"
    }
  ]
}

Focus on:
- Lineup decisions and tactical choices
- Player form and selection controversies
- Managerial decisions
- Bold predictions about what will happen
- Historical context and rivalries
- Pre-match expectations and concerns

DO NOT reference match results, final scores, or post-match analysis since the match hasn't happened yet.

Make the debate engaging and controversial but respectful.`
	} else {
		return `You are a football debate prompt generator. Create engaging, controversial debate topics for POST-MATCH discussions.

IMPORTANT: This is a POST-MATCH debate. The match has already happened. Focus on analysis of what occurred.

Respond with ONLY a single JSON object (no markdown, no code fences, no extra text). Use this structure:
{
  "headline": "A compelling, controversial headline that will spark debate",
  "description": "A brief description providing context for the debate",
  "cards": [
    {
      "stance": "agree",
      "title": "Title for the agree stance", 
      "description": "Brief description supporting this stance"
    },
    {
      "stance": "disagree",
      "title": "Title for the disagree stance",
      "description": "Brief description supporting this stance"
    },
    {
      "stance": "wildcard",
      "title": "Title for a wildcard/unexpected stance",
      "description": "Brief description for an unexpected perspective"
    }
  ]
}

Focus on:
- Key moments and turning points from the match
- Controversial decisions (refereeing, VAR)
- Player performances and impact
- Tactical changes and their effectiveness
- Social media reactions and fan sentiment
- Post-match analysis and what-ifs
- Analysis of the final result

Make the debate engaging and controversial but respectful.`
	}
}

func (pg *PromptGenerator) buildUserPrompt(matchData MatchData, promptType string) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Generate a %s debate prompt for this match:\n\n", promptType))
	prompt.WriteString(fmt.Sprintf("Match: %s vs %s\n", matchData.HomeTeam, matchData.AwayTeam))
	prompt.WriteString(fmt.Sprintf("Date: %s\n", matchData.Date))
	prompt.WriteString(fmt.Sprintf("Status: %s\n", matchData.Status))

	// Add venue, league, and season information if available
	if matchData.Venue != "" {
		prompt.WriteString(fmt.Sprintf("Venue: %s\n", matchData.Venue))
	}
	if matchData.League != "" {
		prompt.WriteString(fmt.Sprintf("League: %s\n", matchData.League))
	}
	if matchData.Season != "" {
		prompt.WriteString(fmt.Sprintf("Season: %s\n", matchData.Season))
	}
	prompt.WriteString("\n")

	if matchData.HeadToHeadSummary != "" {
		prompt.WriteString("HEAD-TO-HEAD:\n")
		prompt.WriteString(matchData.HeadToHeadSummary)
		prompt.WriteString("\n\n")
	}

	if matchData.LeagueTableSummary != "" {
		prompt.WriteString("LEAGUE TABLE:\n")
		prompt.WriteString(matchData.LeagueTableSummary)
		prompt.WriteString("\n\n")
	}

	if matchData.Lineups != nil {
		prompt.WriteString("LINEUPS:\n")
		prompt.WriteString("Home Starters: ")
		for i, player := range matchData.Lineups.HomeStarters {
			if i > 0 {
				prompt.WriteString(", ")
			}
			prompt.WriteString(fmt.Sprintf("%s (%s)", player.Name, player.Pos))
		}
		prompt.WriteString("\n")

		prompt.WriteString("Away Starters: ")
		for i, player := range matchData.Lineups.AwayStarters {
			if i > 0 {
				prompt.WriteString(", ")
			}
			prompt.WriteString(fmt.Sprintf("%s (%s)", player.Name, player.Pos))
		}
		prompt.WriteString("\n\n")
	}

	if matchData.Stats != nil {
		prompt.WriteString("MATCH STATS:\n")
		if promptType == "post_match" {
			// For post-match debates, show final scores and stats
			prompt.WriteString(fmt.Sprintf("Final Score: %d-%d\n", matchData.Stats.HomeScore, matchData.Stats.AwayScore))
			prompt.WriteString(fmt.Sprintf("Shots: %d-%d\n", matchData.Stats.HomeShots, matchData.Stats.AwayShots))
			prompt.WriteString(fmt.Sprintf("Possession: %d%%-%d%%\n", matchData.Stats.HomePossession, matchData.Stats.AwayPossession))
			prompt.WriteString(fmt.Sprintf("Fouls: %d-%d\n", matchData.Stats.HomeFouls, matchData.Stats.AwayFouls))
			prompt.WriteString(fmt.Sprintf("Cards: Yellow(%d-%d) Red(%d-%d)\n\n",
				matchData.Stats.HomeYellowCards, matchData.Stats.AwayYellowCards,
				matchData.Stats.HomeRedCards, matchData.Stats.AwayRedCards))
		} else {
			// For pre-match debates, show current form or recent stats if available
			if matchData.Stats.HomeShots > 0 || matchData.Stats.AwayShots > 0 {
				prompt.WriteString(fmt.Sprintf("Recent Form - Shots: %d-%d\n", matchData.Stats.HomeShots, matchData.Stats.AwayShots))
			}
			if matchData.Stats.HomePossession > 0 || matchData.Stats.AwayPossession > 0 {
				prompt.WriteString(fmt.Sprintf("Recent Form - Possession: %d%%-%d%%\n", matchData.Stats.HomePossession, matchData.Stats.AwayPossession))
			}
			prompt.WriteString("\n")
		}
	}

	if len(matchData.NewsHeadlines) > 0 {
		prompt.WriteString("NEWS HEADLINES:\n")
		for _, headline := range matchData.NewsHeadlines {
			prompt.WriteString(fmt.Sprintf("- %s\n", headline))
		}
		prompt.WriteString("\n")
	}

	if matchData.SocialSentiment != nil {
		prompt.WriteString("SOCIAL SENTIMENT:\n")
		prompt.WriteString(fmt.Sprintf("Twitter Sentiment: %.2f\n", matchData.SocialSentiment.TwitterSentiment))
		prompt.WriteString(fmt.Sprintf("Reddit Sentiment: %.2f\n", matchData.SocialSentiment.RedditSentiment))

		if len(matchData.SocialSentiment.TopTopics) > 0 {
			prompt.WriteString("Top Topics: ")
			prompt.WriteString(strings.Join(matchData.SocialSentiment.TopTopics, ", "))
			prompt.WriteString("\n")
		}

		if len(matchData.SocialSentiment.ControversialMoments) > 0 {
			prompt.WriteString("Controversial Moments:\n")
			for _, moment := range matchData.SocialSentiment.ControversialMoments {
				prompt.WriteString(fmt.Sprintf("- %s\n", moment))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Generate a compelling debate prompt based on this information. Return only valid JSON.")

	return prompt.String()
}

func (pg *PromptGenerator) callOpenAI(ctx context.Context, request OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/chat/completions", pg.OpenAIBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+pg.OpenAIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	var response OpenAIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	return &response, nil
}

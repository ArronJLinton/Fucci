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
	// Three seeded thread starters: pro agree, pro disagree, wildcard/hot-take (passionate fan voice).
	Comments []string `json:"comments"`
}

type DebateCard struct {
	Stance      string `json:"stance"` // "agree", "disagree" (one binary vote per debate)
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// NormalizeDebatePrompt keeps one agree + one disagree card, fills empty descriptions from title, caps comments at 3.
func NormalizeDebatePrompt(p *DebatePrompt) {
	if p == nil {
		return
	}
	var agreeCard, disagreeCard DebateCard
	var hasAgree, hasDisagree bool
	for _, c := range p.Cards {
		if c.Stance == "agree" && strings.TrimSpace(c.Title) != "" && !hasAgree {
			hasAgree = true
			agreeCard = c
		}
		if c.Stance == "disagree" && strings.TrimSpace(c.Title) != "" && !hasDisagree {
			hasDisagree = true
			disagreeCard = c
		}
	}
	p.Cards = nil
	if hasAgree {
		if strings.TrimSpace(agreeCard.Description) == "" {
			agreeCard.Description = agreeCard.Title
		}
		p.Cards = append(p.Cards, agreeCard)
	}
	if hasDisagree {
		if strings.TrimSpace(disagreeCard.Description) == "" {
			disagreeCard.Description = disagreeCard.Title
		}
		p.Cards = append(p.Cards, disagreeCard)
	}
	if len(p.Comments) > 3 {
		p.Comments = p.Comments[:3]
	}
}

// DebatePromptBinaryOK is true when there is a headline and exactly one usable agree + disagree card pair.
func DebatePromptBinaryOK(p *DebatePrompt) bool {
	if p == nil || strings.TrimSpace(p.Headline) == "" {
		return false
	}
	var hasAgree, hasDisagree bool
	for _, c := range p.Cards {
		if c.Stance == "agree" && strings.TrimSpace(c.Title) != "" {
			hasAgree = true
		}
		if c.Stance == "disagree" && strings.TrimSpace(c.Title) != "" {
			hasDisagree = true
		}
	}
	return hasAgree && hasDisagree
}

// DefaultDebateSetCount is the default number of debates returned per type when generating a set.
const DefaultDebateSetCount = 3

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

	NormalizeDebatePrompt(&prompt)
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

// extractJSONArrayFromContent returns the first JSON array from content, stripping optional markdown code fences.
func extractJSONArrayFromContent(content string) string {
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
		return `You are a football debate prompt generator for a mobile app where each debate gets ONE community vote: users swipe agree or disagree with a single proposition.

IMPORTANT: PRE-MATCH — the match has NOT happened yet. No final scores or post-match outcomes.

The headline must read as a clear statement or question fans can answer YES or NO to (agree vs disagree). Frame it so "agree" and "disagree" are natural opposites.

Respond with ONLY one JSON object (no markdown, no code fences, no extra text). Exact shape:
{
  "headline": "Bold, controversial line fans can agree or disagree with (e.g. claim or polarizing question)",
  "description": "Short context that raises the stakes (why it matters for this fixture)",
  "cards": [
    { "stance": "agree", "title": "Short label for the YES / agree side (e.g. how you'd vote if you buy the headline)" },
    { "stance": "disagree", "title": "Short label for the NO / disagree side (e.g. how you'd vote if you reject the headline)" }
  ],
  "comments": [
    "First comment: passionate fan voice backing the agree side",
    "Second comment: passionate fan voice backing the disagree side",
    "Third comment: spicy wildcard / hot-take or angle that still fits the debate (not analyst-speak)"
  ]
}

Rules:
- Exactly two cards: only "agree" and "disagree". No wildcard card. No third card.
- Card objects use only "stance" and "title" (no per-card description field).
- Exactly three strings in "comments". They seed the comments section before real users post.
- Comments must sound like real supporters in the stands or group chat: emotional, direct, maybe messy, NEVER like a polished TV pundit. Avoid jargon stacks (e.g. don't lean on "low block", "xG", "progressive carries" unless a normal fan would say it). Short clauses, heat, banter, and belief are good.
- Keep everything PG-13: no slurs, threats, hate, or harassment.

Topic angles for pre-match:
- Lineups, form, pressure, predictions, rivalries, manager calls, expectations.

DO NOT reference final score or "after the match" facts.`
	}
	return `You are a football debate prompt generator for a mobile app where each debate gets ONE community vote: users swipe agree or disagree with a single proposition.

IMPORTANT: POST-MATCH — the match has finished. Use what happened; reference result and moments when useful.

The headline must read as a clear statement or question fans can answer YES or NO to. "Agree" and "disagree" must be direct opposites.

Respond with ONLY one JSON object (no markdown, no code fences, no extra text). Exact shape:
{
  "headline": "Bold line fans can agree or disagree with",
  "description": "Short context from the match that fuels the argument",
  "cards": [
    { "stance": "agree", "title": "Short label for the YES / agree side" },
    { "stance": "disagree", "title": "Short label for the NO / disagree side" }
  ],
  "comments": [
    "Passionate fan comment supporting the agree side",
    "Passionate fan comment supporting the disagree side",
    "Wildcard / hot-take comment (still about this debate; fan voice not pundit voice)"
  ]
}

Rules:
- Exactly two cards: only "agree" and "disagree". No wildcard card.
- Card objects: only "stance" and "title".
- Exactly three "comments" strings for seeded replies.
- Comments: passionate fan energy, engaging, conversational; NOT polished analyst prose or heavy tactical jargon unless a fan would naturally say it.
- PG-13 only.

Good angles: turning points, calls, performances, blame, praise, narratives after the result.`
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

// maxTokensForDebateSet is the max tokens for a single AI response returning multiple debates.
const maxTokensForDebateSet = 2800

// buildSystemPromptForSet returns a system prompt that asks for an array of N debate prompts.
func (pg *PromptGenerator) buildSystemPromptForSet(promptType string, count int) string {
	phase := "PRE-MATCH"
	phaseNote := "The match has NOT happened yet. Focus on predictions, pressure, expectations, possible outcomes, tactical storylines, player narratives, and what is at stake."
	if promptType == "post_match" {
		phase = "POST-MATCH"
		phaseNote = "The match has already happened. Focus on what actually happened, who delivered, who failed, tactical consequences, emotional fallout, blame, praise, and legacy-defining takeaways."
	}

	return fmt.Sprintf(`You are an elite football debate producer for a mobile app: ONE vote per debate (agree vs disagree on a single proposition).

	IMPORTANT: This is a %s debate. %s

	Each debate is binary: a headline fans can agree or disagree with, two stance labels (agree/disagree), and three seeded comments in a passionate FAN voice (not polished pundit copy; avoid analyst jargon unless a normal supporter would say it). Comments should feel like stands or group-chat energy: short, heated, believable, PG-13.

	JSON rules for EVERY object in the array:
	- Exactly two cards: { "stance": "agree", "title": "..." } and { "stance": "disagree", "title": "..." } only. No wildcard card. No "description" on cards.
	- "comments" must be an array of exactly three strings: (1) backs agree, (2) backs disagree, (3) wildcard/hot-take still tied to this debate.
	- Headline frames an agree/disagree split; description adds match-specific stakes.

	Keep debates distinct: different angles, emotions, and stakes; no recycled headlines.

	Respond with ONLY a JSON array of exactly %d objects (no markdown, no code fences, no extra text).

	Each object must match this structure:
	{
	"headline": "Statement or question fans can agree or disagree with",
	"description": "Context that fuels the split",
	"cards": [
		{ "stance": "agree", "title": "Short YES-side label" },
		{ "stance": "disagree", "title": "Short NO-side label" }
	],
	"comments": ["fan comment pro-agree", "fan comment pro-disagree", "fan wildcard take"]
	}

	Return only the JSON array.`, phase, phaseNote, count)
}

// GenerateDebateSetPrompt performs one AI call and returns multiple debate prompts (e.g. 3) for the given type.
func (pg *PromptGenerator) GenerateDebateSetPrompt(ctx context.Context, matchData MatchData, promptType string, count int) ([]DebatePrompt, error) {
	if count <= 0 {
		count = DefaultDebateSetCount
	}
	if count > 7 {
		count = 7
	}
	systemPrompt := pg.buildSystemPromptForSet(promptType, count)
	userPrompt := pg.buildUserPrompt(matchData, promptType)

	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   maxTokensForDebateSet,
	}

	response, err := pg.callOpenAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	content := extractJSONArrayFromContent(response.Choices[0].Message.Content)
	var prompts []DebatePrompt
	if err := json.Unmarshal([]byte(content), &prompts); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response as array: %w", err)
	}
	for i := range prompts {
		NormalizeDebatePrompt(&prompts[i])
	}
	return prompts, nil
}

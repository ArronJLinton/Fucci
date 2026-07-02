package ai

import (
	"strings"
	"testing"
)

func TestBuildUserPrompt_IncludesH2HAndLeagueTable(t *testing.T) {
	pg := &PromptGenerator{}

	matchData := MatchData{
		MatchID:            "100",
		HomeTeam:           "Team A",
		AwayTeam:           "Team B",
		Date:               "2025-02-01T15:00:00Z",
		Status:             "NS",
		HeadToHeadSummary:  "2024-01-15: Team A 2-1 Team B\n2023-10-10: Team A 0-0 Team B",
		LeagueTableSummary: "1. Arsenal 50 pts\n2. Liverpool 48 pts",
	}

	prompt := pg.buildUserPrompt(matchData, "pre_match")

	if !strings.Contains(prompt, "HEAD-TO-HEAD:") {
		t.Error("prompt should contain HEAD-TO-HEAD section")
	}
	if !strings.Contains(prompt, "2024-01-15: Team A 2-1 Team B") {
		t.Error("prompt should contain head-to-head summary content")
	}
	if !strings.Contains(prompt, "LEAGUE TABLE:") {
		t.Error("prompt should contain LEAGUE TABLE section")
	}
	if !strings.Contains(prompt, "1. Arsenal 50 pts") {
		t.Error("prompt should contain league table summary content")
	}
}

func TestBuildUserPrompt_OmitsH2HAndLeagueTableWhenEmpty(t *testing.T) {
	pg := &PromptGenerator{}

	matchData := MatchData{
		MatchID:  "100",
		HomeTeam: "Team A",
		AwayTeam: "Team B",
		Date:     "2025-02-01T15:00:00Z",
		Status:   "NS",
		// HeadToHeadSummary and LeagueTableSummary left empty
	}

	prompt := pg.buildUserPrompt(matchData, "pre_match")

	if strings.Contains(prompt, "HEAD-TO-HEAD:") {
		t.Error("prompt should not contain HEAD-TO-HEAD when summary is empty")
	}
	if strings.Contains(prompt, "LEAGUE TABLE:") {
		t.Error("prompt should not contain LEAGUE TABLE when summary is empty")
	}
}

func TestExtractJSONFromContent_StripsMarkdownFences(t *testing.T) {
	jsonOnly := `{"headline":"Test","description":"Desc","cards":[]}`
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"raw json", jsonOnly, jsonOnly},
		{"with ```json fence", "```json\n" + jsonOnly + "\n```", jsonOnly},
		{"with ``` fence", "```\n" + jsonOnly + "\n```", jsonOnly},
		{"with leading whitespace", "  " + jsonOnly, jsonOnly},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONFromContent(tt.content)
			if got != tt.expected {
				t.Errorf("extractJSONFromContent() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildUserPrompt_IncludesRoundAndNewsGuidance(t *testing.T) {
	pg := &PromptGenerator{}
	matchData := MatchData{
		MatchID:       "100",
		HomeTeam:      "Brazil",
		AwayTeam:      "Japan",
		Date:          "2026-06-28T19:00:00Z",
		Status:        "NS",
		Round:         "Round of 32",
		NewsHeadlines: []string{"Endrick set to start for Brazil in knockout clash"},
	}
	prompt := pg.buildUserPrompt(matchData, "pre_match")
	if !strings.Contains(prompt, "Round/Stage: Round of 32") {
		t.Error("prompt should contain round/stage")
	}
	if !strings.Contains(prompt, "NEWS HEADLINES (use these") {
		t.Error("prompt should emphasize news headlines")
	}
	if !strings.Contains(prompt, "Endrick set to start") {
		t.Error("prompt should include news headline text")
	}
	if !strings.Contains(prompt, "Avoid generic 'who wins the showdown'") {
		t.Error("prompt should discourage generic showdown framing")
	}
}

func TestBuildSystemPrompt_IncludesRelevanceGuidelines(t *testing.T) {
	pg := &PromptGenerator{}
	pre := pg.buildSystemPrompt("pre_match")
	post := pg.buildSystemPrompt("post_match")
	for _, prompt := range []string{pre, post} {
		if !strings.Contains(prompt, "MATCH-SPECIFIC RELEVANCE") {
			t.Error("system prompt should include match-specific relevance guidelines")
		}
		if !strings.Contains(prompt, "NEWS HEADLINES") {
			t.Error("system prompt should instruct use of news headlines")
		}
		if !strings.Contains(prompt, "Classic giants or rising stars") {
			t.Error("system prompt should ban lazy generic framing examples")
		}
	}
	if !strings.Contains(pre, "knockout rounds") {
		t.Error("pre-match prompt should mention knockout round guidance")
	}
}

func TestBuildSystemPromptForSet_IncludesRelevanceGuidelines(t *testing.T) {
	pg := &PromptGenerator{}
	prompt := pg.buildSystemPromptForSet("pre_match", 3)
	if !strings.Contains(prompt, "MATCH-SPECIFIC RELEVANCE") {
		t.Error("set system prompt should include relevance guidelines")
	}
	if !strings.Contains(prompt, "at least one debate in the set must spring directly from a headline") {
		t.Error("set system prompt should require headline-driven debate in set")
	}
}

func TestBuildSystemPrompt_IncludesBinaryPropositionGuidelines(t *testing.T) {
	pg := &PromptGenerator{}
	for _, promptType := range []string{"pre_match", "post_match"} {
		prompt := pg.buildSystemPrompt(promptType)
		if !strings.Contains(prompt, "BINARY PROPOSITION") {
			t.Errorf("%s system prompt should include binary proposition guidelines", promptType)
		}
		if !strings.Contains(prompt, "should Portugal start Ronaldo") {
			t.Errorf("%s system prompt should include good headline example", promptType)
		}
		if !strings.Contains(prompt, "is it time to rely on younger talent") {
			t.Errorf("%s system prompt should include bad either/or example", promptType)
		}
	}
	setPrompt := pg.buildSystemPromptForSet("pre_match", 3)
	if !strings.Contains(setPrompt, "BINARY PROPOSITION") {
		t.Error("set system prompt should include binary proposition guidelines")
	}
}

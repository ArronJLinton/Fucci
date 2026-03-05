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
		MatchID:   "100",
		HomeTeam:  "Team A",
		AwayTeam:  "Team B",
		Date:      "2025-02-01T15:00:00Z",
		Status:    "NS",
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

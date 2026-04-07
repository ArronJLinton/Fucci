package futbol

import (
	"fmt"
	"strings"
)

const maxStandingsSummaryLines = 6

func FormatLeagueStandingsSummary(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	response, ok := raw["response"].([]any)
	if !ok {
		return ""
	}
	lines := []string{}
outer:
	for _, r := range response {
		block, ok := r.(map[string]any)
		if !ok {
			continue
		}
		league, _ := block["league"].(map[string]any)
		standings, _ := league["standings"].([]any)
		if len(standings) == 0 {
			continue
		}
		table, _ := standings[0].([]any)
		for _, row := range table {
			if len(lines) >= maxStandingsSummaryLines {
				break outer
			}
			entry, ok := row.(map[string]any)
			if !ok {
				continue
			}
			rank, _ := entry["rank"].(float64)
			points, _ := entry["points"].(float64)
			teamName := ""
			if team, ok := entry["team"].(map[string]any); ok {
				teamName, _ = team["name"].(string)
			}
			if teamName == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("%.0f. %s %.0f pts", rank, teamName, points))
		}
	}
	return strings.Join(lines, "\n")
}

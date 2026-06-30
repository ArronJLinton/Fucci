package database

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func TestNationalRankingsSeedUsesUniqueTeamIDs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	seedPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "sql", "schema", "20260630130100_seed_national_team_rankings.sql")
	contents, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("read seed migration: %v", err)
	}

	rowPattern := regexp.MustCompile(`\(\s*(\d+)\s*,\s*\d+\s*,\s*'[^']+'\s*,\s*'[^']+'\s*\)`)
	matches := rowPattern.FindAllStringSubmatch(string(contents), -1)
	if len(matches) == 0 {
		t.Fatal("expected at least one seeded national ranking row")
	}

	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		teamID := match[1]
		if _, exists := seen[teamID]; exists {
			t.Fatalf("duplicate team_id %s found in national rankings seed migration", teamID)
		}
		seen[teamID] = struct{}{}
	}
}

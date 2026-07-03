package api

import (
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

func TestValidateMatchStoryTeamLookup(t *testing.T) {
	match := &MatchInfo{HomeTeam: "Spain", AwayTeam: "Croatia"}
	homeKey := youtube.LookupKeyForTeamName("Spain")

	if err := validateMatchStoryTeamLookup("match", "123", homeKey, match); err != nil {
		t.Fatalf("expected valid home team key, got %v", err)
	}
	if err := validateMatchStoryTeamLookup("match", "123", "france", match); err == nil {
		t.Fatal("expected error for team not in match")
	}
	if err := validateMatchStoryTeamLookup("tournament", "123", homeKey, match); err == nil {
		t.Fatal("expected error for unsupported scope in v1")
	}
}

func TestCloudinaryConfigForMatchStoryContexts(t *testing.T) {
	photo, ok := cloudinaryConfigForContext("match_story_photo")
	if !ok || photo.ResourceType != "image" {
		t.Fatalf("unexpected photo config: %+v ok=%v", photo, ok)
	}
	video, ok := cloudinaryConfigForContext("match_story_video")
	if !ok || video.ResourceType != "video" || video.MaxBytes <= photo.MaxBytes {
		t.Fatalf("unexpected video config: %+v ok=%v", video, ok)
	}
}

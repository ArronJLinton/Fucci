package push

import (
	"testing"
	"time"
)

func TestScheduleMatchPushes(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 6, 30, 14, 0, 0, 0, time.UTC)
	short := &ShortCandidate{Title: "Highlight"}
	highlightsDelay := time.Hour
	stagger := 30 * time.Minute

	tests := []struct {
		name             string
		now              time.Time
		short            *ShortCandidate
		wantHighlights   bool
		wantDebates      bool
	}{
		{
			name:           "before highlights window",
			now:            end.Add(59 * time.Minute),
			short:          short,
			wantHighlights: false,
			wantDebates:    false,
		},
		{
			name:           "highlights due not debates",
			now:            end.Add(time.Hour),
			short:          short,
			wantHighlights: true,
			wantDebates:    false,
		},
		{
			name:           "both due after stagger",
			now:            end.Add(time.Hour + stagger),
			short:          short,
			wantHighlights: true,
			wantDebates:    true,
		},
		{
			name:           "debates only at 1h when no short",
			now:            end.Add(time.Hour),
			short:          nil,
			wantHighlights: false,
			wantDebates:    true,
		},
		{
			name:           "debates only waits 30m longer when short exists",
			now:            end.Add(time.Hour + 29*time.Minute),
			short:          short,
			wantHighlights: true,
			wantDebates:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scheduleMatchPushes(tc.now, end, tc.short, highlightsDelay, stagger)
			if got.HighlightsReady != tc.wantHighlights {
				t.Fatalf("HighlightsReady=%v want %v", got.HighlightsReady, tc.wantHighlights)
			}
			if got.DebatesReady != tc.wantDebates {
				t.Fatalf("DebatesReady=%v want %v", got.DebatesReady, tc.wantDebates)
			}
		})
	}
}

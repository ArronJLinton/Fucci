package push

import "time"

// matchPushSchedule decides which post-FT pushes are due for a fixture at now.
type matchPushSchedule struct {
	HighlightsReady bool
	DebatesReady    bool
}

// scheduleMatchPushes returns which match pushes should be attempted.
// Highlights fire at estimatedEnd + highlightsDelay when a Short exists.
// Debates fire at estimatedEnd + highlightsDelay + stagger when a Short exists,
// otherwise at estimatedEnd + highlightsDelay (debates-only fallback).
func scheduleMatchPushes(
	now, estimatedEnd time.Time,
	short *ShortCandidate,
	highlightsDelay, debatesStagger time.Duration,
) matchPushSchedule {
	if highlightsDelay <= 0 {
		highlightsDelay = DefaultDelayAfterFT
	}
	if debatesStagger <= 0 {
		debatesStagger = DefaultDebatesStaggerAfterHighlights
	}

	debatesDelay := highlightsDelay
	if short != nil {
		debatesDelay = highlightsDelay + debatesStagger
	}

	return matchPushSchedule{
		HighlightsReady: short != nil && !now.Before(estimatedEnd.Add(highlightsDelay)),
		DebatesReady:    !now.Before(estimatedEnd.Add(debatesDelay)),
	}
}

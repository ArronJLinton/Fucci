package youtube

import (
	"regexp"
	"strconv"
)

var iso8601Duration = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// ParseISO8601DurationSeconds parses YouTube ISO 8601 durations (e.g. PT58S, PT1M30S).
func ParseISO8601DurationSeconds(iso string) int {
	m := iso8601Duration.FindStringSubmatch(iso)
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	sec, _ := strconv.Atoi(m[3])
	return h*3600 + min*60 + sec
}

// IsShortDuration returns true when duration is at most maxSeconds (YouTube Shorts cap).
func IsShortDuration(iso string, maxSeconds int) bool {
	sec := ParseISO8601DurationSeconds(iso)
	return sec > 0 && sec <= maxSeconds
}

package push

import (
	"fmt"
	"time"
)

// LocalTimeInTimezone returns the current wall-clock time in the given IANA timezone.
func LocalTimeInTimezone(tz string) (time.Time, error) {
	return LocalTimeInTimezoneAt(tz, time.Now())
}

// LocalTimeInTimezoneAt returns wall-clock time in tz at the given instant.
func LocalTimeInTimezoneAt(tz string, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return now.In(loc), nil
}

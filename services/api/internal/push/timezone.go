package push

import (
	"fmt"
	"time"
)

// LocalTimeInTimezone returns the current wall-clock time in the given IANA timezone.
func LocalTimeInTimezone(tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return time.Now().In(loc), nil
}

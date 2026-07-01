package push

import (
	"fmt"
	"time"
)

const (
	// DefaultDeliveryWindow is how far local clock may be from the target send time (±).
	DefaultDeliveryWindow = 7 * time.Minute
	// ScanInterval matches the 15-minute slot scanner tick.
	ScanInterval = 15 * time.Minute
	// ScanLockTTL prevents duplicate campaign scans across Fly instances.
	ScanLockTTL = 20 * time.Minute
)

// ScanSlot is a UTC 15-minute window identifier for distributed scan locks.
type ScanSlot string

// CurrentScanSlot returns the UTC slot key for now.
func CurrentScanSlot(now time.Time) ScanSlot {
	return ScanSlot(now.UTC().Truncate(ScanInterval).Format("2006-01-02T15:04"))
}

// ScanLockKey builds the Redis lock key for one campaign in one scan slot.
func ScanLockKey(campaignKey string, slot ScanSlot) string {
	return fmt.Sprintf("push:scan:%s:%s", campaignKey, slot)
}

// InDeliveryWindow reports whether localNow is within ±window of today's target time.
func InDeliveryWindow(localNow time.Time, target TargetLocalTime, window time.Duration) bool {
	if window <= 0 {
		window = DefaultDeliveryWindow
	}
	sendAt := time.Date(
		localNow.Year(), localNow.Month(), localNow.Day(),
		target.Hour, target.Minute, 0, 0, localNow.Location(),
	)
	diff := localNow.Sub(sendAt)
	if diff < 0 {
		diff = -diff
	}
	return diff <= window
}

// InDeliveryWindowForTimezone evaluates the delivery window for an IANA timezone.
func InDeliveryWindowForTimezone(tz string, target TargetLocalTime, window time.Duration, now time.Time) (bool, error) {
	if tz == "" {
		tz = "UTC"
	}
	local, err := LocalTimeInTimezoneAt(tz, now)
	if err != nil {
		return false, err
	}
	return InDeliveryWindow(local, target, window), nil
}

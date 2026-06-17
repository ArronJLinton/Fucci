package scheduler

import (
	"testing"
	"time"
)

func TestNextOccurrenceUTC(t *testing.T) {
	timeOfDay := time.Date(0, 1, 1, 4, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before today's slot returns today's slot",
			now:  time.Date(2026, 6, 15, 3, 30, 0, 0, time.UTC),
			want: time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "exactly at slot returns next day",
			now:  time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC),
			want: time.Date(2026, 6, 16, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "after slot returns next day",
			now:  time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC),
			want: time.Date(2026, 6, 16, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "month rollover preserved",
			now:  time.Date(2026, 6, 30, 23, 59, 0, 0, time.UTC),
			want: time.Date(2026, 7, 1, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "non-UTC input is normalized to UTC before comparison",
			now:  time.Date(2026, 6, 15, 22, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			// 22:00 EST == 03:00 UTC next day, which is before 04:00 UTC, so today's UTC slot.
			want: time.Date(2026, 6, 16, 4, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextOccurrenceUTC(tc.now, timeOfDay)
			if !got.Equal(tc.want) {
				t.Fatalf("nextOccurrenceUTC(%s, 04:00:00) = %s; want %s",
					tc.now.Format(time.RFC3339), got.Format(time.RFC3339), tc.want.Format(time.RFC3339))
			}
		})
	}
}

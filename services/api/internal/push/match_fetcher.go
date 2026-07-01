package push

import (
	"context"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

// MatchFetcherFunc adapts a function to MatchFetcher.
type MatchFetcherFunc func(ctx context.Context, date time.Time, leagueID int) ([]MatchFixture, error)

func (f MatchFetcherFunc) FetchLeagueMatches(ctx context.Context, date time.Time, leagueID int) ([]MatchFixture, error) {
	return f(ctx, date, leagueID)
}

// MediaShortsFunc adapts a function to MediaShortsProvider.
type MediaShortsFunc func(ctx context.Context) []youtube.MediaOutletShorts

func (f MediaShortsFunc) MediaOutletsShorts(ctx context.Context) []youtube.MediaOutletShorts {
	return f(ctx)
}

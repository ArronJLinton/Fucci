package push

import (
	"context"
	"fmt"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/news"
)

// NewsOpenStore tracks articles a user has opened (for exclusion).
type NewsOpenStore interface {
	ListOpenedArticleURLsForUser(ctx context.Context, userID int32) ([]string, error)
}

// NewsFeed supplies ranked articles for the daily news push.
type NewsFeed interface {
	RankedArticles(ctx context.Context) ([]news.NewsArticle, error)
}

// NewsDailySelector picks the best unopened article for a user at 12pm local.
type NewsDailySelector struct {
	Feed  NewsFeed
	Opens NewsOpenStore
}

func (s *NewsDailySelector) Select(ctx context.Context, user UserCandidate) (SelectResult, error) {
	if s.Feed == nil {
		return SelectResult{Skip: "news_feed_unconfigured"}, nil
	}
	articles, err := s.Feed.RankedArticles(ctx)
	if err != nil {
		return SelectResult{}, fmt.Errorf("ranked articles: %w", err)
	}
	if len(articles) == 0 {
		return SelectResult{Skip: "no_articles"}, nil
	}

	var opened map[string]struct{}
	if s.Opens != nil {
		urls, err := s.Opens.ListOpenedArticleURLsForUser(ctx, user.UserID)
		if err != nil {
			return SelectResult{}, fmt.Errorf("list opened articles: %w", err)
		}
		opened = news.OpenedURLSet(urls)
	}
	candidates := news.FilterUnopened(articles, opened)
	if len(candidates) == 0 {
		return SelectResult{Skip: "no_unopened_articles"}, nil
	}

	article := candidates[0]
	body := strings.TrimSpace(article.Snippet)
	if body == "" {
		body = article.SourceName
	}
	return SelectResult{
		Request: &SendRequest{
			Title: article.Title,
			Body:  body,
			Data: newsPushData(article.SourceURL),
		},
	}, nil
}

func newsPushData(sourceURL string) map[string]interface{} {
	return map[string]interface{}{
		"type":   "news",
		"route":  "NewsWebView",
		"params": map[string]interface{}{"url": sourceURL},
	}
}

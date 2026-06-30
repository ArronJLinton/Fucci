package push

import (
	"context"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/news"
)

type stubNewsFeed struct {
	articles []news.NewsArticle
	err      error
}

func (s stubNewsFeed) RankedArticles(_ context.Context) ([]news.NewsArticle, error) {
	return s.articles, s.err
}

type stubNewsOpens struct {
	urls []string
}

func (s stubNewsOpens) ListOpenedArticleURLsForUser(_ context.Context, _ int32) ([]string, error) {
	return s.urls, nil
}

func TestNewsDailySelector_PicksBestUnopened(t *testing.T) {
	t.Parallel()
	sel := &NewsDailySelector{
		Feed: stubNewsFeed{articles: []news.NewsArticle{
			{ID: "1", Title: "Opened story", SourceURL: "https://a.com/1", Snippet: "s1"},
			{ID: "2", Title: "Fresh story", SourceURL: "https://a.com/2", Snippet: "s2", SourceName: "ESPN"},
		}},
		Opens: stubNewsOpens{urls: []string{"https://a.com/1"}},
	}
	result, err := sel.Select(context.Background(), UserCandidate{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Request == nil {
		t.Fatalf("expected request, skip=%q", result.Skip)
	}
	if result.Request.Title != "Fresh story" {
		t.Fatalf("got title %q", result.Request.Title)
	}
	if result.Request.Data["type"] != "news" {
		t.Fatalf("unexpected data type: %v", result.Request.Data["type"])
	}
}

func TestNewsDailySelector_SkipsWhenAllOpened(t *testing.T) {
	t.Parallel()
	sel := &NewsDailySelector{
		Feed: stubNewsFeed{articles: []news.NewsArticle{
			{ID: "1", Title: "Only", SourceURL: "https://a.com/1"},
		}},
		Opens: stubNewsOpens{urls: []string{"https://a.com/1"}},
	}
	result, err := sel.Select(context.Background(), UserCandidate{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Skip != "no_unopened_articles" {
		t.Fatalf("expected skip, got %+v", result)
	}
}

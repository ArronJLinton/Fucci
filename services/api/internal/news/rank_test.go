package news

import "testing"

func TestRankArticlesForPush(t *testing.T) {
	t.Parallel()
	today := []NewsArticle{
		{ID: "1", Title: "Premier league recap", SourceURL: "https://a.com/1", PublishedAt: "2026-06-30T10:00:00Z"},
		{ID: "2", Title: "World Cup final preview", SourceURL: "https://a.com/2", PublishedAt: "2026-06-30T09:00:00Z"},
	}
	history := []NewsArticle{
		{ID: "3", Title: "Older FIFA story", SourceURL: "https://a.com/3", PublishedAt: "2026-06-29T12:00:00Z"},
	}
	ranked := RankArticlesForPush(today, history)
	if len(ranked) != 3 {
		t.Fatalf("expected 3 articles, got %d", len(ranked))
	}
	if ranked[0].ID != "2" {
		t.Fatalf("expected WC today article first, got %q", ranked[0].ID)
	}
}

func TestFilterUnopened(t *testing.T) {
	t.Parallel()
	articles := []NewsArticle{
		{ID: "1", SourceURL: "https://a.com/1"},
		{ID: "2", SourceURL: "https://a.com/2"},
	}
	opened := OpenedURLSet([]string{"https://a.com/1"})
	filtered := FilterUnopened(articles, opened)
	if len(filtered) != 1 || filtered[0].ID != "2" {
		t.Fatalf("unexpected filtered: %+v", filtered)
	}
}

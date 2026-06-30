package news

import (
	"sort"
	"strings"
	"time"
)

var worldCupKeywords = []string{
	"world cup", "fifa", "group stage", "round of 16", "quarterfinal",
	"quarter-final", "quarter final", "semifinal", "semi-final", "semi final",
	" final ", " usa ", " canada ", " mexico ", "concacaf", "fan zone",
}

type rankedArticle struct {
	article   NewsArticle
	fromToday bool
}

// RankArticlesForPush merges today and history feeds, dedupes by id, and ranks for push.
// Priority: World Cup keyword match > today pool > recency.
func RankArticlesForPush(today, history []NewsArticle) []NewsArticle {
	byID := map[string]rankedArticle{}
	for _, a := range today {
		if a.ID == "" || a.SourceURL == "" {
			continue
		}
		byID[a.ID] = rankedArticle{article: a, fromToday: true}
	}
	for _, a := range history {
		if a.ID == "" || a.SourceURL == "" {
			continue
		}
		if _, ok := byID[a.ID]; !ok {
			byID[a.ID] = rankedArticle{article: a, fromToday: false}
		}
	}

	items := make([]rankedArticle, 0, len(byID))
	for _, item := range byID {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return compareRankedArticles(items[i], items[j]) > 0
	})

	out := make([]NewsArticle, len(items))
	for i, item := range items {
		out[i] = item.article
	}
	return out
}

func compareRankedArticles(a, b rankedArticle) int {
	if wc := boolRank(worldCupKeywordMatch(a.article)) - boolRank(worldCupKeywordMatch(b.article)); wc != 0 {
		return wc
	}
	if td := boolRank(a.fromToday) - boolRank(b.fromToday); td != 0 {
		return td
	}
	return int(publishedAtUnix(a.article) - publishedAtUnix(b.article))
}

func boolRank(v bool) int {
	if v {
		return 1
	}
	return 0
}

func worldCupKeywordMatch(a NewsArticle) bool {
	haystack := " " + strings.ToLower(a.Title+" "+a.Snippet) + " "
	for _, k := range worldCupKeywords {
		if strings.Contains(haystack, k) {
			return true
		}
	}
	return false
}

func publishedAtUnix(a NewsArticle) int64 {
	if a.PublishedAt == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, a.PublishedAt)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", a.PublishedAt)
		if err != nil {
			return 0
		}
	}
	return t.Unix()
}

// FilterUnopened returns articles whose sourceUrl is not in openedURLs.
func FilterUnopened(articles []NewsArticle, openedURLs map[string]struct{}) []NewsArticle {
	if len(openedURLs) == 0 {
		return articles
	}
	out := make([]NewsArticle, 0, len(articles))
	for _, a := range articles {
		if _, opened := openedURLs[a.SourceURL]; !opened {
			out = append(out, a)
		}
	}
	return out
}

// OpenedURLSet converts a URL slice to a lookup set.
func OpenedURLSet(urls []string) map[string]struct{} {
	set := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		if u != "" {
			set[u] = struct{}{}
		}
	}
	return set
}

package news

import (
	"fmt"
	"log"
	"time"
)

// NewsArticle represents the transformed article for the API response
type NewsArticle struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet,omitempty"`
	ImageURL     string `json:"imageUrl,omitempty"`
	SourceURL    string `json:"sourceUrl"`
	SourceName   string `json:"sourceName"`
	PublishedAt  string `json:"publishedAt"`  // ISO 8601 format
	RelativeTime string `json:"relativeTime"` // "2 hours ago", "1 day ago"
}

// NewsAPIResponse represents the API response structure
type NewsAPIResponse struct {
	TodayArticles   []NewsArticle `json:"todayArticles"`
	HistoryArticles []NewsArticle `json:"historyArticles"`
	Cached          bool          `json:"cached"`
	CachedAt        string        `json:"cachedAt,omitempty"` // ISO 8601 format
}

// TransformRapidAPIResponse transforms RapidAPI response to internal format
func TransformRapidAPIResponse(rapidAPIResp *RapidAPIResponse) (*NewsAPIResponse, error) {
	articles := make([]NewsArticle, 0, len(rapidAPIResp.Data))

	for _, article := range rapidAPIResp.Data {
		articleID, err := GenerateArticleID(article.Link)
		if err != nil {
			log.Printf("news: skipping article, GenerateArticleID failed for link %q: %v", article.Link, err)
			continue
		}

		relativeTime := computeRelativeTime(article.PublishedDatetimeUTC)

		transformedArticle := NewsArticle{
			ID:           articleID,
			Title:        article.Title,
			Snippet:      article.Snippet,
			ImageURL:     article.PhotoURL,
			SourceURL:    article.Link,
			SourceName:   article.SourceName,
			PublishedAt:  article.PublishedDatetimeUTC,
			RelativeTime: relativeTime,
		}

		articles = append(articles, transformedArticle)
	}

	return &NewsAPIResponse{
		TodayArticles: articles,
		// HistoryArticles: []NewsArticle{l},
		Cached: false,
	}, nil
}

// TransformTodayAndHistoryResponse transforms both today's and historical news responses
func TransformTodayAndHistoryResponse(resp *TodayAndHistoryResponse) (*NewsAPIResponse, error) {
	// Transform today's articles
	todayArticles := make([]NewsArticle, 0)
	if resp.TodayResponse != nil {
		for _, article := range resp.TodayResponse.Data {
			articleID, err := GenerateArticleID(article.Link)
			if err != nil {
				log.Printf("news: skipping today article, GenerateArticleID failed for link %q: %v", article.Link, err)
				continue
			}

			relativeTime := computeRelativeTime(article.PublishedDatetimeUTC)

			todayArticles = append(todayArticles, NewsArticle{
				ID:           articleID,
				Title:        article.Title,
				Snippet:      article.Snippet,
				ImageURL:     article.PhotoURL,
				SourceURL:    article.Link,
				SourceName:   article.SourceName,
				PublishedAt:  article.PublishedDatetimeUTC,
				RelativeTime: relativeTime,
			})
		}
	}

	// Transform historical articles
	historyArticles := make([]NewsArticle, 0)
	if resp.HistoryResponse != nil {
		for _, article := range resp.HistoryResponse.Data {
			articleID, err := GenerateArticleID(article.Link)
			if err != nil {
				log.Printf("news: skipping history article, GenerateArticleID failed for link %q: %v", article.Link, err)
				continue
			}

			relativeTime := computeRelativeTime(article.PublishedDatetimeUTC)

			historyArticles = append(historyArticles, NewsArticle{
				ID:           articleID,
				Title:        article.Title,
				Snippet:      article.Snippet,
				ImageURL:     article.PhotoURL,
				SourceURL:    article.Link,
				SourceName:   article.SourceName,
				PublishedAt:  article.PublishedDatetimeUTC,
				RelativeTime: relativeTime,
			})
		}
	}

	return &NewsAPIResponse{
		TodayArticles:   todayArticles,
		HistoryArticles: historyArticles,
		Cached:          false,
	}, nil
}

// computeRelativeTime computes a human-readable relative time string
// from an ISO 8601 timestamp
func computeRelativeTime(isoTimestamp string) string {
	publishedTime, err := time.Parse(time.RFC3339, isoTimestamp)
	if err != nil {
		// If parsing fails, try alternative formats
		publishedTime, err = time.Parse("2006-01-02T15:04:05Z", isoTimestamp)
		if err != nil {
			return "Unknown"
		}
	}

	now := time.Now()
	diff := now.Sub(publishedTime)

	// Less than 1 minute
	if diff < time.Minute {
		return "Just now"
	}

	// Less than 1 hour
	if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	}

	// Less than 1 day
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}

	// Less than 1 week
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}

	// Less than 1 month (approximately 30 days)
	if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (7 * 24))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}

	// Less than 1 year
	if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (30 * 24))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	// 1 year or more
	years := int(diff.Hours() / (365 * 24))
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// MatchNewsAPIResponse represents the API response structure for match-specific news
type MatchNewsAPIResponse struct {
	Articles []NewsArticle `json:"articles"`
	Cached   bool          `json:"cached"`
	CachedAt string        `json:"cachedAt,omitempty"` // ISO 8601 format
}

// TransformMatchNewsResponse transforms RapidAPI response to match news format (today's articles only)
func TransformMatchNewsResponse(rapidAPIResp *RapidAPIResponse) (*MatchNewsAPIResponse, error) {
	articles := make([]NewsArticle, 0, len(rapidAPIResp.Data))

	for _, article := range rapidAPIResp.Data {
		articleID, err := GenerateArticleID(article.Link)
		if err != nil {
			log.Printf("news: skipping match article, GenerateArticleID failed for link %q: %v", article.Link, err)
			continue
		}

		relativeTime := computeRelativeTime(article.PublishedDatetimeUTC)

		transformedArticle := NewsArticle{
			ID:           articleID,
			Title:        article.Title,
			Snippet:      article.Snippet,
			ImageURL:     article.PhotoURL,
			SourceURL:    article.Link,
			SourceName:   article.SourceName,
			PublishedAt:  article.PublishedDatetimeUTC,
			RelativeTime: relativeTime,
		}

		articles = append(articles, transformedArticle)
	}

	return &MatchNewsAPIResponse{
		Articles: articles,
		Cached:   false,
	}, nil
}

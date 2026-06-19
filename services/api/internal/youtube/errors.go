package youtube

import (
	"fmt"
	"strings"
)

// FetchError represents a non-200 YouTube API response.
type FetchError struct {
	StatusCode int
	Body       string
}

func (e *FetchError) Error() string {
	if e == nil {
		return "youtube fetch error"
	}
	if e.Body != "" {
		return fmt.Sprintf("youtube api status %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("youtube api status %d", e.StatusCode)
}

// IsQuotaExceeded reports whether the error likely indicates quota exhaustion.
func IsQuotaExceeded(err error) bool {
	fe, ok := err.(*FetchError)
	if !ok || fe == nil {
		return false
	}
	return fe.StatusCode == 403 && (strings.Contains(fe.Body, "quota") || strings.Contains(fe.Body, "dailyLimitExceeded"))
}

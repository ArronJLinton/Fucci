package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/moderation"
	"github.com/google/uuid"
)

const accountDeactivatedMessage = "This account has been deactivated. Contact support if you believe this is a mistake."

const (
	moderationReportDescMaxLen = 500
	moderationRateLimitN       = 10
	moderationRateWindow       = time.Minute
	moderationNotifyMaxInFlight = 32
)

var (
	contentFilterOnce sync.Once
	contentFilter     *moderation.Filter

	moderationNotifySem = make(chan struct{}, moderationNotifyMaxInFlight)

	defaultModerationRateLimiter = moderationRateLimiter{byUser: make(map[int32]moderationRateEntry)}
)

type moderationRateEntry struct {
	count       int
	windowStart time.Time
}

type moderationRateLimiter struct {
	mu     sync.Mutex
	byUser map[int32]moderationRateEntry
}

func (r *moderationRateLimiter) allow(userID int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byUser == nil {
		r.byUser = make(map[int32]moderationRateEntry)
	}
	now := time.Now()
	entry, ok := r.byUser[userID]
	if !ok || now.Sub(entry.windowStart) >= moderationRateWindow {
		r.byUser[userID] = moderationRateEntry{count: 1, windowStart: now}
		return true
	}
	entry.count++
	r.byUser[userID] = entry
	return entry.count <= moderationRateLimitN
}

func checkModerationRateLimit(ctx context.Context, c *Config, userID int32) bool {
	if c != nil && c.Cache != nil {
		key := fmt.Sprintf("ratelimit:moderation:%d", userID)
		n, err := c.Cache.Incr(ctx, key)
		if err == nil {
			if n == 1 {
				if err := c.Cache.Expire(ctx, key, moderationRateWindow); err != nil {
					log.Printf("[moderation] rate limit Redis Expire failed: %v", err)
				}
			} else {
				ttl, ttlErr := c.Cache.TTL(ctx, key)
				if ttlErr != nil || ttl < 0 {
					if err := c.Cache.Expire(ctx, key, moderationRateWindow); err != nil {
						log.Printf("[moderation] rate limit Redis fallback Expire failed: %v", err)
					}
				}
			}
			return n <= int64(moderationRateLimitN)
		}
		log.Printf("[moderation] rate limit Redis Incr failed: %v; using in-memory fallback", err)
	}
	return defaultModerationRateLimiter.allow(userID)
}

func clampModerationDescription(raw string) (string, error) {
	d := strings.TrimSpace(raw)
	if d == "" {
		return "", nil
	}
	if len(d) > moderationReportDescMaxLen {
		return "", fmt.Errorf("description must be at most %d characters", moderationReportDescMaxLen)
	}
	return d, nil
}

func (c *Config) enqueueModerationNotify(p moderation.ReportEmailPayload) {
	go func() {
		select {
		case moderationNotifySem <- struct{}{}:
			defer func() { <-moderationNotifySem }()
			c.moderationNotifier().NotifyReport(p)
		default:
			// Bound concurrency: fall back to log-only so SMTP work cannot unbounded-grow.
			log.Printf("MODERATION_ALERT queue full; logging only report_id=%s type=%s", p.ReportID, p.ReportableType)
			(&moderation.Notifier{
				Mail: moderation.MailConfig{To: c.ModerationNotifyEmail},
			}).NotifyReport(p)
		}
	}()
}

func (c *Config) objectionableFilter() *moderation.Filter {
	contentFilterOnce.Do(func() {
		contentFilter = moderation.DefaultFilter()
	})
	return contentFilter
}

func (c *Config) rejectIfObjectionable(w http.ResponseWriter, text string) bool {
	if err := c.objectionableFilter().Check(text); err != nil {
		respondWithErrorCode(w, http.StatusBadRequest, err.Error(), "OBJECTIONABLE_CONTENT")
		return true
	}
	return false
}

func (c *Config) moderationNotifier() *moderation.Notifier {
	if c.ModerationNotifier != nil {
		return c.ModerationNotifier
	}
	to := c.ModerationNotifyEmail
	if to == "" {
		to = "contact@magistri.dev"
	}
	return &moderation.Notifier{
		Mail: moderation.MailConfig{
			Host:     c.SMTPHost,
			Port:     c.SMTPPort,
			Username: c.SMTPUsername,
			Password: c.SMTPPassword,
			From:     c.SMTPFrom,
			To:       to,
		},
	}
}

func (c *Config) contentChecker() moderation.ContentChecker {
	if c == nil {
		return nil
	}
	return c.ContentChecker
}

// softFlagUGC runs a synchronous AI moderation check after content is published.
// If flagged, it creates a pending content_report (system reporter) and emails moderation.
// Failures of the AI service are logged and ignored so publishing is not blocked.
func (c *Config) softFlagUGC(
	ctx context.Context,
	authorID int32,
	reportableType, reportableID string,
	texts []string,
	mediaURL, mediaKind string,
) {
	checker := c.contentChecker()
	if checker == nil || c.DB == nil {
		return
	}

	input := moderation.ContentCheckInput{Texts: texts}
	if preview := moderation.MediaPreviewURL(mediaURL, mediaKind); preview != "" {
		input.ImageURLs = []string{preview}
	}

	result, err := checker.Check(ctx, input)
	if err != nil {
		log.Printf("[moderation] AI check failed type=%s id=%s: %v", reportableType, reportableID, err)
		return
	}
	if !result.Flagged {
		return
	}

	systemID, err := c.getSystemUserID(ctx)
	if err != nil || systemID == 0 {
		log.Printf("[moderation] AI soft-flag skipped; system user unavailable: %v", err)
		return
	}

	desc := moderation.FormatAIFlagDescription(result)
	report, err := c.DB.CreateContentReport(ctx, database.CreateContentReportParams{
		ReporterID:     systemID,
		ReportableType: reportableType,
		ReportableID:   reportableID,
		ReportedUserID: sql.NullInt32{Int32: authorID, Valid: authorID != 0},
		Reason:         "inappropriate_content",
		Description:    sql.NullString{String: desc, Valid: true},
	})
	if err != nil {
		log.Printf("[moderation] AI soft-flag report create failed type=%s id=%s: %v", reportableType, reportableID, err)
		return
	}

	reported := authorID
	var reportedPtr *int32
	if authorID != 0 {
		reportedPtr = &reported
	}
	c.enqueueModerationNotify(moderation.ReportEmailPayload{
		ReportID:       report.ID.String(),
		ReporterID:     systemID,
		ReportedUserID: reportedPtr,
		ReportableType: reportableType,
		ReportableID:   reportableID,
		Reason:         "inappropriate_content",
		Description:    desc,
		Source:         "ai_soft_flag",
	})
	log.Printf("[moderation] AI soft-flag queued report_id=%s type=%s id=%s cats=%v",
		report.ID.String(), reportableType, reportableID, result.Categories)
}

func (c *Config) recordTermsAcceptance(ctx context.Context, userID int32) {
	if userID == 0 || c.DBConn == nil {
		// Production always sets DBConn. Unit tests often wire only sqlc DB mocks.
		return
	}
	_, err := c.DBConn.ExecContext(ctx,
		`UPDATE users SET terms_accepted_at = CURRENT_TIMESTAMP, terms_version = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		moderation.CurrentTermsVersion, userID,
	)
	if err != nil {
		log.Printf("record terms acceptance user=%d: %v", userID, err)
	}
}

func requireAcceptedTerms(accepted bool) error {
	if !accepted {
		return errors.New("you must accept the Terms of Use to continue")
	}
	return nil
}

func (c *Config) blockedUserIDSet(ctx context.Context, blockerID int32) map[int32]struct{} {
	out := map[int32]struct{}{}
	if c.DB == nil || blockerID == 0 {
		return out
	}
	ids, err := c.DB.ListBlockedUserIDs(ctx, blockerID)
	if err != nil {
		return out
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func filterStoriesByBlocked(stories []userStoryPayload, blocked map[int32]struct{}) []userStoryPayload {
	if len(blocked) == 0 || len(stories) == 0 {
		return stories
	}
	out := make([]userStoryPayload, 0, len(stories))
	for _, s := range stories {
		if _, ok := blocked[s.UserID]; ok {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (c *Config) ensureUserActive(ctx context.Context, userID int32) error {
	if c.DB == nil || userID == 0 {
		return nil
	}
	active, err := c.DB.IsUserActive(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}
	if !active {
		return errAccountDeactivated
	}
	return nil
}

var errAccountDeactivated = errors.New(accountDeactivatedMessage)

func respondAccountDeactivated(w http.ResponseWriter) {
	respondWithErrorCode(w, http.StatusForbidden, accountDeactivatedMessage, auth.GoogleAuthAccountInactive)
}

func (c *Config) requireActiveAuthedUser(w http.ResponseWriter, r *http.Request) (int32, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return 0, false
	}
	if err := c.ensureUserActive(r.Context(), userID); err != nil {
		if errors.Is(err, errAccountDeactivated) {
			respondAccountDeactivated(w)
			return 0, false
		}
		respondWithError(w, http.StatusInternalServerError, "failed to verify account status")
		return 0, false
	}
	return userID, true
}

// resolveReportableTarget validates reportable_type/id the same way as POST /reports.
// Returns normalized id, reported user, or an http-ready error message + status.
type resolveReportableResult struct {
	ReportableID   string
	ReportedUserID sql.NullInt32
	AlreadyRemoved bool
}

func (c *Config) resolveReportableTarget(
	ctx context.Context,
	reporterID int32,
	reportableType, rawID string,
) (resolveReportableResult, int, string) {
	switch reportableType {
	case "story":
		storyID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return resolveReportableResult{}, http.StatusBadRequest, "invalid reportable_id"
		}
		story, err := c.DB.GetMatchStoryByID(ctx, storyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return resolveReportableResult{}, http.StatusNotFound, "story not found"
			}
			return resolveReportableResult{}, http.StatusInternalServerError, "Failed to load story"
		}
		if !story.IsActive {
			return resolveReportableResult{AlreadyRemoved: true}, http.StatusOK, ""
		}
		if story.UserID == reporterID {
			return resolveReportableResult{}, http.StatusBadRequest, "cannot report your own content"
		}
		return resolveReportableResult{
			ReportableID:   storyID.String(),
			ReportedUserID: sql.NullInt32{Int32: story.UserID, Valid: true},
		}, 0, ""

	case "debate_response":
		commentID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 32)
		if err != nil || commentID <= 0 {
			return resolveReportableResult{}, http.StatusBadRequest, "invalid reportable_id"
		}
		comment, err := c.DB.GetComment(ctx, int32(commentID))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return resolveReportableResult{}, http.StatusNotFound, "comment not found"
			}
			return resolveReportableResult{}, http.StatusInternalServerError, "Failed to load comment"
		}
		if !comment.UserID.Valid {
			return resolveReportableResult{}, http.StatusBadRequest, "comment has no author"
		}
		if comment.UserID.Int32 == reporterID {
			return resolveReportableResult{}, http.StatusBadRequest, "cannot report your own content"
		}
		return resolveReportableResult{
			ReportableID:   strconv.FormatInt(commentID, 10),
			ReportedUserID: comment.UserID,
		}, 0, ""

	case "avatar", "player_profile", "user":
		targetUserID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 32)
		if err != nil || targetUserID <= 0 {
			return resolveReportableResult{}, http.StatusBadRequest, "invalid reportable_id"
		}
		if int32(targetUserID) == reporterID {
			return resolveReportableResult{}, http.StatusBadRequest, "cannot report your own content"
		}
		if _, err := c.DB.GetUser(ctx, int32(targetUserID)); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return resolveReportableResult{}, http.StatusNotFound, "user not found"
			}
			return resolveReportableResult{}, http.StatusInternalServerError, "Failed to load user"
		}
		return resolveReportableResult{
			ReportableID:   strconv.FormatInt(targetUserID, 10),
			ReportedUserID: sql.NullInt32{Int32: int32(targetUserID), Valid: true},
		}, 0, ""
	default:
		return resolveReportableResult{}, http.StatusBadRequest, "reportable_type must be story, debate_response, avatar, player_profile, or user"
	}
}

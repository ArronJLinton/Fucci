package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
	"github.com/google/uuid"
)

const matchStoriesListLimit = 50

type createMatchStoryRequest struct {
	ScopeType     string  `json:"scope_type"`
	ScopeID       string  `json:"scope_id"`
	TeamLookupKey string  `json:"team_lookup_key"`
	ContentType   string  `json:"content_type"`
	MediaURL      string  `json:"media_url"`
	Caption       *string `json:"caption"`
}

type matchStoryResponse struct {
	ID            string  `json:"id"`
	UserID        int32   `json:"user_id"`
	ScopeType     string  `json:"scope_type"`
	ScopeID       string  `json:"scope_id"`
	TeamLookupKey string  `json:"team_lookup_key"`
	ContentType   string  `json:"content_type"`
	MediaURL      string  `json:"media_url"`
	Caption       *string `json:"caption,omitempty"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
	DisplayName   *string `json:"display_name,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
}

type userStoryPayload struct {
	ID          string  `json:"id"`
	ContentType string  `json:"content_type"`
	MediaURL    string  `json:"media_url"`
	UserID      int32   `json:"user_id"`
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid || strings.TrimSpace(ns.String) == "" {
		return nil
	}
	s := ns.String
	return &s
}

func matchStoryFromRow(row database.ListActiveMatchStoriesForTeamRow) userStoryPayload {
	return userStoryPayload{
		ID:          row.ID.String(),
		ContentType: string(row.ContentType),
		MediaURL:    row.MediaUrl,
		UserID:      row.UserID,
		DisplayName: nullStringPtr(row.UserDisplayName),
		AvatarURL:   nullStringPtr(row.UserAvatarUrl),
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func matchStoryFromRecord(row database.MatchStories) matchStoryResponse {
	return matchStoryResponse{
		ID:            row.ID.String(),
		UserID:        row.UserID,
		ScopeType:     string(row.ScopeType),
		ScopeID:       row.ScopeID,
		TeamLookupKey: row.TeamLookupKey,
		ContentType:   string(row.ContentType),
		MediaURL:      row.MediaUrl,
		Caption:       nullStringPtr(row.Caption),
		IsActive:      row.IsActive,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (c *Config) listUserStoriesForTeam(ctx context.Context, scopeType database.StoryScopeType, scopeID, teamLookupKey string) []userStoryPayload {
	if c.DB == nil {
		return []userStoryPayload{}
	}
	rows, err := c.DB.ListActiveMatchStoriesForTeam(ctx, database.ListActiveMatchStoriesForTeamParams{
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		TeamLookupKey: teamLookupKey,
		RowLimit:      matchStoriesListLimit,
	})
	if err != nil {
		return []userStoryPayload{}
	}
	out := make([]userStoryPayload, 0, len(rows))
	for _, row := range rows {
		out = append(out, matchStoryFromRow(row))
	}
	return out
}

func validateMatchStoryTeamLookup(scopeType, scopeID, teamLookupKey string, matchInfo *MatchInfo) error {
	if strings.TrimSpace(scopeID) == "" || strings.TrimSpace(teamLookupKey) == "" {
		return errors.New("scope_id and team_lookup_key are required")
	}
	if scopeType != string(database.StoryScopeTypeMatch) {
		return errors.New("scope_type must be match")
	}
	if matchInfo == nil {
		return errors.New("match not found")
	}
	homeKey := youtube.LookupKeyForTeamName(matchInfo.HomeTeam)
	awayKey := youtube.LookupKeyForTeamName(matchInfo.AwayTeam)
	key := strings.TrimSpace(teamLookupKey)
	if key != homeKey && key != awayKey {
		return errors.New("team_lookup_key must match a team in this match")
	}
	return nil
}

func cloudinaryContextForStoryContent(contentType string) (string, error) {
	switch contentType {
	case string(database.StoryContentTypePhoto):
		return "match_story_photo", nil
	case string(database.StoryContentTypeVideo):
		return "match_story_video", nil
	default:
		return "", errors.New("content_type must be photo or video")
	}
}

// POST /v1/api/stories
func (c *Config) postMatchStory(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	if c.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "Database not configured")
		return
	}

	var req createMatchStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	scopeType := strings.TrimSpace(req.ScopeType)
	if scopeType == "" {
		scopeType = string(database.StoryScopeTypeMatch)
	}
	if scopeType != string(database.StoryScopeTypeMatch) {
		respondWithError(w, http.StatusBadRequest, "scope_type must be match")
		return
	}

	contentType := strings.TrimSpace(req.ContentType)
	if contentType != string(database.StoryContentTypePhoto) && contentType != string(database.StoryContentTypeVideo) {
		respondWithError(w, http.StatusBadRequest, "content_type must be photo or video")
		return
	}

	matchInfo, err := c.lookupMatchInfo(r.Context(), strings.TrimSpace(req.ScopeID))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "match not found")
		return
	}
	if err := validateMatchStoryTeamLookup(scopeType, req.ScopeID, req.TeamLookupKey, matchInfo); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	cloudinaryContext, err := cloudinaryContextForStoryContent(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.validateCloudinaryMediaURLForContext(strings.TrimSpace(req.MediaURL), cloudinaryContext); err != nil {
		if errors.Is(err, ErrCloudinaryURLValidationNotConfigured) {
			respondWithError(w, http.StatusInternalServerError, "Cloudinary is not configured")
			return
		}
		respondWithError(w, http.StatusBadRequest, "media_url is invalid")
		return
	}

	var caption sql.NullString
	if req.Caption != nil {
		c := strings.TrimSpace(*req.Caption)
		if len(c) > 500 {
			respondWithError(w, http.StatusBadRequest, "caption must be 500 characters or fewer")
			return
		}
		if c != "" {
			caption = sql.NullString{String: c, Valid: true}
		}
	}

	row, err := c.DB.CreateMatchStory(r.Context(), database.CreateMatchStoryParams{
		UserID:        userID,
		ScopeType:     database.StoryScopeTypeMatch,
		ScopeID:       strings.TrimSpace(req.ScopeID),
		TeamLookupKey: strings.TrimSpace(req.TeamLookupKey),
		ContentType:   database.StoryContentType(contentType),
		MediaUrl:      strings.TrimSpace(req.MediaURL),
		Caption:       caption,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create story")
		return
	}

	respondWithJSON(w, http.StatusCreated, matchStoryFromRecord(row))
}

type createContentReportRequest struct {
	ReportableType string  `json:"reportable_type"`
	ReportableID   string  `json:"reportable_id"`
	Reason         string  `json:"reason"`
	Description    *string `json:"description"`
}

var allowedReportReasons = map[string]struct{}{
	"spam":                  {},
	"harassment":            {},
	"inappropriate_content": {},
	"fake_team":             {},
	"other":                 {},
}

// POST /v1/api/reports
func (c *Config) postContentReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	if c.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "Database not configured")
		return
	}

	var req createContentReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reportableType := strings.TrimSpace(req.ReportableType)
	if reportableType != "story" {
		respondWithError(w, http.StatusBadRequest, "reportable_type must be story")
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if _, ok := allowedReportReasons[reason]; !ok {
		respondWithError(w, http.StatusBadRequest, "invalid reason")
		return
	}

	storyID, err := uuid.Parse(strings.TrimSpace(req.ReportableID))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid reportable_id")
		return
	}

	story, err := c.DB.GetMatchStoryByID(r.Context(), storyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "story not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to load story")
		return
	}
	if !story.IsActive {
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "already_removed"})
		return
	}

	var description sql.NullString
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		if d != "" {
			description = sql.NullString{String: d, Valid: true}
		}
	}

	if c.DBConn == nil {
		respondWithError(w, http.StatusInternalServerError, "Database not configured")
		return
	}

	tx, err := c.DBConn.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer func() { _ = tx.Rollback() }()

	qtx := c.DB.WithTx(tx)
	if _, err := qtx.DeactivateMatchStory(r.Context(), storyID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusInternalServerError, "Failed to remove story")
		return
	}
	report, err := qtx.CreateContentReport(r.Context(), database.CreateContentReportParams{
		ReporterID:     userID,
		ReportableType: reportableType,
		ReportableID:   storyID,
		Reason:         reason,
		Description:    description,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create report")
		return
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save report")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":                report.ID.String(),
		"reportable_type":   report.ReportableType,
		"reportable_id":     report.ReportableID.String(),
		"reason":            report.Reason,
		"status":            report.Status,
		"story_deactivated": true,
	})
}

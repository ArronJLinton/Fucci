package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/push"
	"github.com/go-chi/chi"
	"log"
)

var expoPushTokenRe = regexp.MustCompile(`^ExponentPushToken\[[A-Za-z0-9_-]+\]$`)

const maxPushDevicesPerUser = 5

type registerPushDeviceRequest struct {
	ExpoPushToken string  `json:"expo_push_token"`
	Platform      string  `json:"platform"`
	Timezone      string  `json:"timezone"`
	AppVersion    *string `json:"app_version"`
}

type pushDeviceResponse struct {
	ID            int32   `json:"id"`
	ExpoPushToken string  `json:"expo_push_token"`
	Platform      string  `json:"platform"`
	Timezone      string  `json:"timezone"`
	Enabled       bool    `json:"enabled"`
	LastSeenAt    string  `json:"last_seen_at"`
	AppVersion    *string `json:"app_version,omitempty"`
}

type pushPreferencesResponse struct {
	MasterEnabled  bool `json:"master_enabled"`
	DebatesEnabled bool `json:"debates_enabled"`
	NewsEnabled    bool `json:"news_enabled"`
	MatchesEnabled bool `json:"matches_enabled"`
}

type updatePushPreferencesRequest struct {
	MasterEnabled  *bool `json:"master_enabled"`
	DebatesEnabled *bool `json:"debates_enabled"`
	NewsEnabled    *bool `json:"news_enabled"`
	MatchesEnabled *bool `json:"matches_enabled"`
}

func toPushDeviceResponse(d database.PushDevices) pushDeviceResponse {
	var appVer *string
	if d.AppVersion.Valid {
		s := d.AppVersion.String
		appVer = &s
	}
	return pushDeviceResponse{
		ID:            d.ID,
		ExpoPushToken: d.ExpoPushToken,
		Platform:      d.Platform,
		Timezone:      d.Timezone,
		Enabled:       d.Enabled,
		LastSeenAt:    d.LastSeenAt.Format(time.RFC3339),
		AppVersion:    appVer,
	}
}

func toPushPreferencesResponse(p database.PushPreferences) pushPreferencesResponse {
	return pushPreferencesResponse{
		MasterEnabled:  p.MasterEnabled,
		DebatesEnabled: p.DebatesEnabled,
		NewsEnabled:    p.NewsEnabled,
		MatchesEnabled: p.MatchesEnabled,
	}
}

func validateExpoPushToken(token string) bool {
	return expoPushTokenRe.MatchString(strings.TrimSpace(token))
}

func validatePushPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "ios", "android":
		return true
	default:
		return false
	}
}

func validateTimezone(tz string) bool {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

func (c *Config) handleRegisterPushDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req registerPushDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.ExpoPushToken = strings.TrimSpace(req.ExpoPushToken)
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.Timezone = strings.TrimSpace(req.Timezone)

	if !validateExpoPushToken(req.ExpoPushToken) {
		respondWithError(w, http.StatusBadRequest, "invalid expo push token format")
		return
	}
	if !validatePushPlatform(req.Platform) {
		respondWithError(w, http.StatusBadRequest, "platform must be ios or android")
		return
	}
	if !validateTimezone(req.Timezone) {
		respondWithError(w, http.StatusBadRequest, "invalid IANA timezone")
		return
	}
	if c.pushStore() == nil {
		respondWithError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	ctx := r.Context()
	if _, err := c.pushStore().EnsurePushPreferences(ctx, userID); err != nil {
		logErrorAndRespond500(w, "push register EnsurePushPreferences", err, "PUSH_PREFS_ENSURE_FAILED")
		return
	}

	devices, err := c.pushStore().ListPushDevicesForUser(ctx, userID)
	if err != nil {
		logErrorAndRespond500(w, "push register ListPushDevicesForUser", err, "PUSH_DEVICE_LIST_FAILED")
		return
	}
	hasToken := false
	for _, d := range devices {
		if d.ExpoPushToken == req.ExpoPushToken {
			hasToken = true
			break
		}
	}
	if !hasToken && len(devices) >= maxPushDevicesPerUser {
		if err := c.pushStore().DeleteOldestPushDeviceForUser(ctx, userID); err != nil {
			logErrorAndRespond500(w, "push register DeleteOldestPushDeviceForUser", err, "PUSH_DEVICE_TRIM_FAILED")
			return
		}
	}

	var appVersion sql.NullString
	if req.AppVersion != nil && strings.TrimSpace(*req.AppVersion) != "" {
		appVersion = sql.NullString{String: strings.TrimSpace(*req.AppVersion), Valid: true}
	}

	device, err := c.pushStore().UpsertPushDevice(ctx, database.UpsertPushDeviceParams{
		UserID:        userID,
		ExpoPushToken: req.ExpoPushToken,
		Platform:      req.Platform,
		Timezone:      req.Timezone,
		AppVersion:    appVersion,
	})
	if err != nil {
		logErrorAndRespond500(w, "push register UpsertPushDevice", err, "PUSH_DEVICE_UPSERT_FAILED")
		return
	}

	respondWithJSON(w, http.StatusOK, toPushDeviceResponse(device))
	log.Printf("[push] registered device user=%d device_id=%d platform=%s", userID, device.ID, device.Platform)
}

func (c *Config) handleDeletePushDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if c.pushStore() == nil {
		respondWithError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	idStr := chi.URLParam(r, "deviceId")
	deviceID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || deviceID <= 0 {
		respondWithError(w, http.StatusBadRequest, "invalid device id")
		return
	}

	ctx := r.Context()
	existing, err := c.pushStore().GetPushDeviceByIDForUser(ctx, database.GetPushDeviceByIDForUserParams{
		ID:     int32(deviceID),
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "device not found")
			return
		}
		logErrorAndRespond500(w, "push delete GetPushDeviceByIDForUser", err, "PUSH_DEVICE_GET_FAILED")
		return
	}
	_ = existing

	if err := c.pushStore().DeletePushDeviceForUser(ctx, database.DeletePushDeviceForUserParams{
		ID:     int32(deviceID),
		UserID: userID,
	}); err != nil {
		logErrorAndRespond500(w, "push delete DeletePushDeviceForUser", err, "PUSH_DEVICE_DELETE_FAILED")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Config) handleGetPushPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if c.pushStore() == nil {
		respondWithError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	ctx := r.Context()
	prefs, err := c.pushStore().GetPushPreferences(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			prefs, err = c.pushStore().EnsurePushPreferences(ctx, userID)
		}
		if err != nil {
			logErrorAndRespond500(w, "push get preferences", err, "PUSH_PREFS_GET_FAILED")
			return
		}
	}
	respondWithJSON(w, http.StatusOK, toPushPreferencesResponse(prefs))
}

func (c *Config) handleUpdatePushPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if c.pushStore() == nil {
		respondWithError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var req updatePushPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := r.Context()
	if _, err := c.pushStore().EnsurePushPreferences(ctx, userID); err != nil {
		logErrorAndRespond500(w, "push update EnsurePushPreferences", err, "PUSH_PREFS_ENSURE_FAILED")
		return
	}

	params := database.UpdatePushPreferencesParams{UserID: userID}
	if req.MasterEnabled != nil {
		params.MasterEnabled = sql.NullBool{Bool: *req.MasterEnabled, Valid: true}
	}
	if req.DebatesEnabled != nil {
		params.DebatesEnabled = sql.NullBool{Bool: *req.DebatesEnabled, Valid: true}
	}
	if req.NewsEnabled != nil {
		params.NewsEnabled = sql.NullBool{Bool: *req.NewsEnabled, Valid: true}
	}
	if req.MatchesEnabled != nil {
		params.MatchesEnabled = sql.NullBool{Bool: *req.MatchesEnabled, Valid: true}
	}

	prefs, err := c.pushStore().UpdatePushPreferences(ctx, params)
	if err != nil {
		logErrorAndRespond500(w, "push update UpdatePushPreferences", err, "PUSH_PREFS_UPDATE_FAILED")
		return
	}
	respondWithJSON(w, http.StatusOK, toPushPreferencesResponse(prefs))
}

func (c *Config) handlePushTest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	svc := c.pushService()
	if svc == nil {
		respondWithError(w, http.StatusServiceUnavailable, "push service unavailable")
		return
	}

	log.Printf("[push] test send requested user=%d env=%s", userID, c.Environment)
	err := svc.SendToUser(r.Context(), push.SendRequest{
		UserID:      userID,
		CampaignKey: push.CampaignTest,
		Title:       "Fucci test push",
		Body:        "Push notifications are working.",
		Data: map[string]interface{}{
			"type":  "news",
			"route": "NewsWebView",
			"params": map[string]interface{}{
				"url": "https://www.fifa.com",
			},
		},
		SkipPrefs:  true,
		SkipDedupe: true,
	})
	if err != nil {
		logErrorAndRespond500(w, "push test send", err, "PUSH_TEST_SEND_FAILED")
		return
	}
	log.Printf("[push] test send accepted user=%d", userID)
	w.WriteHeader(http.StatusAccepted)
}

func (c *Config) pushService() *push.Service {
	if c.PushService != nil {
		return c.PushService
	}
	if c.DB == nil {
		return nil
	}
	client := &push.Client{Token: c.ExpoAccessToken}
	return &push.Service{Store: c.DB, Sender: client}
}

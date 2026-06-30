package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/push"
	"github.com/go-chi/chi"
)

type fakePushStore struct {
	devices     []database.PushDevices
	prefs       database.PushPreferences
	ledgerKeys  map[string]struct{}
	deliveryLog []database.PushDeliveryLog
}

func (f *fakePushStore) ListEnabledPushDevicesForUser(ctx context.Context, userID int32) ([]database.PushDevices, error) {
	var out []database.PushDevices
	for _, d := range f.devices {
		if d.UserID == userID && d.Enabled {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakePushStore) GetPushPreferences(ctx context.Context, userID int32) (database.PushPreferences, error) {
	if f.prefs.UserID == 0 {
		return database.PushPreferences{}, sql.ErrNoRows
	}
	return f.prefs, nil
}

func (f *fakePushStore) TryInsertPushSendLedger(ctx context.Context, arg database.TryInsertPushSendLedgerParams) (database.PushSendLedger, error) {
	if f.ledgerKeys == nil {
		f.ledgerKeys = map[string]struct{}{}
	}
	key := arg.CampaignKey + "|" + arg.LocalDate.Format("2006-01-02")
	if _, ok := f.ledgerKeys[key]; ok {
		return database.PushSendLedger{}, sql.ErrNoRows
	}
	f.ledgerKeys[key] = struct{}{}
	return database.PushSendLedger{ID: 1, UserID: arg.UserID, CampaignKey: arg.CampaignKey, LocalDate: arg.LocalDate}, nil
}

func (f *fakePushStore) InsertPushDeliveryLog(ctx context.Context, arg database.InsertPushDeliveryLogParams) (database.PushDeliveryLog, error) {
	row := database.PushDeliveryLog{
		ID:           int32(len(f.deliveryLog) + 1),
		UserID:       arg.UserID,
		PushDeviceID: arg.PushDeviceID,
		CampaignKey:  arg.CampaignKey,
		Title:        arg.Title,
		Status:       arg.Status,
	}
	f.deliveryLog = append(f.deliveryLog, row)
	return row, nil
}

func (f *fakePushStore) DisablePushDevice(ctx context.Context, id int32) error {
	for i := range f.devices {
		if f.devices[i].ID == id {
			f.devices[i].Enabled = false
		}
	}
	return nil
}

type fakePushDB struct {
	fakePushStore
	devicesByUser map[int32][]database.PushDevices
	nextID        int32
}

func (f *fakePushDB) ListPushDevicesForUser(ctx context.Context, userID int32) ([]database.PushDevices, error) {
	return f.devicesByUser[userID], nil
}

func (f *fakePushDB) EnsurePushPreferences(ctx context.Context, userID int32) (database.PushPreferences, error) {
	if f.prefs.UserID == 0 {
		f.prefs = database.PushPreferences{UserID: userID}
	}
	return f.prefs, nil
}

func (f *fakePushDB) UpsertPushDevice(ctx context.Context, arg database.UpsertPushDeviceParams) (database.PushDevices, error) {
	list := f.devicesByUser[arg.UserID]
	for i, d := range list {
		if d.ExpoPushToken == arg.ExpoPushToken {
			d.Timezone = arg.Timezone
			d.Platform = arg.Platform
			d.LastSeenAt = time.Now()
			list[i] = d
			f.devicesByUser[arg.UserID] = list
			f.devices = list
			return d, nil
		}
	}
	f.nextID++
	d := database.PushDevices{
		ID:            f.nextID,
		UserID:        arg.UserID,
		ExpoPushToken: arg.ExpoPushToken,
		Platform:      arg.Platform,
		Timezone:      arg.Timezone,
		Enabled:       true,
		LastSeenAt:    time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	f.devicesByUser[arg.UserID] = append(list, d)
	f.devices = f.devicesByUser[arg.UserID]
	return d, nil
}

func (f *fakePushDB) GetPushDeviceByIDForUser(ctx context.Context, arg database.GetPushDeviceByIDForUserParams) (database.PushDevices, error) {
	for _, d := range f.devicesByUser[arg.UserID] {
		if d.ID == arg.ID {
			return d, nil
		}
	}
	return database.PushDevices{}, sql.ErrNoRows
}

func (f *fakePushDB) DeletePushDeviceForUser(ctx context.Context, arg database.DeletePushDeviceForUserParams) error {
	list := f.devicesByUser[arg.UserID]
	out := list[:0]
	for _, d := range list {
		if d.ID != arg.ID {
			out = append(out, d)
		}
	}
	f.devicesByUser[arg.UserID] = out
	f.devices = out
	return nil
}

func (f *fakePushDB) DeleteOldestPushDeviceForUser(ctx context.Context, userID int32) error {
	list := f.devicesByUser[userID]
	if len(list) == 0 {
		return nil
	}
	oldestID := list[len(list)-1].ID
	return f.DeletePushDeviceForUser(ctx, database.DeletePushDeviceForUserParams{ID: oldestID, UserID: userID})
}

func (f *fakePushDB) GetPushPreferences(ctx context.Context, userID int32) (database.PushPreferences, error) {
	return f.fakePushStore.GetPushPreferences(ctx, userID)
}

func (f *fakePushDB) UpdatePushPreferences(ctx context.Context, arg database.UpdatePushPreferencesParams) (database.PushPreferences, error) {
	p := f.prefs
	if p.UserID == 0 {
		p.UserID = arg.UserID
	}
	if arg.MasterEnabled.Valid {
		p.MasterEnabled = arg.MasterEnabled.Bool
	}
	if arg.DebatesEnabled.Valid {
		p.DebatesEnabled = arg.DebatesEnabled.Bool
	}
	if arg.NewsEnabled.Valid {
		p.NewsEnabled = arg.NewsEnabled.Bool
	}
	if arg.MatchesEnabled.Valid {
		p.MatchesEnabled = arg.MatchesEnabled.Bool
	}
	f.prefs = p
	return p, nil
}

func newFakePushDB() *fakePushDB {
	return &fakePushDB{devicesByUser: map[int32][]database.PushDevices{}}
}

func authPushRequest(method, path string, body []byte, userID int32) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	return req.WithContext(auth.ContextWithClaims(req.Context(), &auth.JWTClaims{UserID: userID}))
}

func TestValidateExpoPushToken(t *testing.T) {
	if !validateExpoPushToken("ExponentPushToken[abc123XYZ_-]") {
		t.Fatal("expected valid token")
	}
	if validateExpoPushToken("invalid") {
		t.Fatal("expected invalid token")
	}
}

func TestValidatePushPlatform(t *testing.T) {
	if !validatePushPlatform("ios") || !validatePushPlatform("android") {
		t.Fatal("expected ios/android valid")
	}
	if validatePushPlatform("web") {
		t.Fatal("expected web invalid")
	}
}

func TestNormalizeTimezone(t *testing.T) {
	if got := normalizeTimezone("America/New_York"); got != "America/New_York" {
		t.Fatalf("expected America/New_York, got %q", got)
	}
	if got := normalizeTimezone("Not/A_Real_Zone"); got != "UTC" {
		t.Fatalf("expected UTC for unknown zone, got %q", got)
	}
	if got := normalizeTimezone(""); got != "UTC" {
		t.Fatalf("expected UTC for empty, got %q", got)
	}
}

func TestHandleRegisterPushDevice_Validation(t *testing.T) {
	cfg := &Config{PushDB: newFakePushDB()}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{
		"expo_push_token": "bad",
		"platform":        "ios",
		"timezone":        "America/New_York",
	})
	req := authPushRequest(http.MethodPost, "/push/devices", body, 1)
	cfg.handleRegisterPushDevice(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRegisterPushDevice_Success(t *testing.T) {
	fake := newFakePushDB()
	cfg := &Config{PushDB: fake}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{
		"expo_push_token": "ExponentPushToken[abc123XYZ_-]",
		"platform":        "ios",
		"timezone":        "America/New_York",
	})
	req := authPushRequest(http.MethodPost, "/push/devices", body, 42)
	cfg.handleRegisterPushDevice(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp pushDeviceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.ExpoPushToken != "ExponentPushToken[abc123XYZ_-]" || !resp.Enabled {
		t.Fatalf("unexpected device: %+v", resp)
	}
}

func TestHandleGetPushPreferences_CreatesDefaults(t *testing.T) {
	fake := newFakePushDB()
	cfg := &Config{PushDB: fake}
	rec := httptest.NewRecorder()
	req := authPushRequest(http.MethodGet, "/push/preferences", nil, 9)
	cfg.handleGetPushPreferences(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp pushPreferencesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.MasterEnabled || resp.DebatesEnabled {
		t.Fatalf("expected defaults off, got %+v", resp)
	}
}

func TestHandleUpdatePushPreferences_EnableAll(t *testing.T) {
	fake := newFakePushDB()
	cfg := &Config{PushDB: fake}
	rec := httptest.NewRecorder()
	master := true
	debates := true
	news := true
	matches := true
	body, _ := json.Marshal(updatePushPreferencesRequest{
		MasterEnabled:  &master,
		DebatesEnabled: &debates,
		NewsEnabled:    &news,
		MatchesEnabled: &matches,
	})
	req := authPushRequest(http.MethodPut, "/push/preferences", body, 9)
	cfg.handleUpdatePushPreferences(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp pushPreferencesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.MasterEnabled || !resp.DebatesEnabled || !resp.NewsEnabled || !resp.MatchesEnabled {
		t.Fatalf("expected all enabled, got %+v", resp)
	}
}

func TestHandleDeletePushDevice_Success(t *testing.T) {
	fake := newFakePushDB()
	fake.devicesByUser[3] = []database.PushDevices{
		{ID: 10, UserID: 3, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true},
	}
	cfg := &Config{PushDB: fake}
	rec := httptest.NewRecorder()
	req := authPushRequest(http.MethodDelete, "/push/devices/10", nil, 3)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("deviceId", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	cfg.handleDeletePushDevice(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if len(fake.devicesByUser[3]) != 0 {
		t.Fatalf("expected device removed, got %+v", fake.devicesByUser[3])
	}
}

func TestHandlePushTest_AcceptedInProduction(t *testing.T) {
	store := &fakePushStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 1, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
	}
	svc := &push.Service{Store: store, Sender: &fakeSender{}}
	cfg := &Config{Environment: "production", PushService: svc}
	rec := httptest.NewRecorder()
	req := authPushRequest(http.MethodPost, "/push/test", nil, 1)
	cfg.handlePushTest(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlePushTest_AcceptedInDevelopment(t *testing.T) {
	store := &fakePushStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 1, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
	}
	svc := &push.Service{Store: store, Sender: &fakeSender{}}
	cfg := &Config{Environment: "development", PushService: svc}
	rec := httptest.NewRecorder()
	req := authPushRequest(http.MethodPost, "/push/test", nil, 1)
	cfg.handlePushTest(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlePushTest_RespectsPrefsAndDedupe(t *testing.T) {
	store := &fakePushStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 1, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{UserID: 1, MasterEnabled: true},
	}
	svc := &push.Service{Store: store, Sender: &fakeSender{}}
	cfg := &Config{Environment: "production", PushService: svc}

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := authPushRequest(http.MethodPost, "/push/test", nil, 1)
		cfg.handlePushTest(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
		}
	}

	if len(store.deliveryLog) != 2 {
		t.Fatalf("expected 2 delivery log entries, got %+v", store.deliveryLog)
	}
	if store.deliveryLog[0].Status != "sent" {
		t.Fatalf("expected first test push to send, got %+v", store.deliveryLog[0])
	}
	if store.deliveryLog[1].Status != "skipped_dedupe" {
		t.Fatalf("expected second test push to dedupe, got %+v", store.deliveryLog[1])
	}
}

func TestPushService_SkipsWhenMasterDisabled(t *testing.T) {
	store := &fakePushStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 7, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{UserID: 7, MasterEnabled: false},
	}
	svc := &push.Service{
		Store:  store,
		Sender: &fakeSender{},
	}
	err := svc.SendToUser(context.Background(), push.SendRequest{
		UserID:      7,
		CampaignKey: "debate:daily",
		Title:       "Test",
		Category:    "debates",
	})
	if err != nil {
		t.Fatalf("SendToUser: %v", err)
	}
	if len(store.deliveryLog) != 1 || store.deliveryLog[0].Status != "skipped_prefs" {
		t.Fatalf("expected skipped_prefs log, got %+v", store.deliveryLog)
	}
}

type fakeSender struct{}

func (f *fakeSender) Send(ctx context.Context, messages []push.Message) ([]push.SendResult, error) {
	out := make([]push.SendResult, len(messages))
	for i, m := range messages {
		out[i] = push.SendResult{Token: m.To, Status: "ok", TicketID: "ticket-1"}
	}
	return out, nil
}

func TestPushService_Dedupe(t *testing.T) {
	store := &fakePushStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 7, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{UserID: 7, MasterEnabled: true, DebatesEnabled: true},
		ledgerKeys: map[string]struct{}{
			"debate:daily|" + time.Now().UTC().Format("2006-01-02"): {},
		},
	}
	svc := &push.Service{Store: store, Sender: &fakeSender{}}
	err := svc.SendToUser(context.Background(), push.SendRequest{
		UserID:      7,
		CampaignKey: "debate:daily",
		Title:       "Test",
		Category:    "debates",
		Timezone:    "UTC",
	})
	if err != nil {
		t.Fatalf("SendToUser: %v", err)
	}
	if len(store.deliveryLog) != 1 || store.deliveryLog[0].Status != "skipped_dedupe" {
		t.Fatalf("expected skipped_dedupe, got %+v", store.deliveryLog)
	}
}

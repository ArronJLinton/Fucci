package push

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

type memStore struct {
	devices     []database.PushDevices
	prefs       database.PushPreferences
	ledgerKeys  map[string]struct{}
	deliveryLog []database.PushDeliveryLog
}

func (m *memStore) ListEnabledPushDevicesForUser(ctx context.Context, userID int32) ([]database.PushDevices, error) {
	var out []database.PushDevices
	for _, d := range m.devices {
		if d.UserID == userID && d.Enabled {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *memStore) GetPushPreferences(ctx context.Context, userID int32) (database.PushPreferences, error) {
	if m.prefs.UserID == 0 {
		return database.PushPreferences{}, sql.ErrNoRows
	}
	return m.prefs, nil
}

func (m *memStore) TryInsertPushSendLedger(ctx context.Context, arg database.TryInsertPushSendLedgerParams) (database.PushSendLedger, error) {
	if m.ledgerKeys == nil {
		m.ledgerKeys = map[string]struct{}{}
	}
	key := arg.CampaignKey + "|" + arg.LocalDate.Format("2006-01-02")
	if _, ok := m.ledgerKeys[key]; ok {
		return database.PushSendLedger{}, sql.ErrNoRows
	}
	m.ledgerKeys[key] = struct{}{}
	return database.PushSendLedger{ID: 1, UserID: arg.UserID, CampaignKey: arg.CampaignKey, LocalDate: arg.LocalDate}, nil
}

func (m *memStore) InsertPushDeliveryLog(ctx context.Context, arg database.InsertPushDeliveryLogParams) (database.PushDeliveryLog, error) {
	row := database.PushDeliveryLog{
		ID:           int32(len(m.deliveryLog) + 1),
		UserID:       arg.UserID,
		PushDeviceID: arg.PushDeviceID,
		CampaignKey:  arg.CampaignKey,
		Title:        arg.Title,
		Status:       arg.Status,
	}
	m.deliveryLog = append(m.deliveryLog, row)
	return row, nil
}

func (m *memStore) DisablePushDevice(ctx context.Context, id int32) error {
	for i := range m.devices {
		if m.devices[i].ID == id {
			m.devices[i].Enabled = false
		}
	}
	return nil
}

type stubSender struct {
	sent int
}

func (s *stubSender) Send(ctx context.Context, messages []Message) ([]SendResult, error) {
	s.sent = len(messages)
	out := make([]SendResult, len(messages))
	for i := range messages {
		out[i] = SendResult{Status: "ok", TicketID: "t1"}
	}
	return out, nil
}

func TestSendToUser_NoDevices(t *testing.T) {
	t.Parallel()
	svc := &Service{Store: &memStore{}, Sender: &stubSender{}}
	if err := svc.SendToUser(context.Background(), SendRequest{UserID: 1, CampaignKey: "x", Title: "Hi"}); err != nil {
		t.Fatal(err)
	}
}

func TestSendToUser_CategoryDisabled(t *testing.T) {
	t.Parallel()
	store := &memStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 3, ExpoPushToken: "ExponentPushToken[x]", Enabled: true},
		},
		prefs: database.PushPreferences{UserID: 3, MasterEnabled: true, NewsEnabled: false},
	}
	svc := &Service{Store: store, Sender: &stubSender{}}
	if err := svc.SendToUser(context.Background(), SendRequest{
		UserID: 3, CampaignKey: "news:daily", Title: "News", Category: "news",
	}); err != nil {
		t.Fatal(err)
	}
	if len(store.deliveryLog) != 1 || store.deliveryLog[0].Status != "skipped_prefs" {
		t.Fatalf("expected skipped_prefs, got %+v", store.deliveryLog)
	}
}

func TestSendToUser_Success(t *testing.T) {
	t.Parallel()
	sender := &stubSender{}
	store := &memStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 5, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{
			UserID: 5, MasterEnabled: true, DebatesEnabled: true,
		},
	}
	svc := &Service{Store: store, Sender: sender}
	if err := svc.SendToUser(context.Background(), SendRequest{
		UserID: 5, CampaignKey: "debate:daily", Title: "Debate", Category: "debates", Timezone: "UTC",
	}); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 1 {
		t.Fatalf("expected 1 send, got %d", sender.sent)
	}
	if len(store.deliveryLog) != 1 || store.deliveryLog[0].Status != "sent" {
		t.Fatalf("expected sent log, got %+v", store.deliveryLog)
	}
}

func TestCategoryEnabled(t *testing.T) {
	t.Parallel()
	prefs := database.PushPreferences{MasterEnabled: true, MatchesEnabled: true}
	if !categoryEnabled(prefs, "matches") {
		t.Fatal("matches should be enabled")
	}
	if categoryEnabled(prefs, "news") {
		t.Fatal("news should be disabled")
	}
	if !categoryEnabled(prefs, "") {
		t.Fatal("empty category should pass")
	}
}

func TestLocalDateForTimezone(t *testing.T) {
	t.Parallel()
	d, err := localDateForTimezone("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if d.IsZero() {
		t.Fatal("expected non-zero date")
	}
	utc, err := localDateForTimezone("")
	if err != nil {
		t.Fatal(err)
	}
	if utc.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", utc.Location())
	}
}

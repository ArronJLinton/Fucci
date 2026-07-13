package push

import (
	"context"
	"database/sql"
	"errors"
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

func (m *memStore) DeletePushSendLedger(ctx context.Context, arg database.DeletePushSendLedgerParams) error {
	key := arg.CampaignKey + "|" + arg.LocalDate.Format("2006-01-02")
	delete(m.ledgerKeys, key)
	return nil
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

type flakySender struct {
	calls int
}

func (s *flakySender) Send(ctx context.Context, messages []Message) ([]SendResult, error) {
	s.calls++
	if s.calls == 1 {
		return nil, errors.New("expo unavailable")
	}
	out := make([]SendResult, len(messages))
	for i := range messages {
		out[i] = SendResult{Status: "ok", TicketID: "t1"}
	}
	return out, nil
}

type ticketFailureSender struct {
	calls int
}

func (s *ticketFailureSender) Send(ctx context.Context, messages []Message) ([]SendResult, error) {
	s.calls++
	out := make([]SendResult, len(messages))
	for i := range messages {
		if s.calls == 1 {
			out[i] = SendResult{Status: "error", Error: "Expo rejected ticket"}
			continue
		}
		out[i] = SendResult{Status: "ok", TicketID: "t1"}
	}
	return out, nil
}

type mixedResultSender struct {
	calls int
}

func (s *mixedResultSender) Send(ctx context.Context, messages []Message) ([]SendResult, error) {
	s.calls++
	return []SendResult{
		{Status: "ok", TicketID: "t1"},
		{Status: "error", Error: "Expo rejected ticket"},
	}, nil
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

func TestSendToUser_ReleasesDedupeAfterSendError(t *testing.T) {
	t.Parallel()
	sender := &flakySender{}
	store := &memStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 5, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{
			UserID: 5, MasterEnabled: true, DebatesEnabled: true,
		},
	}
	svc := &Service{Store: store, Sender: sender}
	req := SendRequest{
		UserID: 5, CampaignKey: "debate:daily", Title: "Debate", Category: "debates", Timezone: "UTC",
	}
	if err := svc.SendToUser(context.Background(), req); err == nil {
		t.Fatal("expected first send to fail")
	}
	if err := svc.SendToUser(context.Background(), req); err != nil {
		t.Fatalf("expected retry to send after ledger release: %v", err)
	}
	if sender.calls != 2 {
		t.Fatalf("expected sender to be called twice, got %d", sender.calls)
	}
	if len(store.deliveryLog) != 1 || store.deliveryLog[0].Status != "sent" {
		t.Fatalf("expected retry to record sent log, got %+v", store.deliveryLog)
	}
}

func TestSendToUser_ReleasesDedupeAfterAllTicketsFail(t *testing.T) {
	t.Parallel()
	sender := &ticketFailureSender{}
	store := &memStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 5, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{
			UserID: 5, MasterEnabled: true, DebatesEnabled: true,
		},
	}
	svc := &Service{Store: store, Sender: sender}
	req := SendRequest{
		UserID: 5, CampaignKey: "debate:daily", Title: "Debate", Category: "debates", Timezone: "UTC",
	}
	if err := svc.SendToUser(context.Background(), req); err == nil {
		t.Fatal("expected first send to fail when all tickets fail")
	}
	if err := svc.SendToUser(context.Background(), req); err != nil {
		t.Fatalf("expected retry to send after ledger release: %v", err)
	}
	if sender.calls != 2 {
		t.Fatalf("expected sender to be called twice, got %d", sender.calls)
	}
	if len(store.deliveryLog) != 2 || store.deliveryLog[0].Status != "failed" || store.deliveryLog[1].Status != "sent" {
		t.Fatalf("expected failed log followed by sent log, got %+v", store.deliveryLog)
	}
}

func TestSendToUser_KeepsDedupeAfterAnyTicketSucceeds(t *testing.T) {
	t.Parallel()
	sender := &mixedResultSender{}
	store := &memStore{
		devices: []database.PushDevices{
			{ID: 1, UserID: 5, ExpoPushToken: "ExponentPushToken[abc]", Enabled: true, Timezone: "UTC"},
			{ID: 2, UserID: 5, ExpoPushToken: "ExponentPushToken[def]", Enabled: true, Timezone: "UTC"},
		},
		prefs: database.PushPreferences{
			UserID: 5, MasterEnabled: true, DebatesEnabled: true,
		},
	}
	svc := &Service{Store: store, Sender: sender}
	req := SendRequest{
		UserID: 5, CampaignKey: "debate:daily", Title: "Debate", Category: "debates", Timezone: "UTC",
	}
	if err := svc.SendToUser(context.Background(), req); err != nil {
		t.Fatalf("expected mixed ticket result to count as delivered: %v", err)
	}
	if err := svc.SendToUser(context.Background(), req); err != nil {
		t.Fatalf("expected duplicate send to skip without error: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected dedupe to skip retry after partial success, got %d sender calls", sender.calls)
	}
	if len(store.deliveryLog) != 3 || store.deliveryLog[0].Status != "sent" || store.deliveryLog[1].Status != "failed" || store.deliveryLog[2].Status != "skipped_dedupe" {
		t.Fatalf("expected sent, failed, skipped_dedupe logs, got %+v", store.deliveryLog)
	}
}

func TestCategoryEnabled(t *testing.T) {
	t.Parallel()
	prefs := userPrefView{MatchesEnabled: true}
	if !categoryEnabled(prefs, CategoryMatches) {
		t.Fatal("matches should be enabled")
	}
	if categoryEnabled(prefs, CategoryNews) {
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

package push

import (
	"context"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

type dispatchMemStore struct {
	memStore
	candidates []database.ListSlotCampaignCandidatesRow
}

func (m *dispatchMemStore) ListSlotCampaignCandidates(ctx context.Context) ([]database.ListSlotCampaignCandidatesRow, error) {
	return m.candidates, nil
}

type lockCache struct {
	keys map[string]struct{}
}

func (c *lockCache) SetNX(_ context.Context, key string, _ time.Duration) (bool, error) {
	if c.keys == nil {
		c.keys = map[string]struct{}{}
	}
	if _, ok := c.keys[key]; ok {
		return false, nil
	}
	c.keys[key] = struct{}{}
	return true, nil
}

type fixedSelector struct {
	req *SendRequest
}

func (s fixedSelector) Select(_ context.Context, _ UserCandidate) (SelectResult, error) {
	if s.req == nil {
		return SelectResult{Skip: "no_content"}, nil
	}
	return SelectResult{Request: s.req}, nil
}

func TestDispatcher_RunCampaign_SendsInWindow(t *testing.T) {
	t.Parallel()
	sender := &stubSender{}
	store := &dispatchMemStore{
		memStore: memStore{
			devices: []database.PushDevices{
				{ID: 1, UserID: 7, ExpoPushToken: "ExponentPushToken[x]", Enabled: true, Timezone: "UTC"},
			},
			prefs: database.PushPreferences{UserID: 7, MasterEnabled: true, DebatesEnabled: true},
		},
		candidates: []database.ListSlotCampaignCandidatesRow{
			{UserID: 7, Timezone: "UTC", DebatesEnabled: true},
		},
	}
	svc := &Service{Store: store, Sender: sender}
	fixedNow := time.Date(2026, 6, 30, 18, 0, 0, 0, time.UTC)
	d := NewDispatcher(DispatcherConfig{
		Service: svc,
		Store:   store,
		Lock:   &lockCache{},
		Campaigns: []RegisteredCampaign{{
			Slot: SlotCampaign{
				Key: CampaignDebateDaily, Category: CategoryDebates,
				SendAt: TargetLocalTime{Hour: 18, Minute: 0},
			},
			Selector: fixedSelector{req: &SendRequest{Title: "Debate", Body: "Vote now"}},
		}},
		Now: func() time.Time { return fixedNow },
	})

	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 1 {
		t.Fatalf("expected 1 send, got %d", sender.sent)
	}
}

func TestDispatcher_RunCampaign_SkipsOutsideWindow(t *testing.T) {
	t.Parallel()
	sender := &stubSender{}
	store := &dispatchMemStore{
		memStore: memStore{prefs: database.PushPreferences{UserID: 7, MasterEnabled: true, DebatesEnabled: true}},
		candidates: []database.ListSlotCampaignCandidatesRow{
			{UserID: 7, Timezone: "UTC", DebatesEnabled: true},
		},
	}
	svc := &Service{Store: store, Sender: sender}
	outsideNow := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	d := NewDispatcher(DispatcherConfig{
		Service: svc,
		Store:   store,
		Campaigns: []RegisteredCampaign{{
			Slot: SlotCampaign{
				Key: CampaignDebateDaily, Category: CategoryDebates,
				SendAt: TargetLocalTime{Hour: 18, Minute: 0},
			},
			Selector: fixedSelector{req: &SendRequest{Title: "Debate"}},
		}},
		Now: func() time.Time { return outsideNow },
	})

	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 0 {
		t.Fatalf("expected no sends outside window, got %d", sender.sent)
	}
}

func TestDispatcher_UnconfiguredSelectorsSkip(t *testing.T) {
	t.Parallel()
	sender := &stubSender{}
	store := &dispatchMemStore{
		candidates: []database.ListSlotCampaignCandidatesRow{
			{UserID: 1, Timezone: "UTC", NewsEnabled: true, DebatesEnabled: true},
		},
	}
	svc := &Service{Store: store, Sender: sender}
	atNoon := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	d := NewDispatcher(DispatcherConfig{
		Service:   svc,
		Store:     store,
		Campaigns: RegisteredCampaigns(CampaignDeps{}),
		Now:       func() time.Time { return atNoon },
	})
	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 0 {
		t.Fatalf("expected stub selectors to skip, got %d sends", sender.sent)
	}
}

func TestDispatcher_ScanLockSkipsDuplicateCampaignEntry(t *testing.T) {
	t.Parallel()
	sender := &stubSender{}
	store := &dispatchMemStore{
		memStore: memStore{
			devices: []database.PushDevices{
				{ID: 1, UserID: 9, ExpoPushToken: "ExponentPushToken[x]", Enabled: true, Timezone: "UTC"},
			},
			prefs: database.PushPreferences{UserID: 9, MasterEnabled: true, NewsEnabled: true},
		},
		candidates: []database.ListSlotCampaignCandidatesRow{
			{UserID: 9, Timezone: "UTC", NewsEnabled: true},
		},
	}
	svc := &Service{Store: store, Sender: sender}
	atNoon := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	campaign := RegisteredCampaign{
		Slot: SlotCampaign{
			Key: CampaignNewsDaily, Category: CategoryNews,
			SendAt: TargetLocalTime{Hour: 12, Minute: 0},
		},
		Selector: fixedSelector{req: &SendRequest{Title: "News"}},
	}
	d := NewDispatcher(DispatcherConfig{
		Service:   svc,
		Store:     store,
		Campaigns: []RegisteredCampaign{campaign, campaign},
		Now:       func() time.Time { return atNoon },
	})
	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 1 {
		t.Fatalf("expected scan lock to allow only one campaign run, got %d", sender.sent)
	}
}

func TestDispatchMemStore_ListCandidatesEmpty(t *testing.T) {
	t.Parallel()
	store := &dispatchMemStore{}
	rows, err := store.ListSlotCampaignCandidates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty, got %d", len(rows))
	}
}

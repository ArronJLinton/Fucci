package push

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// scanLocker acquires distributed locks for campaign scan deduplication.
type scanLocker interface {
	SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// DispatchStore extends Store with candidate listing for slot campaigns.
type DispatchStore interface {
	Store
	ListSlotCampaignCandidates(ctx context.Context) ([]database.ListSlotCampaignCandidatesRow, error)
}

// DispatcherConfig wires the slot scanner and campaign runners.
type DispatcherConfig struct {
	Service   *Service
	Store     DispatchStore
	Lock      scanLocker
	Campaigns []RegisteredCampaign
	Window    time.Duration
	Now       func() time.Time
}

// Dispatcher runs slot campaigns on a fixed interval (scheduler.IntervalJob).
type Dispatcher struct {
	service   *Service
	store     DispatchStore
	lock      scanLocker
	campaigns []RegisteredCampaign
	window    time.Duration
	now       func() time.Time
}

func NewDispatcher(cfg DispatcherConfig) *Dispatcher {
	window := cfg.Window
	if window <= 0 {
		window = DefaultDeliveryWindow
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Dispatcher{
		service:   cfg.Service,
		store:     cfg.Store,
		lock:      cfg.Lock,
		campaigns: cfg.Campaigns,
		window:    window,
		now:       now,
	}
}

func (d *Dispatcher) Name() string { return "push-slot-scanner" }

func (d *Dispatcher) Run(ctx context.Context) error {
	if d.service == nil || d.store == nil {
		return fmt.Errorf("push dispatcher not configured")
	}
	slot := CurrentScanSlot(d.now())
	candidates, err := d.store.ListSlotCampaignCandidates(ctx)
	if err != nil {
		return err
	}
	for _, campaign := range d.campaigns {
		if err := d.runCampaign(ctx, campaign, slot, candidates); err != nil {
			log.Printf("[push-dispatcher] campaign=%s slot=%s error: %v", campaign.Slot.Key, slot, err)
		}
	}
	return nil
}

func (d *Dispatcher) runCampaign(
	ctx context.Context,
	campaign RegisteredCampaign,
	slot ScanSlot,
	candidates []database.ListSlotCampaignCandidatesRow,
) error {
	if !d.acquireScanLock(ctx, campaign.Slot.Key, slot) {
		return nil
	}

	stats := dispatchStats{}
	for _, row := range candidates {
		user := candidateFromRow(row)
		if !user.categoryEnabled(campaign.Slot.Category) {
			continue
		}
		inWindow, err := InDeliveryWindowForTimezone(user.Timezone, campaign.Slot.SendAt, d.window, d.now())
		if err != nil {
			log.Printf("[push-dispatcher] campaign=%s user=%d tz=%q: %v", campaign.Slot.Key, user.UserID, user.Timezone, err)
			stats.skippedTZ++
			continue
		}
		if !inWindow {
			stats.outsideWindow++
			continue
		}

		stats.inWindow++
		result, err := campaign.Selector.Select(ctx, user)
		if err != nil {
			log.Printf("[push-dispatcher] campaign=%s user=%d select error: %v", campaign.Slot.Key, user.UserID, err)
			stats.selectErrors++
			continue
		}
		if result.Request == nil {
			stats.selectorSkipped++
			continue
		}

		req := *result.Request
		req.UserID = user.UserID
		req.CampaignKey = campaign.Slot.Key
		req.Category = campaign.Slot.Category
		if req.Timezone == "" {
			req.Timezone = user.Timezone
		}
		if err := d.service.SendToUser(ctx, req); err != nil {
			log.Printf("[push-dispatcher] campaign=%s user=%d send error: %v", campaign.Slot.Key, user.UserID, err)
			stats.sendErrors++
			continue
		}
		stats.sent++
	}

	log.Printf(
		"[push-dispatcher] campaign=%s slot=%s in_window=%d sent=%d outside=%d selector_skip=%d tz_err=%d select_err=%d send_err=%d",
		campaign.Slot.Key, slot, stats.inWindow, stats.sent, stats.outsideWindow,
		stats.selectorSkipped, stats.skippedTZ, stats.selectErrors, stats.sendErrors,
	)
	return nil
}

func (d *Dispatcher) acquireScanLock(ctx context.Context, campaignKey string, slot ScanSlot) bool {
	if d.lock == nil {
		return true
	}
	key := ScanLockKey(campaignKey, slot)
	ok, err := d.lock.SetNX(ctx, key, ScanLockTTL)
	if err != nil {
		log.Printf("[push-dispatcher] SetNX(%s) failed: %v — proceeding without lock", key, err)
		return true
	}
	if !ok {
		log.Printf("[push-dispatcher] skip duplicate scan campaign=%s slot=%s", campaignKey, slot)
	}
	return ok
}

type dispatchStats struct {
	inWindow        int
	outsideWindow   int
	selectorSkipped int
	skippedTZ       int
	selectErrors    int
	sendErrors      int
	sent            int
}

func candidateFromRow(row database.ListSlotCampaignCandidatesRow) UserCandidate {
	return UserCandidate{
		UserID:         row.UserID,
		Timezone:       row.Timezone,
		DebatesEnabled: row.DebatesEnabled,
		NewsEnabled:    row.NewsEnabled,
		MatchesEnabled: row.MatchesEnabled,
	}
}

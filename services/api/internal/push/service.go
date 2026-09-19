package push

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// Store is the DB surface used by Service.
type Store interface {
	ListEnabledPushDevicesForUser(ctx context.Context, userID int32) ([]database.PushDevices, error)
	GetPushPreferences(ctx context.Context, userID int32) (database.PushPreferences, error)
	TryInsertPushSendLedger(ctx context.Context, arg database.TryInsertPushSendLedgerParams) (database.PushSendLedger, error)
	DeletePushSendLedger(ctx context.Context, arg database.DeletePushSendLedgerParams) error
	InsertPushDeliveryLog(ctx context.Context, arg database.InsertPushDeliveryLogParams) (database.PushDeliveryLog, error)
	DisablePushDevice(ctx context.Context, id int32) error
}

// Sender delivers push messages (Expo client in production).
type Sender interface {
	Send(ctx context.Context, messages []Message) ([]SendResult, error)
}

// SendRequest describes one logical notification to a user.
type SendRequest struct {
	UserID      int32
	CampaignKey string
	Title       string
	Body        string
	Data        map[string]interface{}
	// Category gates prefs when set: debates | news | matches
	Category string
	// SkipPrefs bypasses preference checks (admin test sends).
	SkipPrefs bool
	// SkipDedupe bypasses ledger insert (admin test sends).
	SkipDedupe bool
	Timezone   string // IANA for local_date dedupe; defaults UTC
}

// Service orchestrates preference filtering, dedupe, send, and logging.
type Service struct {
	Store  Store
	Sender Sender
}

func (s *Service) SendToUser(ctx context.Context, req SendRequest) error {
	if s.Store == nil || s.Sender == nil {
		return fmt.Errorf("push service not configured")
	}
	devices, err := s.Store.ListEnabledPushDevicesForUser(ctx, req.UserID)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		log.Printf("[push] SendToUser user=%d campaign=%s: no enabled devices registered", req.UserID, req.CampaignKey)
		return nil
	}

	if !req.SkipPrefs {
		prefs, err := s.Store.GetPushPreferences(ctx, req.UserID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.logSkipped(ctx, req, sql.NullInt32{}, "skipped_prefs", "no preferences row")
				return nil
			}
			return err
		}
		if !prefs.MasterEnabled {
			s.logSkipped(ctx, req, sql.NullInt32{}, "skipped_prefs", "master disabled")
			return nil
		}
		if !categoryEnabled(prefViewFromDB(prefs), req.Category) {
			s.logSkipped(ctx, req, sql.NullInt32{}, "skipped_prefs", "category disabled")
			return nil
		}
	}

	tz := req.Timezone
	if tz == "" && len(devices) > 0 {
		tz = devices[0].Timezone
	}
	localDate, err := localDateForTimezone(tz)
	if err != nil {
		localDate = time.Now().UTC().Truncate(24 * time.Hour)
	}

	if !req.SkipDedupe {
		ledger, err := s.Store.TryInsertPushSendLedger(ctx, database.TryInsertPushSendLedgerParams{
			UserID:      req.UserID,
			CampaignKey: req.CampaignKey,
			LocalDate:   localDate,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				s.logSkipped(ctx, req, sql.NullInt32{}, "skipped_dedupe", "already sent today")
				return nil
			}
			return err
		}
		_ = ledger
	}

	messages := make([]Message, 0, len(devices))
	for _, d := range devices {
		messages = append(messages, Message{
			To:       d.ExpoPushToken,
			Title:    req.Title,
			Body:     req.Body,
			Data:     req.Data,
			Sound:    "default",
			Priority: "high",
		})
	}

	results, err := s.Sender.Send(ctx, messages)
	if err != nil {
		if !req.SkipDedupe {
			s.releasePushSendLedger(ctx, req, localDate, "send error")
		}
		return err
	}

	delivered := false
	for i, res := range results {
		var deviceID sql.NullInt32
		if i < len(devices) {
			deviceID = sql.NullInt32{Int32: devices[i].ID, Valid: true}
			if res.DeviceNotRegistered {
				_ = s.Store.DisablePushDevice(ctx, devices[i].ID)
			}
		}
		status := "sent"
		var errMsg sql.NullString
		if res.Status != "ok" {
			status = "failed"
			if res.Error != "" {
				errMsg = sql.NullString{String: res.Error, Valid: true}
			}
		} else {
			delivered = true
		}
		var ticket sql.NullString
		if res.TicketID != "" {
			ticket = sql.NullString{String: res.TicketID, Valid: true}
		}
		_, _ = s.Store.InsertPushDeliveryLog(ctx, database.InsertPushDeliveryLogParams{
			UserID:       req.UserID,
			PushDeviceID: deviceID,
			CampaignKey:  req.CampaignKey,
			Title:        req.Title,
			ExpoTicketID: ticket,
			Status:       status,
			ErrorMessage: errMsg,
		})
	}
	if !delivered {
		if !req.SkipDedupe {
			s.releasePushSendLedger(ctx, req, localDate, "all tickets failed")
		}
		return fmt.Errorf("expo push accepted request but all tickets failed")
	}
	return nil
}

func (s *Service) releasePushSendLedger(ctx context.Context, req SendRequest, localDate time.Time, reason string) {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.Store.DeletePushSendLedger(releaseCtx, database.DeletePushSendLedgerParams{
		UserID:      req.UserID,
		CampaignKey: req.CampaignKey,
		LocalDate:   localDate,
	}); err != nil {
		log.Printf("[push] SendToUser user=%d campaign=%s: release ledger after %s: %v", req.UserID, req.CampaignKey, reason, err)
	}
}

func (s *Service) logSkipped(ctx context.Context, req SendRequest, deviceID sql.NullInt32, status, msg string) {
	_, _ = s.Store.InsertPushDeliveryLog(ctx, database.InsertPushDeliveryLogParams{
		UserID:       req.UserID,
		PushDeviceID: deviceID,
		CampaignKey:  req.CampaignKey,
		Title:        req.Title,
		Status:       status,
		ErrorMessage: sql.NullString{String: msg, Valid: true},
	})
}

func localDateForTimezone(tz string) (time.Time, error) {
	if tz == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	local, err := LocalTimeInTimezone(tz)
	if err != nil {
		return time.Time{}, err
	}
	y, m, d := local.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
}

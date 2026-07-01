package push

import "github.com/ArronJLinton/fucci-api/internal/database"

// Campaign keys and preference categories shared by slot and event-driven sends.
const (
	CampaignTest       = "test:manual"
	CampaignDebateDaily = "debate:daily"
	CampaignNewsDaily   = "news:daily"

	CategoryDebates = "debates"
	CategoryNews    = "news"
	CategoryMatches = "matches"
)

// userPrefView holds category opt-in flags for preference gating.
type userPrefView struct {
	DebatesEnabled bool
	NewsEnabled    bool
	MatchesEnabled bool
}

func prefViewFromDB(p database.PushPreferences) userPrefView {
	return userPrefView{
		DebatesEnabled: p.DebatesEnabled,
		NewsEnabled:    p.NewsEnabled,
		MatchesEnabled: p.MatchesEnabled,
	}
}

func categoryEnabled(p userPrefView, category string) bool {
	switch category {
	case CategoryDebates:
		return p.DebatesEnabled
	case CategoryNews:
		return p.NewsEnabled
	case CategoryMatches:
		return p.MatchesEnabled
	case "":
		return true
	default:
		return false
	}
}

// TargetLocalTime is a daily wall-clock send target in the user's IANA timezone.
type TargetLocalTime struct {
	Hour   int
	Minute int
}

// SlotCampaign describes a recurring daily push dispatched via the slot scanner.
type SlotCampaign struct {
	Key      string
	Category string
	SendAt   TargetLocalTime
}

// DefaultSlotCampaigns returns the Phase 2 daily slot campaigns (debate 6pm, news 12pm local).
func DefaultSlotCampaigns() []SlotCampaign {
	return []SlotCampaign{
		{Key: CampaignDebateDaily, Category: CategoryDebates, SendAt: TargetLocalTime{Hour: 18, Minute: 0}},
		{Key: CampaignNewsDaily, Category: CategoryNews, SendAt: TargetLocalTime{Hour: 12, Minute: 0}},
	}
}

// RegisteredCampaign pairs a slot campaign with its content selector.
type RegisteredCampaign struct {
	Slot     SlotCampaign
	Selector CampaignSelector
}

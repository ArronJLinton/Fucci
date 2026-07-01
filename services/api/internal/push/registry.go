package push

// CampaignDeps wires real selectors for slot campaigns.
type CampaignDeps struct {
	NewsFeed NewsFeed
	NewsOpens NewsOpenStore
	Debates  DebatePushStore
}

// RegisteredCampaigns returns daily slot campaigns with content selectors.
func RegisteredCampaigns(deps CampaignDeps) []RegisteredCampaign {
	selectors := map[string]CampaignSelector{
		CampaignNewsDaily:   &NewsDailySelector{Feed: deps.NewsFeed, Opens: deps.NewsOpens},
		CampaignDebateDaily: &DebateDailySelector{Store: deps.Debates},
	}
	out := make([]RegisteredCampaign, len(DefaultSlotCampaigns()))
	for i, slot := range DefaultSlotCampaigns() {
		sel := selectors[slot.Key]
		if sel == nil {
			sel = UnimplementedSelector{Name: slot.Key}
		}
		out[i] = RegisteredCampaign{Slot: slot, Selector: sel}
	}
	return out
}

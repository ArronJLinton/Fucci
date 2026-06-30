package push

import "context"

// UserCandidate is a user eligible for slot campaign dispatch.
type UserCandidate struct {
	UserID         int32
	Timezone       string
	DebatesEnabled bool
	NewsEnabled    bool
	MatchesEnabled bool
}

func (u UserCandidate) categoryEnabled(category string) bool {
	return categoryEnabled(userPrefView{
		DebatesEnabled: u.DebatesEnabled,
		NewsEnabled:    u.NewsEnabled,
		MatchesEnabled: u.MatchesEnabled,
	}, category)
}

// SelectResult is returned by CampaignSelector. Request nil means skip (see Skip reason).
type SelectResult struct {
	Request *SendRequest
	Skip    string
}

// CampaignSelector picks notification content for one user in a slot campaign.
type CampaignSelector interface {
	Select(ctx context.Context, user UserCandidate) (SelectResult, error)
}

// UnimplementedSelector skips every user until campaign content logic is wired.
type UnimplementedSelector struct {
	Name string
}

func (s UnimplementedSelector) Select(_ context.Context, _ UserCandidate) (SelectResult, error) {
	reason := "selector_not_implemented"
	if s.Name != "" {
		reason = reason + ":" + s.Name
	}
	return SelectResult{Skip: reason}, nil
}

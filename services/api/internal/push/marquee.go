package push

// Marquee ranks API-Football team ids by FIFA rank for fixture filtering.
type Marquee struct {
	// team_id -> fifa_rank
	ranks   map[int32]int32
	maxRank int32
}

func NewMarquee(rankRows []RankRow, maxRank int32) *Marquee {
	if maxRank <= 0 {
		maxRank = MarqueeMaxFIFARank
	}
	ranks := make(map[int32]int32, len(rankRows))
	for _, row := range rankRows {
		ranks[row.TeamID] = row.FIFARank
	}
	return &Marquee{ranks: ranks, maxRank: maxRank}
}

// RankRow is one national team ranking entry.
type RankRow struct {
	TeamID   int32
	FIFARank int32
}

// IsMarquee reports whether either team is ranked within maxRank.
// When no rankings are loaded, all fixtures are treated as marquee.
func (m *Marquee) IsMarquee(homeTeamID, awayTeamID int) bool {
	if m == nil || len(m.ranks) == 0 {
		return true
	}
	best := m.bestRank(homeTeamID, awayTeamID)
	return best > 0 && best <= int(m.maxRank)
}

func (m *Marquee) bestRank(homeTeamID, awayTeamID int) int {
	best := 0
	for _, id := range []int{homeTeamID, awayTeamID} {
		if id <= 0 {
			continue
		}
		if rank, ok := m.ranks[int32(id)]; ok {
			r := int(rank)
			if best == 0 || r < best {
				best = r
			}
		}
	}
	return best
}

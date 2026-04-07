package futbol

import "fmt"

func FormatLeagueStandingsSummary(leagueID, season string) string {
	return fmt.Sprintf("League standings context for league %s season %s", leagueID, season)
}

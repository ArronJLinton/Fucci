package futbol

import (
	"fmt"
	"strings"
)

func MatchesFromRaw(raw map[string]any) MatchesDTO {
	dto := MatchesDTO{}
	if raw == nil {
		return dto
	}
	if results, ok := raw["results"].(float64); ok {
		dto.Results = int(results)
	}
	response, ok := raw["response"].([]any)
	if !ok {
		return dto
	}
	dto.Matches = make([]MatchDTO, 0, len(response))
	for _, row := range response {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		fixture, _ := item["fixture"].(map[string]any)
		statusMap := map[string]any{}
		if fixture != nil {
			if s, ok := fixture["status"].(map[string]any); ok {
				statusMap = s
			}
		}
		short, _ := statusMap["short"].(string)
		matchID := ""
		if idNum, ok := fixture["id"].(float64); ok {
			matchID = fmt.Sprintf("%.0f", idNum)
		}
		dto.Matches = append(dto.Matches, MatchDTO{
			ID:     matchID,
			Status: NormalizeMatchStatus(short),
		})
	}
	return dto
}

func NormalizeMatchStatus(status string) MatchStatus {
	s := strings.TrimSpace(strings.ToUpper(status))
	switch s {
	case "LIVE", "1H", "2H", "ET", "P", "BT":
		return MatchStatusLive
	case "HT":
		return MatchStatusInPlay
	case "NS", "TBD":
		return MatchStatusScheduled
	case "FT", "AET", "PEN", "CANC", "ABD":
		return MatchStatusFinished
	default:
		return MatchStatusScheduled
	}
}

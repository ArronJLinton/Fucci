package futbol

import (
	"fmt"
	"strings"
	"unicode"
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

func NormalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, ".", "")
	name = strings.Join(strings.Fields(name), " ")
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, name)
	return name
}

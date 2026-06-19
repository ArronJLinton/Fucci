package youtube

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NormalizeTeamLookupKey mirrors mobile normalizeMatchTeamNameForLookup for DB registry lookup.
func NormalizeTeamLookupKey(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ := transform.String(t, name)
	s = strings.ToLower(s)
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// Aliases maps alternate API-Football team name keys to registry lookup_key values.
var Aliases = map[string]string{
	"turkey":  "turkiye",
	"usa":     "united states",
	"korea republic": "south korea",
}

// LookupKeyForTeamName returns the DB lookup_key for a match team display name.
func LookupKeyForTeamName(teamName string) string {
	key := NormalizeTeamLookupKey(teamName)
	if alias, ok := Aliases[key]; ok {
		return alias
	}
	return key
}

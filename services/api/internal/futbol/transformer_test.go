package futbol

import "testing"

func TestMatchesFromRaw_MapsCoreFields(t *testing.T) {
	raw := map[string]any{
		"results": float64(2),
		"response": []any{
			map[string]any{
				"fixture": map[string]any{
					"id": float64(12345),
					"status": map[string]any{
						"short": "FT",
					},
				},
			},
			map[string]any{
				"fixture": map[string]any{
					"id": float64(777),
					"status": map[string]any{
						"short": "1H",
					},
				},
			},
		},
	}

	dto := MatchesFromRaw(raw)
	if dto.Results != 2 {
		t.Fatalf("expected results=2, got %d", dto.Results)
	}
	if len(dto.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(dto.Matches))
	}
	if dto.Matches[0].ID != "12345" || dto.Matches[0].Status != MatchStatusFinished {
		t.Fatalf("unexpected first match mapping: %#v", dto.Matches[0])
	}
	if dto.Matches[1].ID != "777" || dto.Matches[1].Status != MatchStatusLive {
		t.Fatalf("unexpected second match mapping: %#v", dto.Matches[1])
	}
}

func TestNormalizeName_RemovesPunctuationAndCase(t *testing.T) {
	got := NormalizeName("  K. Mbappé #9 ")
	want := "k mbappé "
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

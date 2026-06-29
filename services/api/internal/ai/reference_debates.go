package ai

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ReferenceDebate is a curated World Cup debate used to steer AI generation quality.
type ReferenceDebate struct {
	ID          int      `json:"id"`
	Headline    string   `json:"headline"`
	Description string   `json:"description"`
	Comments    []string `json:"comments"` // three "Fucci's Take" seeded comments
}

//go:embed reference_debates.json
var referenceDebatesJSON []byte

var (
	referenceDebatesOnce sync.Once
	referenceDebates     []ReferenceDebate
	referenceDebatesErr  error
)

// LoadReferenceDebates returns the embedded FIFA World Cup reference debate corpus.
func LoadReferenceDebates() ([]ReferenceDebate, error) {
	referenceDebatesOnce.Do(func() {
		referenceDebatesErr = json.Unmarshal(referenceDebatesJSON, &referenceDebates)
	})
	return referenceDebates, referenceDebatesErr
}

// ReferenceDebateCount returns how many reference debates are embedded.
func ReferenceDebateCount() int {
	debates, err := LoadReferenceDebates()
	if err != nil {
		return 0
	}
	return len(debates)
}

// ReferenceDebatesPromptSection returns a system-prompt appendix with style examples
// from the embedded corpus (full examples + headline index for tone matching).
func ReferenceDebatesPromptSection() string {
	debates, err := LoadReferenceDebates()
	if err != nil || len(debates) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\nREFERENCE CORPUS — FIFA World Cup debate style guide (1966–2026):\n")
	b.WriteString("Match this tone: bold polarizing headlines, stakes-driven descriptions, and seeded \"Fucci's Take\" comments that sound like passionate fans in the stands — not TV pundits.\n")
	b.WriteString("Headlines should be plain-language yes/no propositions. Descriptions should draw on notable controversy, records, or narratives.\n")
	b.WriteString("Each debate needs exactly three seeded comments labeled \"Fucci's Take\" in the app:\n")
	b.WriteString("  (1) backs the agree side with heat and conviction\n")
	b.WriteString("  (2) backs the disagree side with a sharp counter\n")
	b.WriteString("  (3) a wildcard/hot-take angle that still fits the debate\n")
	b.WriteString("Comments: short, conversational, PG-13, emotionally charged — never analyst jargon unless a normal fan would say it.\n\n")

	exampleCount := 5
	if len(debates) < exampleCount {
		exampleCount = len(debates)
	}
	b.WriteString("FULL EXAMPLES (headline + description + three Fucci's Takes):\n")
	for i := 0; i < exampleCount; i++ {
		appendReferenceDebateExample(&b, debates[i])
	}

	b.WriteString("\nALL REFERENCE HEADLINES (match this quality and specificity when generating new debates):\n")
	headlineCount := 25
	if len(debates) < headlineCount {
		headlineCount = len(debates)
	}
	for i := 0; i < headlineCount; i++ {
		b.WriteString(fmt.Sprintf("- [%d] %s\n", debates[i].ID, debates[i].Headline))
	}

	return b.String()
}

func appendReferenceDebateExample(b *strings.Builder, d ReferenceDebate) {
	b.WriteString(fmt.Sprintf("\nExample #%d:\n", d.ID))
	b.WriteString(fmt.Sprintf("Headline: %s\n", d.Headline))
	b.WriteString(fmt.Sprintf("Description: %s\n", d.Description))
	for i, c := range d.Comments {
		if strings.TrimSpace(c) == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("Fucci's Take %d: %s\n", i+1, c))
	}
}

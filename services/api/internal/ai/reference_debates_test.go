package ai

import (
	"strings"
	"testing"
)

func TestLoadReferenceDebates(t *testing.T) {
	debates, err := LoadReferenceDebates()
	if err != nil {
		t.Fatalf("LoadReferenceDebates() error = %v", err)
	}
	if len(debates) != 100 {
		t.Fatalf("LoadReferenceDebates() count = %d, want 100", len(debates))
	}
	for _, d := range debates {
		if strings.TrimSpace(d.Headline) == "" {
			t.Fatalf("debate #%d missing headline", d.ID)
		}
		if strings.TrimSpace(d.Description) == "" {
			t.Fatalf("debate #%d missing description", d.ID)
		}
		if len(d.Comments) != 3 {
			t.Fatalf("debate #%d has %d comments, want 3", d.ID, len(d.Comments))
		}
		for i, c := range d.Comments {
			if strings.TrimSpace(c) == "" {
				t.Fatalf("debate #%d comment %d is empty", d.ID, i+1)
			}
		}
	}
}

func TestReferenceDebatesPromptSection(t *testing.T) {
	section := ReferenceDebatesPromptSection()
	if section == "" {
		t.Fatal("ReferenceDebatesPromptSection() returned empty string")
	}
	if !strings.Contains(section, "REFERENCE CORPUS") {
		t.Error("prompt section should mention REFERENCE CORPUS")
	}
	if !strings.Contains(section, "Fucci's Take") {
		t.Error("prompt section should mention Fucci's Take")
	}
	if !strings.Contains(section, "Messi scored all five") {
		t.Error("prompt section should include example headline from corpus")
	}
}

func TestBuildSystemPrompt_IncludesReferenceCorpus(t *testing.T) {
	pg := &PromptGenerator{}
	pre := pg.buildSystemPrompt("pre_match")
	post := pg.buildSystemPrompt("post_match")
	for _, prompt := range []string{pre, post} {
		if !strings.Contains(prompt, "REFERENCE CORPUS") {
			t.Error("system prompt should include reference corpus section")
		}
		if !strings.Contains(prompt, "Fucci's Take") {
			t.Error("system prompt should mention Fucci's Take comments")
		}
	}
}

func TestBuildSystemPromptForSet_IncludesReferenceCorpus(t *testing.T) {
	pg := &PromptGenerator{}
	prompt := pg.buildSystemPromptForSet("pre_match", 3)
	if !strings.Contains(prompt, "REFERENCE CORPUS") {
		t.Error("set system prompt should include reference corpus section")
	}
	if !strings.Contains(prompt, "Fucci's Take") {
		t.Error("set system prompt should mention Fucci's Take comments")
	}
}

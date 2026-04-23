package snapchat

import (
	"strings"
	"testing"
)

func TestNormalizeSnapchatUsername(t *testing.T) {
	t.Parallel()
	if _, ok := NormalizeSnapchatUsername(""); ok {
		t.Fatal("empty should reject")
	}
	if _, ok := NormalizeSnapchatUsername("bad spaces"); ok {
		t.Fatal("spaces in username should reject")
	}
	u, ok := NormalizeSnapchatUsername("  PSG  ")
	if !ok || u != "psg" {
		t.Fatalf("want psg, ok=%v got %q", ok, u)
	}
	raw := strings.Repeat("a", SnapchatUsernameMaxQueryBytes+1)
	if _, ok := NormalizeSnapchatUsername(raw); ok {
		t.Fatal("oversized raw query should reject before plausible check")
	}
	if _, ok := NormalizeSnapchatUsername(strings.Repeat("a", SnapchatUsernameMaxRunes+1)); ok {
		t.Fatal("over max normalized length should reject")
	}
}

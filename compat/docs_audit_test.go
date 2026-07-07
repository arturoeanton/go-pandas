package compat_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestKnownDifferencesDoesNotClaimCategoricalObjectBacked guards the
// docs against the pre-v0.7 claim that categorical data has no typed
// storage — a stale statement fixed in v0.7.1.
func TestKnownDifferencesDoesNotClaimCategoricalObjectBacked(t *testing.T) {
	stale := []string{
		"categorical data have no typed storage",
		"categorical data has no typed storage",
		"categorical data and",
	}
	for _, file := range []string{
		"known_differences.md",
		filepath.Join("..", "docs", "dtype_semantics.md"),
		filepath.Join("..", "docs", "categorical.md"),
		filepath.Join("..", "README.md"),
	} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := strings.ToLower(string(raw))
		for _, phrase := range stale {
			if strings.Contains(text, phrase) {
				t.Errorf("%s still contains stale claim %q", file, phrase)
			}
		}
	}
	// And the positive claim must be present where it matters.
	raw, err := os.ReadFile("known_differences.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "typed storage since v0.7") {
		t.Error("known_differences.md must state categorical typed storage since v0.7")
	}
}

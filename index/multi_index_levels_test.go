package index

import (
	"fmt"
	"testing"
)

func levelsMI(t *testing.T) *MultiIndex {
	t.Helper()
	mi, err := NewMultiIndexFromTuples([][]any{
		{"AR", "BA", 2023}, {"AR", "CO", 2024}, {"BR", "SP", 2023}, {"AR", "BA", 2024},
	}, []string{"c", "t", "y"})
	if err != nil {
		t.Fatal(err)
	}
	return mi
}

func TestDropLevel(t *testing.T) {
	mi := levelsMI(t)
	before := fmt.Sprint(mi.Tuples())

	byName, err := mi.DropLevel("t")
	if err != nil {
		t.Fatal(err)
	}
	dm := byName.(*MultiIndex)
	if dm.NLevels() != 2 || dm.Names()[1] != "y" {
		t.Fatalf("droplevel shape: %v", dm.Names())
	}
	if got := dm.Tuple(3); got[0] != "AR" || got[1] != 2024 {
		t.Fatalf("droplevel tuple = %v", got)
	}
	byPos, err := mi.DropLevel(-1) // negative position, pandas-style
	if err != nil {
		t.Fatal(err)
	}
	if byPos.(*MultiIndex).Names()[1] != "t" {
		t.Fatalf("droplevel(-1) names = %v", byPos.(*MultiIndex).Names())
	}
	// Dropping to one level yields a flat index.
	two, _ := NewMultiIndexFromTuples([][]any{{"a", 1}, {"b", 2}}, []string{"k", "n"})
	flat, err := two.DropLevel("n")
	if err != nil {
		t.Fatal(err)
	}
	if _, isMI := flat.(*MultiIndex); isMI {
		t.Fatal("2-level drop must produce a flat index")
	}
	if flat.At(0) != "a" {
		t.Fatalf("flat label = %v", flat.At(0))
	}
	if _, err := mi.DropLevel("nope"); err == nil {
		t.Fatal("unknown level must error")
	}
	if _, err := mi.DropLevel(7); err == nil {
		t.Fatal("out-of-range level must error")
	}
	if fmt.Sprint(mi.Tuples()) != before {
		t.Fatal("DropLevel mutated the input")
	}
}

func TestSwapAndReorderLevels(t *testing.T) {
	mi := levelsMI(t)

	sw, err := mi.SwapLevel("c", "y")
	if err != nil {
		t.Fatal(err)
	}
	if n := sw.Names(); n[0] != "y" || n[2] != "c" {
		t.Fatalf("swap names = %v", n)
	}
	if got := sw.Tuple(0); got[0] != 2023 || got[2] != "AR" {
		t.Fatalf("swap tuple = %v", got)
	}
	RequireLookupAgrees(t, sw)

	// Default swaps the last two levels.
	def, err := mi.SwapLevel()
	if err != nil {
		t.Fatal(err)
	}
	if n := def.Names(); n[1] != "y" || n[2] != "t" {
		t.Fatalf("default swap names = %v", n)
	}

	re, err := mi.ReorderLevels("y", "c", "t")
	if err != nil {
		t.Fatal(err)
	}
	if n := re.Names(); n[0] != "y" || n[1] != "c" || n[2] != "t" {
		t.Fatalf("reorder names = %v", n)
	}
	if _, err := mi.ReorderLevels("y", "c"); err == nil {
		t.Fatal("incomplete reorder must error")
	}
	if _, err := mi.ReorderLevels("y", "y", "t"); err == nil {
		t.Fatal("repeated level must error")
	}
	if _, err := mi.SwapLevel("c"); err == nil {
		t.Fatal("single selector must error")
	}
}

// RequireLookupAgrees re-checks the lookup-vs-scan invariant on a
// derived index (level ops build fresh lookups).
func RequireLookupAgrees(t *testing.T, mi *MultiIndex) {
	t.Helper()
	for i := 0; i < mi.Len(); i++ {
		found := false
		for _, p := range mi.PositionsTuple(mi.Tuple(i)) {
			if p == i {
				found = true
			}
		}
		if !found {
			t.Fatalf("lookup misses row %d", i)
		}
	}
}

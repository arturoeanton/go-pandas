package index

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func sampleMI(t *testing.T) *MultiIndex {
	t.Helper()
	mi, err := NewMultiIndexFromArrays(
		[][]any{{"AR", "AR", "BR", "AR"}, {"BA", "CO", "SP", "BA"}},
		[]string{"country", "city"})
	if err != nil {
		t.Fatal(err)
	}
	return mi
}

func TestMultiIndexFromArraysShape(t *testing.T) {
	mi := sampleMI(t)
	if mi.Len() != 4 || mi.NLevels() != 2 {
		t.Fatalf("shape: len=%d levels=%d", mi.Len(), mi.NLevels())
	}
	if names := mi.Names(); names[0] != "country" || names[1] != "city" {
		t.Fatalf("names = %v", names)
	}
	// Levels are sorted unique labels (pandas parity).
	levels := mi.Levels()
	if levels[0][0] != "AR" || levels[0][1] != "BR" || len(levels[0]) != 2 {
		t.Fatalf("level 0 = %v", levels[0])
	}
	if levels[1][0] != "BA" || levels[1][1] != "CO" || levels[1][2] != "SP" {
		t.Fatalf("level 1 = %v", levels[1])
	}
	codes := mi.Codes()
	want0 := []int32{0, 0, 1, 0}
	want1 := []int32{0, 1, 2, 0}
	for i := range want0 {
		if codes[0][i] != want0[i] || codes[1][i] != want1[i] {
			t.Fatalf("codes = %v", codes)
		}
	}
}

func TestMultiIndexFromArraysErrors(t *testing.T) {
	if _, err := NewMultiIndexFromArrays(nil, nil); err == nil {
		t.Fatal("empty arrays must error")
	}
	if _, err := NewMultiIndexFromArrays([][]any{{1, 2}, {1}}, nil); err == nil {
		t.Fatal("ragged arrays must error")
	}
	if _, err := NewMultiIndexFromArrays([][]any{{1}}, []string{"a", "b"}); err == nil {
		t.Fatal("name count mismatch must error")
	}
	if _, err := NewMultiIndexFromArrays([][]any{{[]int{1}}}, nil); err == nil {
		t.Fatal("unhashable label must error")
	}
}

func TestMultiIndexFromTuples(t *testing.T) {
	mi, err := NewMultiIndexFromTuples(
		[][]any{{"x", 1}, {"y", nil}, {"x", 1}}, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if mi.Len() != 3 {
		t.Fatalf("len = %d", mi.Len())
	}
	if got := mi.Tuple(1); got[0] != "y" || got[1] != nil {
		t.Fatalf("tuple(1) = %v", got)
	}
	if !mi.IsNA(1, 1) || mi.IsNA(0, 0) {
		t.Fatal("NA flags wrong")
	}
	// Duplicate tuples allowed and both found.
	if got := mi.PositionsTuple([]any{"x", 1}); len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("duplicate positions = %v", got)
	}
	if _, err := NewMultiIndexFromTuples([][]any{{"x", 1}, {"y"}}, nil); err == nil {
		t.Fatal("ragged tuples must error")
	}
	if _, err := NewMultiIndexFromTuples(nil, nil); err == nil {
		t.Fatal("empty tuples must error")
	}
}

func TestMultiIndexMixedFamilies(t *testing.T) {
	// string + int level values in ONE level: mixed families fall back
	// to first-appearance order (documented) instead of erroring.
	mi, err := NewMultiIndexFromArrays([][]any{{"b", 1, "b"}}, []string{"k"})
	if err != nil {
		t.Fatal(err)
	}
	levels := mi.Levels()
	if levels[0][0] != "b" || levels[0][1] != 1 {
		t.Fatalf("mixed level order = %v", levels[0])
	}
	if got := mi.PositionsTuple([]any{1}); len(got) != 1 || got[0] != 1 {
		t.Fatalf("mixed lookup = %v", got)
	}
}

func TestMultiIndexTypedLevels(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.AddDate(0, 1, 0)
	mi, err := NewMultiIndexFromArrays(
		[][]any{{"a", "b"}, {t1, t0}}, []string{"k", "when"})
	if err != nil {
		t.Fatal(err)
	}
	if lv := mi.Levels()[1]; lv[0] != t0 || lv[1] != t1 {
		t.Fatalf("time level not sorted: %v", lv)
	}
	// Numeric widths collapse in lookups (int 1 == int64 1 == 1.0).
	ni, _ := NewMultiIndexFromArrays([][]any{{1, 2}}, []string{"n"})
	if got := ni.PositionsTuple([]any{int64(1)}); len(got) != 1 {
		t.Fatalf("int64 lookup vs int level = %v", got)
	}
	if got := ni.PositionsTuple([]any{1.0}); len(got) != 1 {
		t.Fatalf("float lookup vs int level = %v", got)
	}
}

func TestMultiIndexAtValuesPos(t *testing.T) {
	mi := sampleMI(t)
	tup := mi.At(2).(Tuple)
	if tup[0] != "BR" || tup[1] != "SP" {
		t.Fatalf("At(2) = %v", tup)
	}
	if len(mi.Values()) != 4 {
		t.Fatal("Values length")
	}
	pos, ok := mi.Pos(Tuple{"AR", "CO"})
	if !ok || pos != 1 {
		t.Fatalf("Pos = %d, %v", pos, ok)
	}
	if _, ok := mi.Pos(Tuple{"AR", "SP"}); ok {
		t.Fatal("nonexistent combination must not resolve")
	}
	if _, ok := mi.Pos("AR"); ok {
		t.Fatal("bare label on 2-level index must not resolve")
	}
	if got := mi.Positions([]any{"AR", "BA"}); len(got) != 2 {
		t.Fatalf("Positions dup = %v", got)
	}
}

func TestMultiIndexNAMatching(t *testing.T) {
	mi, _ := NewMultiIndexFromArrays(
		[][]any{{"a", nil, "a"}, {"x", "y", nil}}, []string{"1", "2"})
	if got := mi.PositionsTuple([]any{nil, "y"}); len(got) != 1 || got[0] != 1 {
		t.Fatalf("NA component lookup = %v", got)
	}
	if got := mi.PositionsTuple([]any{"a", nil}); len(got) != 1 || got[0] != 2 {
		t.Fatalf("trailing NA lookup = %v", got)
	}
	if got := mi.PositionsPrefix([]any{nil}); len(got) != 1 {
		t.Fatalf("NA prefix = %v", got)
	}
}

func TestMultiIndexTakeSlice(t *testing.T) {
	mi := sampleMI(t)
	before := fmt.Sprint(mi.Tuples())

	taken := mi.Take([]int{3, 0, 0, -1}).(*MultiIndex)
	if taken.Len() != 4 {
		t.Fatalf("take len = %d", taken.Len())
	}
	if got := taken.Tuple(0); got[0] != "AR" || got[1] != "BA" {
		t.Fatalf("take tuple 0 = %v", got)
	}
	if got := taken.Tuple(3); got[0] != nil || got[1] != nil {
		t.Fatalf("negative position must be all-NA tuple: %v", got)
	}
	if names := taken.Names(); names[0] != "country" {
		t.Fatalf("take names = %v", names)
	}
	// Levels are shared, not compacted (documented).
	if len(taken.Levels()[1]) != 3 {
		t.Fatalf("take levels = %v", taken.Levels())
	}
	// Lookup on the derived index is coherent with its own codes.
	if got := taken.PositionsTuple([]any{"AR", "BA"}); len(got) != 3 {
		t.Fatalf("take lookup = %v", got)
	}

	sliced := mi.SlicePos(1, 3).(*MultiIndex)
	if sliced.Len() != 2 || sliced.Tuple(0)[1] != "CO" {
		t.Fatalf("slice = %v", sliced.Tuples())
	}

	if fmt.Sprint(mi.Tuples()) != before {
		t.Fatal("Take/SlicePos mutated the source index")
	}
}

func TestMultiIndexCloneEquals(t *testing.T) {
	mi := sampleMI(t)
	clone := mi.Clone().(*MultiIndex)
	if !mi.Equals(clone) {
		t.Fatal("clone must equal source")
	}
	other, _ := NewMultiIndexFromArrays(
		[][]any{{"AR", "AR", "BR", "AR"}, {"BA", "CO", "SP", "CO"}}, nil)
	if mi.Equals(other) {
		t.Fatal("different tuples must not be equal")
	}
	if mi.Equals(NewRangeIndex(4)) {
		t.Fatal("MultiIndex must not equal a flat index")
	}
}

func TestMultiIndexString(t *testing.T) {
	mi, _ := NewMultiIndexFromTuples([][]any{{"a", nil}}, []string{"x", "y"})
	s := mi.String()
	if !strings.Contains(s, "(a, NA)") || !strings.Contains(s, "names=[x y]") {
		t.Fatalf("String = %s", s)
	}
	// Long indexes truncate.
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = i
	}
	big, _ := NewMultiIndexFromArrays([][]any{arr}, []string{"n"})
	if !strings.Contains(big.String(), "... (100 total)") {
		t.Fatalf("long String = %.120s", big.String())
	}
}

func TestMultiIndexTupleString(t *testing.T) {
	if got := (Tuple{"AR", nil, 3}).String(); got != "(AR, NA, 3)" {
		t.Fatalf("Tuple.String = %q", got)
	}
}

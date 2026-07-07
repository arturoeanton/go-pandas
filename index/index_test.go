package index

import (
	"testing"
	"time"
)

func TestRangeIndex(t *testing.T) {
	ix := NewRangeIndex(5)
	if ix.Len() != 5 {
		t.Fatalf("Len = %d", ix.Len())
	}
	if ix.At(2) != 2 {
		t.Errorf("At(2) = %v", ix.At(2))
	}
	if p, ok := ix.Pos(3); !ok || p != 3 {
		t.Errorf("Pos(3) = %d, %v", p, ok)
	}
	if _, ok := ix.Pos(9); ok {
		t.Error("Pos(9) should miss")
	}
	pos, err := ix.Slice(1, 3)
	if err != nil || len(pos) != 3 || pos[0] != 1 || pos[2] != 3 {
		t.Errorf("Slice(1,3) = %v, %v", pos, err)
	}
}

func TestRangeIndexFrom(t *testing.T) {
	ix := RangeIndexFrom(10, 20, 5)
	if ix.Len() != 2 || ix.At(1) != 15 {
		t.Errorf("RangeIndexFrom: len=%d at1=%v", ix.Len(), ix.At(1))
	}
}

func TestStringIndex(t *testing.T) {
	ix := NewStringIndex([]string{"a", "b", "c", "b"})
	if p, ok := ix.Pos("b"); !ok || p != 1 {
		t.Errorf("Pos(b) = %d, %v", p, ok)
	}
	if got := ix.Positions("b"); len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("Positions(b) = %v", got)
	}
	pos, err := ix.Slice("a", "c")
	if err != nil || len(pos) != 3 {
		t.Errorf("Slice(a,c) = %v, %v", pos, err)
	}
}

func TestDatetimeIndex(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ix := NewDatetimeIndex([]time.Time{t0, t0.AddDate(0, 0, 1), t0.AddDate(0, 0, 2)})
	if p, ok := ix.Pos(t0.AddDate(0, 0, 1)); !ok || p != 1 {
		t.Errorf("Pos = %d, %v", p, ok)
	}
	pos, err := ix.Slice(t0, t0.AddDate(0, 0, 1))
	if err != nil || len(pos) != 2 {
		t.Errorf("datetime Slice = %v, %v", pos, err)
	}
}

func TestEquals(t *testing.T) {
	a := NewStringIndex([]string{"x", "y"})
	b := NewStringIndex([]string{"x", "y"})
	c := NewStringIndex([]string{"x", "z"})
	if !a.Equals(b) {
		t.Error("equal indexes reported unequal")
	}
	if a.Equals(c) {
		t.Error("different indexes reported equal")
	}
	r1, r2 := NewRangeIndex(3), NewRangeIndex(3)
	if !r1.Equals(r2) {
		t.Error("equal range indexes reported unequal")
	}
}

func TestSetOperations(t *testing.T) {
	a := NewStringIndex([]string{"a", "b", "c"})
	b := NewStringIndex([]string{"b", "c", "d"})
	if u := Union(a, b); u.Len() != 4 || u.At(3) != "d" {
		t.Errorf("Union = %v", u.Values())
	}
	if i := Intersection(a, b); i.Len() != 2 || i.At(0) != "b" {
		t.Errorf("Intersection = %v", i.Values())
	}
	if d := Difference(a, b); d.Len() != 1 || d.At(0) != "a" {
		t.Errorf("Difference = %v", d.Values())
	}
}

func TestAlign(t *testing.T) {
	a := NewStringIndex([]string{"a", "b"})
	b := NewStringIndex([]string{"b", "c"})
	lp, rp, result, err := Align(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 3 {
		t.Fatalf("aligned len = %d", result.Len())
	}
	// labels: a, b, c -> left: 0, 1, -1; right: -1, 0, 1
	if lp[0] != 0 || lp[1] != 1 || lp[2] != -1 {
		t.Errorf("leftPos = %v", lp)
	}
	if rp[0] != -1 || rp[1] != 0 || rp[2] != 1 {
		t.Errorf("rightPos = %v", rp)
	}
	// identical indexes align to identity
	lp2, rp2, _, _ := Align(a, a.Clone())
	if lp2[1] != 1 || rp2[1] != 1 {
		t.Errorf("identity align = %v, %v", lp2, rp2)
	}
}

func TestMultiIndex(t *testing.T) {
	mi, err := NewMultiIndexFromArrays([][]any{{"AR", "AR", "BR"}, {2023, 2024, 2023}}, []string{"country", "year"})
	if err != nil {
		t.Fatal(err)
	}
	if mi.Len() != 3 || mi.NLevels() != 2 {
		t.Fatalf("MultiIndex shape: len=%d levels=%d", mi.Len(), mi.NLevels())
	}
	// v0.8: At returns index.Tuple (underlying []any).
	tuple := mi.At(1).(Tuple)
	if tuple[0] != "AR" || tuple[1] != 2024 {
		t.Errorf("At(1) = %v", tuple)
	}
	if _, err := NewMultiIndexFromArrays([][]any{{1, 2}, {1}}, nil); err == nil {
		t.Error("ragged MultiIndex arrays should fail")
	}
}

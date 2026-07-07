package ndarray

import (
	"math"
	"testing"
)

func TestIsIn(t *testing.T) {
	a := Array([]float64{1, 2, 3, 2, math.NaN()})
	got := a.IsIn([]any{2.0, 7, math.NaN()}) // int 7 normalizes; NaN never matches
	want := []bool{false, true, false, true, false}
	for i, w := range want {
		if got.data[i] != w {
			t.Fatalf("IsIn = %v, want %v", got.data, want)
		}
	}
	s := ArrayString([]string{"a", "b", "c"}).IsIn([]any{"b", "z", 3})
	if s.data[0] || !s.data[1] || s.data[2] {
		t.Fatalf("string IsIn = %v", s.data)
	}
	b := ArrayBool([]bool{true, false}).IsIn([]any{true})
	if !b.data[0] || b.data[1] {
		t.Fatalf("bool IsIn = %v", b.data)
	}
}

func TestSearchSorted(t *testing.T) {
	a := Array([]float64{1, 2, 2, 4, 7})
	left, err := a.SearchSorted([]float64{0, 2, 3, 9}, "left")
	if err != nil {
		t.Fatal(err)
	}
	if left[0] != 0 || left[1] != 1 || left[2] != 3 || left[3] != 5 {
		t.Fatalf("left = %v", left)
	}
	right, err := a.SearchSorted([]float64{2}, "right")
	if err != nil || right[0] != 3 {
		t.Fatalf("right = %v err=%v", right, err)
	}
	if _, err := a.SearchSorted([]float64{1}, "middle"); err == nil {
		t.Fatal("bad side must error")
	}
	m, _ := Arange(6).Reshape(2, 3)
	if _, err := m.SearchSorted([]float64{1}, "left"); err == nil {
		t.Fatal("2-D must error")
	}
	if _, err := ArrayString([]string{"a"}).SearchSorted([]float64{1}, "left"); err == nil {
		t.Fatal("string array must error")
	}
}

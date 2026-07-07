package series

import (
	"math"
	"testing"
)

// TestNAComparisonSemantics documents the uniform rule: every comparison
// against NA is false — including Ne (a documented difference from
// pandas, where NaN != x is True).
func TestNAComparisonSemantics(t *testing.T) {
	s := NewSeries("v", []any{1, nil})
	if got := s.Eq(1).AsMask(); !got[0] || got[1] {
		t.Errorf("Eq = %v", got)
	}
	if got := s.Ne(1).AsMask(); got[0] || got[1] {
		t.Errorf("Ne with NA should be false (documented difference), got %v", got)
	}
	if got := s.Ne(nil).AsMask(); got[0] || got[1] {
		t.Errorf("Ne(nil) = %v", got)
	}
	if got := s.Eq(nil).AsMask(); got[0] || got[1] {
		t.Errorf("Eq(nil) = %v; NA never equals anything", got)
	}
}

func TestOpsDoNotMutateInputs(t *testing.T) {
	a := FloatSeries("a", []float64{1, 2})
	b := FloatSeries("b", []float64{10, 20})
	if _, err := a.Add(b); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Div(b); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Cumsum(); err != nil {
		t.Fatal(err)
	}
	_ = a.SortValues(false)
	_ = a.FillNA(0)
	if v, _ := a.At(0); v != 1.0 {
		t.Errorf("a mutated: %v", v)
	}
	if v, _ := b.At(1); v != 20.0 {
		t.Errorf("b mutated: %v", v)
	}
}

func TestDiffPeriodsAndPctChangeEdge(t *testing.T) {
	s := FloatSeries("v", []float64{1, 2, 4, 8})
	d2, err := s.Diff(2)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := d2.At(0); v != nil {
		t.Errorf("diff(2)[0] = %v", v)
	}
	if v, _ := d2.At(2); v != 3.0 {
		t.Errorf("diff(2)[2] = %v", v)
	}
	// negative periods compare against the next value, like pandas
	dneg, err := s.Diff(-1)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := dneg.At(0); v != -1.0 {
		t.Errorf("diff(-1)[0] = %v", v)
	}
	if v, _ := dneg.At(3); v != nil {
		t.Errorf("diff(-1) tail = %v", v)
	}
	// pct_change against a zero denominator yields +Inf, like pandas
	z := FloatSeries("v", []float64{0, 5})
	pc, err := z.PctChange(1)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := pc.At(1); !math.IsInf(v.(float64), 1) {
		t.Errorf("pct_change over zero = %v, want +Inf", v)
	}
}

func TestValueCountsWithNA(t *testing.T) {
	s := NewSeries("v", []any{"a", nil, "a", nil, nil})
	def := s.ValueCounts()
	if def.Len() != 1 {
		t.Fatalf("default value_counts should drop NA, len = %d", def.Len())
	}
	kept := s.ValueCounts(ValueCountsDropNA(false))
	if kept.Len() != 2 {
		t.Fatalf("dropna=false len = %d", kept.Len())
	}
	// NA count (3) outranks "a" (2)
	if v, _ := kept.At(0); v != 3 {
		t.Errorf("NA count = %v", v)
	}
	norm := s.ValueCounts(ValueCountsNormalize(true))
	if v, _ := norm.At(0); v != 1.0 {
		t.Errorf("normalized over non-NA = %v", v)
	}
}

// TestUniqueUnhashableValues is the regression test for the map-key panic
// on slice-valued cells (e.g. the output of Str().Split).
func TestUniqueUnhashableValues(t *testing.T) {
	split := StringSeries("s", []string{"a,b", "a,b", "c"}).Str().Split(",")
	u := split.Unique()
	if u.Len() != 2 {
		t.Errorf("unique of split cells = %d, want 2", u.Len())
	}
	if n := split.NUnique(true); n != 2 {
		t.Errorf("nunique of split cells = %d", n)
	}
	vc := split.ValueCounts()
	if v, _ := vc.At(0); v != 2 {
		t.Errorf("value_counts of split cells = %v", v)
	}
}

// TestUniqueNumericWidths: 1 (int) and 1.0 (float) count as one value,
// like pandas.
func TestUniqueNumericWidths(t *testing.T) {
	s := NewSeries("v", []any{1, 1.0, int64(1), 2})
	if n := s.NUnique(true); n != 2 {
		t.Errorf("nunique across numeric widths = %d, want 2", n)
	}
}

func TestRollingCenter(t *testing.T) {
	s := FloatSeries("v", []float64{1, 2, 3, 4, 5})
	// pandas: s.rolling(3, center=True, min_periods=1).mean()
	//         -> [1.5, 2.0, 3.0, 4.0, 4.5]
	out, err := s.Rolling(3, RollingCenter(true), RollingMinPeriods(1)).Mean()
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{1.5, 2, 3, 4, 4.5}
	for i, w := range want {
		if v, _ := out.At(i); v != w {
			t.Errorf("center[%d] = %v, want %v", i, v, w)
		}
	}
	// default min_periods: clipped windows stay NA
	strict, err := s.Rolling(3, RollingCenter(true)).Mean()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := strict.At(0); v != nil {
		t.Errorf("strict center head = %v, want NA", v)
	}
	if v, _ := strict.At(4); v != nil {
		t.Errorf("strict center tail = %v, want NA", v)
	}
	if v, _ := strict.At(2); v != 3.0 {
		t.Errorf("strict center middle = %v", v)
	}
}

func TestReindexMissingLabelsAndShiftLarge(t *testing.T) {
	s := IntSeries("v", []int{1, 2})
	shifted := s.Shift(5)
	if shifted.Count() != 0 {
		t.Errorf("shift beyond length should be all NA, count = %d", shifted.Count())
	}
}

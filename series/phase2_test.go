package series

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
)

func TestReindex(t *testing.T) {
	s := NewSeries("v", []any{10, 20}, WithIndex(index.NewStringIndex([]string{"a", "b"})))
	out, err := s.Reindex(index.NewStringIndex([]string{"b", "c", "a"}))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := out.At(0); v != 20 {
		t.Errorf("reindex[b] = %v", v)
	}
	if v, _ := out.At(1); v != nil {
		t.Errorf("reindex[c] = %v, want NA", v)
	}
	if v, _ := out.At(2); v != 10 {
		t.Errorf("reindex[a] = %v", v)
	}
}

func TestArgsortAndAliases(t *testing.T) {
	s := IntSeries("v", []int{30, 10, 20})
	arg := s.Argsort()
	if v, _ := arg.At(0); v != 1 {
		t.Errorf("argsort[0] = %v", v)
	}
	if v, _ := arg.At(2); v != 0 {
		t.Errorf("argsort[2] = %v", v)
	}
	// aliases
	if v, _ := s.ILoc(1); v != 10 {
		t.Errorf("ILoc = %v", v)
	}
	labeled := NewSeries("v", []any{1}, WithIndex(index.NewStringIndex([]string{"k"})))
	if v, _ := labeled.AtLabel("k"); v != 1 {
		t.Errorf("AtLabel = %v", v)
	}
	if v, _ := labeled.ReplaceNA(0).At(0); v != 1 {
		t.Errorf("ReplaceNA = %v", v)
	}
}

func TestRankMethods(t *testing.T) {
	s := IntSeries("v", []int{3, 1, 4, 1, 5})
	min, err := s.Rank(RankMethod("min"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := min.At(1); v != 1.0 {
		t.Errorf("rank min = %v", v)
	}
	max, _ := s.Rank(RankMethod("max"))
	if v, _ := max.At(1); v != 2.0 {
		t.Errorf("rank max = %v", v)
	}
	first, _ := s.Rank(RankMethod("first"))
	if v, _ := first.At(3); v != 2.0 {
		t.Errorf("rank first = %v", v)
	}
	desc, _ := s.Rank(RankAscending(false))
	if v, _ := desc.At(4); v != 1.0 {
		t.Errorf("rank desc = %v", v)
	}
	if _, err := s.Rank(RankMethod("wat")); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Errorf("bad rank method error = %v", err)
	}
}

func TestClipBoundsAndShift(t *testing.T) {
	s := IntSeries("v", []int{1, 5, 9})
	lo, err := s.Clip(3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := lo.At(0); v != int64(3) {
		t.Errorf("clip lower only = %v", v)
	}
	if v, _ := lo.At(2); v != int64(9) {
		t.Errorf("clip lower only kept = %v", v)
	}
	back := s.Shift(-1)
	if v, _ := back.At(0); v != 5 {
		t.Errorf("shift -1 = %v", v)
	}
	if v, _ := back.At(2); v != nil {
		t.Errorf("shift -1 tail = %v, want NA", v)
	}
}

func TestStringRegexAccessors(t *testing.T) {
	s := StringSeries("s", []string{"abc", "bcd"})
	m, err := s.Str().Match("a")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.AsMask(); !got[0] || got[1] {
		t.Errorf("Match = %v", got)
	}
	c, err := s.Str().ContainsRegex("c.$")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.AsMask(); got[0] || !got[1] {
		t.Errorf("ContainsRegex = %v", got)
	}
	r, err := s.Str().ReplaceRegex("[bc]", "-")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.At(0); v != "a--" {
		t.Errorf("ReplaceRegex = %v", v)
	}
	if _, err := s.Str().Match("("); err == nil {
		t.Error("bad regex should error")
	}
}

func TestExpandingExtras(t *testing.T) {
	s := FloatSeries("v", []float64{2, 1, 3})
	min, err := s.Expanding().Min()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := min.At(2); v != 1.0 {
		t.Errorf("expanding min = %v", v)
	}
	mx, _ := s.Expanding().Max()
	if v, _ := mx.At(2); v != 3.0 {
		t.Errorf("expanding max = %v", v)
	}
	cnt, _ := s.Expanding().Count()
	if v, _ := cnt.At(2); v != 3.0 {
		t.Errorf("expanding count = %v", v)
	}
	sd, _ := s.Expanding(2).Std()
	if v, _ := sd.At(0); v != nil {
		t.Errorf("expanding std min_periods = %v", v)
	}
	rollingVar, err := s.Rolling(2).Var()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := rollingVar.At(1); v != 0.5 {
		t.Errorf("rolling var = %v", v)
	}
	cnt2, _ := s.Rolling(2, MinPeriods(1)).Count()
	if v, _ := cnt2.At(0); v != 1.0 {
		t.Errorf("rolling count = %v", v)
	}
}

package series

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
)

func TestConstructorsAndInference(t *testing.T) {
	s := NewSeries("age", []any{10, 20, nil, 30})
	if s.Len() != 4 || s.DType() != dtype.Int {
		t.Fatalf("len=%d dtype=%v", s.Len(), s.DType())
	}
	if !s.HasNA() || s.Count() != 3 {
		t.Errorf("HasNA=%v Count=%d", s.HasNA(), s.Count())
	}
	f := SeriesOf("x", []float64{1.5, math.NaN()})
	if f.DType() != dtype.Float64 || f.Count() != 1 {
		t.Errorf("float series: dtype=%v count=%d", f.DType(), f.Count())
	}
	if StringSeries("s", []string{"a"}).DType() != dtype.String {
		t.Error("StringSeries dtype")
	}
	if BoolSeries("b", []bool{true}).DType() != dtype.Bool {
		t.Error("BoolSeries dtype")
	}
	if TimeSeries("t", []time.Time{time.Now()}).DType() != dtype.Time {
		t.Error("TimeSeries dtype")
	}
}

func TestIndexing(t *testing.T) {
	s := IntSeries("v", []int{1, 2, 3, 4, 5})
	if v, _ := s.At(2); v != 3 {
		t.Errorf("At(2) = %v", v)
	}
	if _, err := s.At(9); !errors.Is(err, errs.ErrIndexOutOfBounds) {
		t.Errorf("At(9) error = %v", err)
	}
	labeled := NewSeries("v", []any{10, 20}, WithIndex(index.NewStringIndex([]string{"a", "b"})))
	if v, err := labeled.Loc("b"); err != nil || v != 20 {
		t.Errorf("Loc(b) = %v, %v", v, err)
	}
	h := s.Head(2)
	if h.Len() != 2 {
		t.Errorf("Head len = %d", h.Len())
	}
	tl := s.Tail(2)
	if v, _ := tl.At(1); v != 5 {
		t.Errorf("Tail last = %v", v)
	}
	sl, err := s.Slice(1, 3)
	if err != nil || sl.Len() != 2 {
		t.Errorf("Slice = %v, %v", sl, err)
	}
	tk, err := s.Take([]int{4, 0})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := tk.At(0); v != 5 {
		t.Errorf("Take order: %v", v)
	}
	if err := s.Set(0, 99); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.At(0); v != 99 {
		t.Errorf("Set: %v", v)
	}
	if err := s.Set(0, nil); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.At(0); v != nil {
		t.Errorf("Set nil should mark missing, got %v", v)
	}
}

func TestMissingOps(t *testing.T) {
	s := NewSeries("x", []any{1, nil, 3})
	isna := s.IsNA().AsMask()
	if isna[0] || !isna[1] || isna[2] {
		t.Errorf("IsNA = %v", isna)
	}
	if s.NotNA().AsMask()[1] {
		t.Error("NotNA at missing should be false")
	}
	d := s.DropNA()
	if d.Len() != 2 {
		t.Errorf("DropNA len = %d", d.Len())
	}
	f := s.FillNA(0)
	if v, _ := f.At(1); v != 0 {
		t.Errorf("FillNA = %v", v)
	}
	if f.HasNA() {
		t.Error("FillNA left missing values")
	}
}

func TestAstype(t *testing.T) {
	s := NewSeries("x", []any{"1", "2", nil})
	got, err := s.Astype(dtype.Int)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := got.At(0); v != 1 {
		t.Errorf("Astype = %v", v)
	}
	if got.DType() != dtype.Int {
		t.Errorf("Astype dtype = %v", got.DType())
	}
	if _, err := NewSeries("x", []any{"abc"}).Astype(dtype.Int); err == nil {
		t.Error("Astype invalid should fail")
	}
}

func TestArithmetic(t *testing.T) {
	a := IntSeries("a", []int{1, 2, 3})
	b := IntSeries("b", []int{10, 20, 30})
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := sum.At(2); v != int64(33) {
		t.Errorf("Add = %v (%T)", v, v)
	}
	if sum.DType() != dtype.Int64 {
		t.Errorf("int add dtype = %v", sum.DType())
	}
	div, _ := a.Div(b)
	if v, _ := div.At(0); v != 0.1 {
		t.Errorf("Div = %v", v)
	}
	sc, _ := a.MulScalar(10)
	if v, _ := sc.At(1); v != int64(20) {
		t.Errorf("MulScalar = %v", v)
	}
	// NA propagation
	withNA := NewSeries("x", []any{1, nil})
	r, _ := withNA.AddScalar(1)
	if v, _ := r.At(1); v != nil {
		t.Errorf("NA + 1 = %v, want nil", v)
	}
	// length mismatch
	if _, err := a.Add(IntSeries("c", []int{1})); !errors.Is(err, errs.ErrLengthMismatch) {
		t.Errorf("length mismatch error = %v", err)
	}
	// string concatenation
	s1 := StringSeries("s", []string{"a"})
	s2 := StringSeries("s", []string{"b"})
	cat, err := s1.Add(s2)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := cat.At(0); v != "ab" {
		t.Errorf("string Add = %v", v)
	}
}

func TestComparisons(t *testing.T) {
	s := IntSeries("v", []int{1, 5, 10})
	if got := s.Gt(4).AsMask(); got[0] || !got[1] || !got[2] {
		t.Errorf("Gt = %v", got)
	}
	if got := s.Eq(5).AsMask(); !got[1] || got[0] {
		t.Errorf("Eq = %v", got)
	}
	if got := s.Between(2, 9, "both").AsMask(); got[0] || !got[1] || got[2] {
		t.Errorf("Between = %v", got)
	}
	if got := s.IsIn(1, 10).AsMask(); !got[0] || got[1] || !got[2] {
		t.Errorf("IsIn = %v", got)
	}
	// NA compares false
	withNA := NewSeries("x", []any{1, nil})
	if got := withNA.Ge(0).AsMask(); got[1] {
		t.Error("NA comparison should be false")
	}
	strs := StringSeries("s", []string{"apple", "banana"})
	if got := strs.Lt("b").AsMask(); !got[0] || got[1] {
		t.Errorf("string Lt = %v", got)
	}
}

func TestReductions(t *testing.T) {
	s := NewSeries("v", []any{1.0, 2.0, 3.0, 4.0, nil})
	if sum, _ := s.Sum(); sum != 10 {
		t.Errorf("Sum = %v", sum)
	}
	if mean, _ := s.Mean(); mean != 2.5 {
		t.Errorf("Mean = %v", mean)
	}
	if med, _ := s.Median(); med != 2.5 {
		t.Errorf("Median = %v", med)
	}
	if q, _ := s.Quantile(0.25); q != 1.75 {
		t.Errorf("Quantile(0.25) = %v", q)
	}
	if mn, _ := s.Min(); mn != 1.0 {
		t.Errorf("Min = %v", mn)
	}
	if mx, _ := s.Max(); mx != 4.0 {
		t.Errorf("Max = %v", mx)
	}
	if v, _ := s.Var(); !almost(v, 5.0/3.0) {
		t.Errorf("Var = %v", v)
	}
	if sd, _ := s.Std(); !almost(sd, math.Sqrt(5.0/3.0)) {
		t.Errorf("Std = %v", sd)
	}
	// skipna=false makes reductions NaN
	if sum, _ := s.Sum(SkipNA(false)); !math.IsNaN(sum) {
		t.Errorf("Sum(skipna=false) = %v", sum)
	}
	// string min/max
	strs := StringSeries("s", []string{"b", "a", "c"})
	if mn, _ := strs.Min(); mn != "a" {
		t.Errorf("string Min = %v", mn)
	}
}

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSortUniqueValueCounts(t *testing.T) {
	s := NewSeries("v", []any{3, 1, nil, 2})
	sorted := s.SortValues(true)
	if v, _ := sorted.At(0); v != 1 {
		t.Errorf("sorted first = %v", v)
	}
	// NA goes last
	if v, _ := sorted.At(3); v != nil {
		t.Errorf("sorted last = %v, want NA", v)
	}
	desc := s.SortValues(false)
	if v, _ := desc.At(0); v != 3 {
		t.Errorf("desc first = %v", v)
	}
	u := NewSeries("v", []any{1, 1, 2, nil, nil}).Unique()
	if u.Len() != 3 {
		t.Errorf("Unique len = %d", u.Len())
	}
	if n := NewSeries("v", []any{1, 1, 2, nil}).NUnique(true); n != 2 {
		t.Errorf("NUnique(drop) = %d", n)
	}
	if n := NewSeries("v", []any{1, 1, 2, nil}).NUnique(false); n != 3 {
		t.Errorf("NUnique(keep) = %d", n)
	}
	vc := StringSeries("fruit", []string{"a", "b", "a"}).ValueCounts()
	if vc.Len() != 2 {
		t.Fatalf("ValueCounts len = %d", vc.Len())
	}
	if v, _ := vc.At(0); v != 2 {
		t.Errorf("top count = %v", v)
	}
}

func TestStringAccessor(t *testing.T) {
	s := StringSeries("s", []string{"Hello", "World"})
	if got := s.Str().Contains("ell").AsMask(); !got[0] || got[1] {
		t.Errorf("Contains = %v", got)
	}
	if v, _ := s.Str().Upper().At(0); v != "HELLO" {
		t.Errorf("Upper = %v", v)
	}
	if v, _ := s.Str().Lower().At(1); v != "world" {
		t.Errorf("Lower = %v", v)
	}
	if v, _ := s.Str().Len().At(0); v != 5 {
		t.Errorf("Len = %v", v)
	}
	if v, _ := StringSeries("s", []string{" x "}).Str().Strip().At(0); v != "x" {
		t.Errorf("Strip = %v", v)
	}
	if v, _ := s.Str().Replace("l", "L").At(0); v != "HeLLo" {
		t.Errorf("Replace = %v", v)
	}
	if got := s.Str().HasPrefix("He").AsMask(); !got[0] || got[1] {
		t.Errorf("HasPrefix = %v", got)
	}
}

func TestDatetimeAccessor(t *testing.T) {
	t0 := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC) // a Friday
	s := TimeSeries("t", []time.Time{t0})
	if v, _ := s.Dt().Year().At(0); v != 2024 {
		t.Errorf("Year = %v", v)
	}
	if v, _ := s.Dt().Month().At(0); v != 3 {
		t.Errorf("Month = %v", v)
	}
	if v, _ := s.Dt().Day().At(0); v != 15 {
		t.Errorf("Day = %v", v)
	}
	if v, _ := s.Dt().Hour().At(0); v != 10 {
		t.Errorf("Hour = %v", v)
	}
	if v, _ := s.Dt().Weekday().At(0); v != 4 {
		t.Errorf("Weekday = %v, want 4 (Friday, Monday=0)", v)
	}
}

func TestRolling(t *testing.T) {
	s := FloatSeries("v", []float64{1, 2, 3, 4, 5})
	mean, err := s.Rolling(3).Mean()
	if err != nil {
		t.Fatal(err)
	}
	// first two windows incomplete -> NA
	if v, _ := mean.At(0); v != nil {
		t.Errorf("rolling[0] = %v, want NA", v)
	}
	if v, _ := mean.At(2); v != 2.0 {
		t.Errorf("rolling[2] = %v", v)
	}
	if v, _ := mean.At(4); v != 4.0 {
		t.Errorf("rolling[4] = %v", v)
	}
	sum, _ := s.Rolling(2).Sum()
	if v, _ := sum.At(1); v != 3.0 {
		t.Errorf("rolling sum = %v", v)
	}
	mn, _ := s.Rolling(2, RollingMinPeriods(1)).Min()
	if v, _ := mn.At(0); v != 1.0 {
		t.Errorf("min_periods=1 first window = %v", v)
	}
	exp, err := s.Expanding().Mean()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := exp.At(4); v != 3.0 {
		t.Errorf("expanding mean = %v", v)
	}
}

func TestToNDArrayAndDescribe(t *testing.T) {
	s := NewSeries("v", []any{1.0, nil, 3.0})
	arr, err := s.ToNDArray()
	if err != nil {
		t.Fatal(err)
	}
	data := arr.Data()
	if data[0] != 1 || !math.IsNaN(data[1]) || data[2] != 3 {
		t.Errorf("ToNDArray = %v", data)
	}
	d, err := FloatSeries("v", []float64{1, 2, 3, 4}).Describe()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := d.Loc("count"); v != 4.0 {
		t.Errorf("describe count = %v", v)
	}
	if v, _ := d.Loc("50%"); v != 2.5 {
		t.Errorf("describe median = %v", v)
	}
}

func TestApplyAndCopy(t *testing.T) {
	s := IntSeries("v", []int{1, 2})
	doubled := s.Apply(func(v any) any { return v.(int) * 2 })
	if v, _ := doubled.At(1); v != 4 {
		t.Errorf("Apply = %v", v)
	}
	c := s.Copy()
	_ = c.Set(0, 99)
	if v, _ := s.At(0); v == 99 {
		t.Error("Copy should be independent")
	}
	r := s.Rename("w")
	if r.Name() != "w" || s.Name() != "v" {
		t.Error("Rename should not mutate the source")
	}
}

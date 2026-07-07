package column

import (
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
)

func TestFromTypedStoresRealSlices(t *testing.T) {
	cases := []struct {
		values any
		dt     dtype.DType
	}{
		{[]bool{true, false}, dtype.Bool},
		{[]int{1, 2}, dtype.Int},
		{[]int64{1, 2}, dtype.Int64},
		{[]float32{1.5}, dtype.Float32},
		{[]float64{1.5}, dtype.Float64},
		{[]string{"a"}, dtype.String},
		{[]time.Time{time.Now()}, dtype.Time},
	}
	for _, tc := range cases {
		c := FromTyped(tc.values)
		if c == nil {
			t.Fatalf("FromTyped(%T) = nil", tc.values)
		}
		if c.DType() != tc.dt {
			t.Errorf("FromTyped(%T) dtype = %v, want %v", tc.values, c.DType(), tc.dt)
		}
		if IsObjectBacked(c) {
			t.Errorf("FromTyped(%T) is object-backed", tc.values)
		}
		if StorageDType(c) != tc.dt {
			t.Errorf("StorageDType = %v", StorageDType(c))
		}
	}
	if FromTyped([]complex128{1}) != nil {
		t.Error("unsupported slice should return nil")
	}
}

func TestNaNBecomesMask(t *testing.T) {
	nan := 0.0
	nan = nan / nan
	c := FromTyped([]float64{1, nan, 3})
	if !c.IsNA(1) || c.IsNA(0) {
		t.Errorf("NaN mask: %v %v", c.IsNA(0), c.IsNA(1))
	}
	if c.Value(1) != nil {
		t.Errorf("masked Value = %v", c.Value(1))
	}
}

func TestInferAndFallback(t *testing.T) {
	ints := Infer([]any{1, nil, 3})
	if ints.DType() != dtype.Int || IsObjectBacked(ints) {
		t.Errorf("int inference: dtype=%v object=%v", ints.DType(), IsObjectBacked(ints))
	}
	if !ints.IsNA(1) {
		t.Error("nil should mask")
	}
	promoted := Infer([]any{1, nil, 2.5})
	if promoted.DType() != dtype.Float64 || IsObjectBacked(promoted) {
		t.Errorf("mixed numeric should be Float64Column, got %v object=%v", promoted.DType(), IsObjectBacked(promoted))
	}
	if v := promoted.Value(0); v != 1.0 {
		t.Errorf("promoted int value = %v (%T)", v, v)
	}
	mixed := Infer([]any{1, "a"})
	if !IsObjectBacked(mixed) || mixed.DType() != dtype.Object {
		t.Errorf("mixed incompatible should be object, got %v", mixed.DType())
	}
	if StorageDType(mixed) != dtype.Object {
		t.Error("object storage dtype")
	}
}

func TestSetTakeSliceCopy(t *testing.T) {
	c := FromTyped([]int{10, 20, 30})
	if err := c.SetValue(1, nil); err != nil {
		t.Fatal(err)
	}
	if !c.IsNA(1) {
		t.Error("SetValue(nil) should mask")
	}
	if err := c.SetValue(1, 25); err != nil {
		t.Fatal(err)
	}
	if c.Value(1) != 25 || c.IsNA(1) {
		t.Errorf("SetValue = %v", c.Value(1))
	}
	if err := c.SetValue(0, "nope"); err == nil {
		t.Error("incompatible SetValue should error")
	}
	taken, err := c.Take([]int{2, -1, 0})
	if err != nil {
		t.Fatal(err)
	}
	if taken.Value(0) != 30 || !taken.IsNA(1) || taken.Value(2) != 10 {
		t.Errorf("take = %v", taken.Values())
	}
	sl, err := c.Slice(1, 3)
	if err != nil || sl.Len() != 2 {
		t.Fatalf("slice = %v, %v", sl, err)
	}
	cp := c.Copy()
	_ = cp.SetValue(0, 99)
	if c.Value(0) == 99 {
		t.Error("Copy shares storage")
	}
}

func TestFloat64sFastPath(t *testing.T) {
	ints := FromTyped([]int{1, 2, 3})
	fs, mask, ok := ints.Float64s()
	if !ok || fs[2] != 3 || mask[0] {
		t.Errorf("int Float64s = %v %v %v", fs, mask, ok)
	}
	strs := FromTyped([]string{"a"})
	if _, _, ok := strs.Float64s(); ok {
		t.Error("string Float64s should report not-ok")
	}
	bools := FromTyped([]bool{true, false})
	fs, _, ok = bools.Float64s()
	if !ok || fs[0] != 1 || fs[1] != 0 {
		t.Errorf("bool Float64s = %v", fs)
	}
	objNumeric := FromAny([]any{1, 2.5}, dtype.Float64)
	fs, _, ok = objNumeric.Float64s()
	if !ok || fs[1] != 2.5 {
		t.Errorf("converted Float64s = %v %v", fs, ok)
	}
}

func TestTimeColumnNaT(t *testing.T) {
	c := FromAny([]any{time.Now(), nil}, dtype.Time)
	if c.DType() != dtype.Time {
		t.Fatalf("dtype = %v", c.DType())
	}
	if _, isNaT := c.Value(1).(dtype.NaTMarker); !isNaT {
		t.Errorf("masked time Value = %T, want NaT marker", c.Value(1))
	}
	if vals := c.Values(); vals[1] != nil {
		t.Errorf("Values() should box missing as nil, got %v", vals[1])
	}
}

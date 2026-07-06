package dtype

import (
	"math"
	"testing"
	"time"
)

func TestInferDType(t *testing.T) {
	cases := []struct {
		name   string
		values []any
		want   DType
	}{
		{"bool", []any{true, false}, Bool},
		{"int", []any{1, 2, 3}, Int},
		{"int64", []any{int64(1), int64(2)}, Int64},
		{"float64", []any{1.5, 2.5}, Float64},
		{"string", []any{"a", "b"}, String},
		{"time", []any{time.Now(), time.Now()}, Time},
		{"mixed int float", []any{1, 2.5}, Float64},
		{"mixed int int64", []any{1, int64(2)}, Int64},
		{"mixed incompatible", []any{1, "a"}, Object},
		{"all NA", []any{nil, nil}, Object},
		{"int with NA", []any{1, nil, 3}, Int},
		{"float with NaN", []any{1.5, math.NaN()}, Float64},
	}
	for _, tc := range cases {
		if got := InferDType(tc.values); got != tc.want {
			t.Errorf("%s: InferDType = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestInferDTypeStrict(t *testing.T) {
	if got := InferDTypeStrict([]any{1, 2.5}); got != Object {
		t.Errorf("strict mixed int/float = %v, want Object", got)
	}
	if got := InferDTypeStrict([]any{1, 2}); got != Int {
		t.Errorf("strict ints = %v, want Int", got)
	}
}

func TestMissing(t *testing.T) {
	if !IsNA(nil) {
		t.Error("IsNA(nil) = false")
	}
	if !IsNA(math.NaN()) {
		t.Error("IsNA(NaN) = false")
	}
	if !IsNA(NA()) {
		t.Error("IsNA(NA()) = false")
	}
	if !IsNA(NaT()) {
		t.Error("IsNA(NaT()) = false")
	}
	if IsNA("") {
		t.Error("IsNA(\"\") = true; empty string must not be missing")
	}
	if IsNA(0) {
		t.Error("IsNA(0) = true")
	}
	if !NotNA(1) || !NotNull(1) || IsNull(1) {
		t.Error("NotNA/IsNull aliases broken")
	}
}

func TestCastValue(t *testing.T) {
	if v, err := CastValue("42", Int); err != nil || v != 42 {
		t.Errorf("cast \"42\" to Int = %v, %v", v, err)
	}
	if v, err := CastValue("2.5", Float64); err != nil || v != 2.5 {
		t.Errorf("cast \"2.5\" to Float64 = %v, %v", v, err)
	}
	if _, err := CastValue("abc", Int); err == nil {
		t.Error("cast \"abc\" to Int should fail")
	}
	if v, err := CastValue(3, Float64); err != nil || v != 3.0 {
		t.Errorf("cast 3 to Float64 = %v, %v", v, err)
	}
	if v, err := CastValue(1.0, Bool); err != nil || v != true {
		t.Errorf("cast 1.0 to Bool = %v, %v", v, err)
	}
	if v, err := CastValue(7, String); err != nil || v != "7" {
		t.Errorf("cast 7 to String = %v, %v", v, err)
	}
	if v, err := CastValue(nil, Int); err != nil || v != nil {
		t.Errorf("cast nil should pass through, got %v, %v", v, err)
	}
	if v, err := CastValue("2024-01-02", Time); err != nil {
		t.Errorf("cast date string to Time: %v", err)
	} else if v.(time.Time).Year() != 2024 {
		t.Errorf("cast date string year = %v", v)
	}
}

func TestCastSlice(t *testing.T) {
	out, err := CastSlice([]any{"1", "2", nil}, Int)
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != 1 || out[1] != 2 || out[2] != nil {
		t.Errorf("CastSlice = %v", out)
	}
	if _, err := CastSlice([]any{"1", "x"}, Int); err == nil {
		t.Error("CastSlice with invalid value should fail")
	}
}

func TestPromote(t *testing.T) {
	cases := []struct{ a, b, want DType }{
		{Int, Float64, Float64},
		{Int, Int64, Int64},
		{Int8, Int16, Int16},
		{Bool, Int, Int},
		{Float32, Float32, Float32},
		{Float32, Int64, Float64},
		{String, Int, Object},
		{Object, Int, Object},
		{Int, Int, Int},
		{UInt8, UInt16, UInt16},
		{Int32, UInt32, Int64},
	}
	for _, tc := range cases {
		if got := Promote(tc.a, tc.b); got != tc.want {
			t.Errorf("Promote(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCanCast(t *testing.T) {
	if !CanCast(Int, Float64) || !CanCast(String, Int) || !CanCast(Float64, String) {
		t.Error("CanCast common conversions should be allowed")
	}
	if CanCast(Time, Int) {
		t.Error("CanCast(Time, Int) should be false")
	}
}

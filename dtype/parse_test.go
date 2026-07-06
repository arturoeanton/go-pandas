package dtype

import (
	"errors"
	"testing"

	"github.com/arturoeanton/go-pandas/errs"
)

func TestParseDType(t *testing.T) {
	cases := map[string]DType{
		"int64":           Int64,
		"Int64":           Int64,
		"float64":         Float64,
		"string":          String,
		"bool":            Bool,
		"boolean":         Bool,
		"datetime64[ns]":  Time,
		"datetime64":      Time,
		"timedelta64[ns]": Timedelta,
		"category":        Category,
		"object":          Object,
		"number":          Number,
	}
	for name, want := range cases {
		got, err := ParseDType(name)
		if err != nil || got != want {
			t.Errorf("ParseDType(%q) = %v, %v; want %v", name, got, err, want)
		}
	}
	if _, err := ParseDType("wat"); !errors.Is(err, errs.ErrInvalidDType) {
		t.Errorf("ParseDType(wat) error = %v", err)
	}
}

func TestKindAndMatches(t *testing.T) {
	if Int64.Kind() != KindSignedInt || UInt8.Kind() != KindUnsignedInt {
		t.Error("integer kinds")
	}
	if Float32.Kind() != KindFloat || String.Kind() != KindString || Time.Kind() != KindDatetime {
		t.Error("float/string/datetime kinds")
	}
	if !Matches(Number, Int) || !Matches(Number, Float64) || !Matches(Number, Bool) {
		t.Error("Number should match numeric dtypes")
	}
	if Matches(Number, String) {
		t.Error("Number must not match String")
	}
	if !Matches(Int64, Int64) || Matches(Int64, Int32) {
		t.Error("exact dtype matching")
	}
}

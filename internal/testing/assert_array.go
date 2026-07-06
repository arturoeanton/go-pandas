package testutil

import (
	"math"
	"testing"

	"github.com/arturoeanton/go-pandas/ndarray"
)

// AssertArrayAllClose compares an NDArray against a golden array: exact
// shape, tolerant elements, NaN positions marked by nan_at.
func AssertArrayAllClose(t *testing.T, got *ndarray.NDArray, expected GoldenExpected) {
	t.Helper()
	shape := got.Shape()
	if len(shape) != len(expected.Shape) {
		t.Fatalf("shape = %v, want %v", shape, expected.Shape)
	}
	for i := range shape {
		if shape[i] != expected.Shape[i] {
			t.Fatalf("shape = %v, want %v", shape, expected.Shape)
		}
	}
	want := append([]float64(nil), expected.Data...)
	for _, i := range expected.NaNAt {
		if i >= 0 && i < len(want) {
			want[i] = math.NaN()
		}
	}
	data := got.Data()
	if len(data) != len(want) {
		t.Fatalf("data length = %d, want %d", len(data), len(want))
	}
	for i := range want {
		if !AllClose(data[i], want[i]) {
			t.Fatalf("data[%d] = %v, want %v (full: %v)", i, data[i], want[i], data)
		}
	}
}

// AssertBoolArrayEqual compares a BoolArray against golden booleans.
func AssertBoolArrayEqual(t *testing.T, got *ndarray.BoolArray, expected GoldenExpected) {
	t.Helper()
	if expected.Shape != nil {
		shape := got.Shape()
		if len(shape) != len(expected.Shape) {
			t.Fatalf("shape = %v, want %v", shape, expected.Shape)
		}
		for i := range shape {
			if shape[i] != expected.Shape[i] {
				t.Fatalf("shape = %v, want %v", shape, expected.Shape)
			}
		}
	}
	data := got.Data()
	if len(data) != len(expected.BoolData) {
		t.Fatalf("bool length = %d, want %d", len(data), len(expected.BoolData))
	}
	for i := range data {
		if data[i] != expected.BoolData[i] {
			t.Fatalf("bool[%d] = %v, want %v (full: %v)", i, data[i], expected.BoolData[i], data)
		}
	}
}

package testutil

import (
	"fmt"
	"testing"

	"github.com/arturoeanton/go-pandas/series"
)

// AssertSeriesEqual compares a Series against golden values (and index
// labels, when present).
func AssertSeriesEqual(t *testing.T, got *series.Series, expected GoldenExpected) {
	t.Helper()
	values := got.Values()
	if len(values) != len(expected.Values) {
		t.Fatalf("series length = %d, want %d (values: %v)", len(values), len(expected.Values), values)
	}
	for i := range values {
		if !CellEqual(values[i], expected.Values[i]) {
			t.Fatalf("series[%d] = %v (%T), want %v", i, values[i], values[i], expected.Values[i])
		}
	}
	if expected.Index != nil {
		idx := got.Index()
		if idx.Len() != len(expected.Index) {
			t.Fatalf("series index length = %d, want %d", idx.Len(), len(expected.Index))
		}
		for i, want := range expected.Index {
			if fmt.Sprint(idx.At(i)) != want {
				t.Fatalf("series index[%d] = %v, want %v", i, idx.At(i), want)
			}
		}
	}
}

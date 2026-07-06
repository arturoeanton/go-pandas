package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
)

// CellEqual compares a go-pandas cell against a JSON-decoded expected
// value: numbers with tolerance across int/float widths, NA against null,
// times against their pandas string form.
func CellEqual(got, want any) bool {
	if want == nil || dtype.IsNA(got) {
		return want == nil && dtype.IsNA(got)
	}
	if wf, ok := want.(float64); ok {
		gf, ok := dtype.AsFloat(got)
		if !ok {
			return false
		}
		if _, isBool := got.(bool); isBool {
			return false // bool 1/0 must not match numbers
		}
		return AllClose(gf, wf)
	}
	if wb, ok := want.(bool); ok {
		gb, ok := got.(bool)
		return ok && gb == wb
	}
	if ws, ok := want.(string); ok {
		switch g := got.(type) {
		case string:
			return g == ws
		case time.Time:
			return g.Format("2006-01-02 15:04:05") == ws || g.Format("2006-01-02") == ws
		}
		return false
	}
	return got == want
}

// AssertFrameEqual compares a DataFrame against a golden frame: exact
// column order, exact row order, tolerant cells and optional index labels.
func AssertFrameEqual(t *testing.T, got *dataframe.DataFrame, expected GoldenExpected) {
	t.Helper()
	cols := got.Columns()
	if len(cols) != len(expected.Columns) {
		t.Fatalf("columns = %v, want %v", cols, expected.Columns)
	}
	for i := range cols {
		if cols[i] != expected.Columns[i] {
			t.Fatalf("columns = %v, want %v", cols, expected.Columns)
		}
	}
	rows := got.ToRows()
	if len(rows) != len(expected.Rows) {
		t.Fatalf("row count = %d, want %d (rows: %v)", len(rows), len(expected.Rows), rows)
	}
	for i := range rows {
		if len(expected.Rows[i]) != len(cols) {
			t.Fatalf("golden row %d has %d cells for %d columns", i, len(expected.Rows[i]), len(cols))
		}
		for j := range expected.Rows[i] {
			if !CellEqual(rows[i][j], expected.Rows[i][j]) {
				t.Fatalf("cell [%d][%s] = %v (%T), want %v", i, cols[j], rows[i][j], rows[i][j], expected.Rows[i][j])
			}
		}
	}
	if expected.Index != nil {
		idx := got.Index()
		if idx.Len() != len(expected.Index) {
			t.Fatalf("index length = %d, want %d", idx.Len(), len(expected.Index))
		}
		for i, want := range expected.Index {
			if fmt.Sprint(idx.At(i)) != want {
				t.Fatalf("index[%d] = %v, want %v", i, idx.At(i), want)
			}
		}
	}
}

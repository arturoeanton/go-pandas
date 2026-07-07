package io

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TestJSONColumnsOrientRowOrder is the regression test for lexicographic
// row-key sorting ("10" < "2") scrambling frames with 10+ rows.
func TestJSONColumnsOrientRowOrder(t *testing.T) {
	table := &Table{Columns: []string{"v"}}
	for i := 0; i < 12; i++ {
		table.Rows = append(table.Rows, []any{i * 10})
	}
	var buf bytes.Buffer
	if err := WriteJSONTable(&buf, table, WithOrient("columns")); err != nil {
		t.Fatal(err)
	}
	back, err := ReadJSONTableReader(&buf, WithOrient("columns"))
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rows) != 12 {
		t.Fatalf("rows = %d", len(back.Rows))
	}
	for i, row := range back.Rows {
		if row[0] != i*10 {
			t.Fatalf("row %d = %v, want %d (numeric key ordering)", i, row[0], i*10)
		}
	}
}

// TestCSVEmptyStringVsNA: an empty cell is NA with the default options,
// but stays an empty string when custom NA values exclude it.
func TestCSVEmptyStringVsNA(t *testing.T) {
	csv := "a,b\n1,\n2,x\n"
	def, err := ReadCSVTableReader(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if def.Rows[0][1] != nil {
		t.Errorf("default: empty cell = %v, want NA", def.Rows[0][1])
	}
	custom, err := ReadCSVTableReader(strings.NewReader(csv), WithNAValues("na-token"))
	if err != nil {
		t.Fatal(err)
	}
	if custom.Rows[0][1] != "" {
		t.Errorf("custom NA set: empty cell = %q (%T), want empty string", custom.Rows[0][1], custom.Rows[0][1])
	}
}

// TestGoldenScriptsDeterministic ensures repeated CSV parses of the same
// input produce identical tables (guards against map iteration leaking
// into results).
func TestCSVDeterministic(t *testing.T) {
	csv := "b,a\n1,2\n3,4\n"
	first, err := ReadCSVTableReader(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := ReadCSVTableReader(strings.NewReader(csv))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(again.Columns) != fmt.Sprint(first.Columns) ||
			fmt.Sprint(again.Rows) != fmt.Sprint(first.Rows) {
			t.Fatal("CSV parsing is not deterministic")
		}
	}
}

package io

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadCSVUseColsAndKeepDefaultNA(t *testing.T) {
	table, err := ReadCSVTableReader(strings.NewReader("a,b,c\n1,2,3\n4,5,6\n"),
		WithUseCols("a", "c"))
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 2 || table.Columns[1] != "c" {
		t.Fatalf("usecols columns = %v", table.Columns)
	}
	if table.Rows[1][1] != 6 {
		t.Errorf("usecols rows = %v", table.Rows)
	}
	if _, err := ReadCSVTableReader(strings.NewReader("a,b\n1,2\n"), WithUseCols("z")); err == nil {
		t.Error("unknown usecols should error")
	}
	// custom NA values replace the defaults unless KeepDefaultNA is set
	custom, err := ReadCSVTableReader(strings.NewReader("a\nmissing\nNA\n"),
		WithNAValues("missing"))
	if err != nil {
		t.Fatal(err)
	}
	if custom.Rows[0][0] != nil || custom.Rows[1][0] != "NA" {
		t.Errorf("custom NA = %v", custom.Rows)
	}
	both, err := ReadCSVTableReader(strings.NewReader("a\nmissing\nNA\n"),
		WithNAValues("missing"), WithKeepDefaultNA(true))
	if err != nil {
		t.Fatal(err)
	}
	if both.Rows[0][0] != nil || both.Rows[1][0] != nil {
		t.Errorf("keep default NA = %v", both.Rows)
	}
}

func TestJSONSplitAndColumnsOrient(t *testing.T) {
	table := &Table{
		Columns: []string{"a", "b"},
		Rows:    [][]any{{1, "x"}, {2, "y"}},
	}
	var buf bytes.Buffer
	if err := WriteJSONTable(&buf, table, WithOrient("split")); err != nil {
		t.Fatal(err)
	}
	back, err := ReadJSONTableReader(&buf, WithOrient("split"))
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rows) != 2 || back.Rows[1][1] != "y" {
		t.Errorf("split round trip = %v", back.Rows)
	}
	buf.Reset()
	if err := WriteJSONTable(&buf, table, WithOrient("columns")); err != nil {
		t.Fatal(err)
	}
	back, err = ReadJSONTableReader(&buf, WithOrient("columns"))
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rows) != 2 || back.Rows[0][0] != 1 {
		t.Errorf("columns round trip = %v", back.Rows)
	}
	if _, err := ReadJSONTableReader(strings.NewReader("{}"), WithOrient("nope")); err == nil {
		t.Error("unknown orient should error")
	}
}

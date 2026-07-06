package io

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestReadCSV(t *testing.T) {
	csv := "name,age,score\nAna,30,9.5\nLuis,40,8.0\n"
	table, err := ReadCSVTableReader(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 3 || table.Columns[0] != "name" {
		t.Fatalf("columns = %v", table.Columns)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows = %d", len(table.Rows))
	}
	if table.Rows[0][1] != 30 {
		t.Errorf("age inferred = %v (%T)", table.Rows[0][1], table.Rows[0][1])
	}
	if table.Rows[0][2] != 9.5 {
		t.Errorf("score inferred = %v", table.Rows[0][2])
	}
}

func TestReadCSVNoHeader(t *testing.T) {
	table, err := ReadCSVTableReader(strings.NewReader("1,2\n3,4\n"), WithHeader(false))
	if err != nil {
		t.Fatal(err)
	}
	if table.Columns[0] != "column_0" || len(table.Rows) != 2 {
		t.Errorf("no header: %v, %d rows", table.Columns, len(table.Rows))
	}
}

func TestReadCSVOptions(t *testing.T) {
	// custom comma + NA values + no inference
	table, err := ReadCSVTableReader(strings.NewReader("a;b\nx;NA\ny;3\n"),
		WithComma(';'))
	if err != nil {
		t.Fatal(err)
	}
	if table.Rows[0][1] != nil {
		t.Errorf("NA cell = %v", table.Rows[0][1])
	}
	if table.Rows[1][1] != 3 {
		t.Errorf("inferred = %v", table.Rows[1][1])
	}
	raw, err := ReadCSVTableReader(strings.NewReader("a\n42\n"), WithInferTypes(false))
	if err != nil {
		t.Fatal(err)
	}
	if raw.Rows[0][0] != "42" {
		t.Errorf("no inference = %v (%T)", raw.Rows[0][0], raw.Rows[0][0])
	}
	limited, err := ReadCSVTableReader(strings.NewReader("a\n1\n2\n3\n"), WithLimit(2))
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.Rows) != 2 {
		t.Errorf("limit rows = %d", len(limited.Rows))
	}
}

func TestReadCSVParseDates(t *testing.T) {
	table, err := ReadCSVTableReader(strings.NewReader("day,v\n2024-01-02,1\n"),
		WithParseDates("day"))
	if err != nil {
		t.Fatal(err)
	}
	tm, ok := table.Rows[0][0].(time.Time)
	if !ok || tm.Year() != 2024 {
		t.Errorf("parsed date = %v", table.Rows[0][0])
	}
}

func TestWriteCSV(t *testing.T) {
	table := &Table{
		Columns: []string{"a", "b"},
		Rows:    [][]any{{1, "x"}, {nil, "y"}},
	}
	var buf bytes.Buffer
	if err := WriteCSVTable(&buf, table); err != nil {
		t.Fatal(err)
	}
	want := "a,b\n1,x\n,y\n"
	if buf.String() != want {
		t.Errorf("csv = %q, want %q", buf.String(), want)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	jsonStr := `[{"a": 1, "b": "x"}, {"a": 2, "b": "y"}]`
	table, err := ReadJSONTableReader(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 2 || table.Rows[0][0] != 1 {
		t.Fatalf("json read: %v", table.Rows)
	}
	var buf bytes.Buffer
	if err := WriteJSONTable(&buf, table); err != nil {
		t.Fatal(err)
	}
	table2, err := ReadJSONTableReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(table2.Rows) != 2 || table2.Rows[1][1] != "y" {
		t.Errorf("json round trip: %v", table2.Rows)
	}
}

func TestJSONValues(t *testing.T) {
	table, err := ReadJSONTableReader(strings.NewReader(`[[1, "a"], [2, "b"]]`), WithOrient("values"))
	if err != nil {
		t.Fatal(err)
	}
	if table.Columns[0] != "column_0" || table.Rows[1][0] != 2 {
		t.Errorf("values orient: %v %v", table.Columns, table.Rows)
	}
}

func TestNDJSONRoundTrip(t *testing.T) {
	ndjson := "{\"a\": 1}\n{\"a\": 2}\n"
	table, err := ReadNDJSONTableReader(strings.NewReader(ndjson))
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 2 || table.Rows[1][0] != 2 {
		t.Fatalf("ndjson read: %v", table.Rows)
	}
	var buf bytes.Buffer
	if err := WriteNDJSONTable(&buf, table); err != nil {
		t.Fatal(err)
	}
	if strings.Count(buf.String(), "\n") != 2 {
		t.Errorf("ndjson write: %q", buf.String())
	}
}

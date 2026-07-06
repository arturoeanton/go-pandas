package dataframe

import (
	"path/filepath"
	"testing"
)

func TestCSVRoundTrip(t *testing.T) {
	df := sampleFrame(t)
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := df.ToCSV(path); err != nil {
		t.Fatal(err)
	}
	back, err := ReadCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Len() != 3 {
		t.Fatalf("csv round trip len = %d", back.Len())
	}
	if v := colValues(t, back, "age"); v[1] != 40 {
		t.Errorf("csv age = %v (%T)", v[1], v[1])
	}
	if v := colValues(t, back, "salary"); v[2] != 1500.0 {
		t.Errorf("csv salary = %v", v)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	df := sampleFrame(t)
	path := filepath.Join(t.TempDir(), "out.json")
	if err := df.ToJSON(path); err != nil {
		t.Fatal(err)
	}
	back, err := ReadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Len() != 3 {
		t.Fatalf("json round trip len = %d", back.Len())
	}
	if v := colValues(t, back, "name"); v[0] != "Ana" {
		t.Errorf("json name = %v", v)
	}
}

func TestNDJSONRoundTrip(t *testing.T) {
	df := sampleFrame(t)
	path := filepath.Join(t.TempDir(), "out.ndjson")
	if err := df.ToNDJSON(path); err != nil {
		t.Fatal(err)
	}
	back, err := ReadNDJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Len() != 3 {
		t.Fatalf("ndjson round trip len = %d", back.Len())
	}
	if v := colValues(t, back, "country"); v[2] != "BR" {
		t.Errorf("ndjson country = %v", v)
	}
}

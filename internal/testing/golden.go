package testutil

import (
	"encoding/json"
	"os"
	"testing"
)

// GoldenFile is one golden suite generated from real pandas/NumPy.
type GoldenFile struct {
	Suite         string       `json:"suite"`
	PandasVersion string       `json:"pandas_version,omitempty"`
	NumpyVersion  string       `json:"numpy_version,omitempty"`
	Cases         []GoldenCase `json:"cases"`
}

// GoldenCase is one recorded behavior: the Python operation and its
// serialized result.
type GoldenCase struct {
	Name      string          `json:"name"`
	Operation string          `json:"operation,omitempty"`
	Expected  json.RawMessage `json:"expected"`
}

// GoldenExpected is the union of every expected-result shape the golden
// files use. Exactly one family of fields is populated per case.
type GoldenExpected struct {
	// Frame result.
	Columns []string `json:"columns,omitempty"`
	Rows    [][]any  `json:"rows,omitempty"`
	// Series result.
	Values []any `json:"values,omitempty"`
	// Shared row/series labels (stringified).
	Index []string `json:"index,omitempty"`
	// Array result.
	Shape []int     `json:"shape,omitempty"`
	Data  []float64 `json:"data,omitempty"`
	// Boolean array result.
	BoolData []bool `json:"bool_data,omitempty"`
	// Scalar result.
	Scalar *float64 `json:"scalar,omitempty"`
	// Property-only checks (random arrays).
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
	// Dtype kind character ('i', 'f', 'b', 'O', 'U', 'M') for
	// dtype-sensitive cases.
	Kind string `json:"kind,omitempty"`
	// Error expected.
	Error bool `json:"error,omitempty"`
	// Marks Data entries that are null in JSON (NaN in NumPy).
	NaNAt []int `json:"nan_at,omitempty"`
}

// LoadGolden reads and parses a golden suite file.
func LoadGolden(t *testing.T, path string) GoldenFile {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v", path, err)
	}
	var f GoldenFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parsing golden %s: %v", path, err)
	}
	return f
}

// ParseExpected decodes a case's expected payload.
func ParseExpected(t *testing.T, c GoldenCase) GoldenExpected {
	t.Helper()
	var e GoldenExpected
	if err := json.Unmarshal(c.Expected, &e); err != nil {
		t.Fatalf("parsing expected for %s: %v", c.Name, err)
	}
	return e
}

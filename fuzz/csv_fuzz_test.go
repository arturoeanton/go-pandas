// Package fuzz_test stresses the go-pandas entry points with arbitrary
// inputs. The invariants under fuzzing: no panics, structured errors only,
// no runaway memory.
package fuzz_test

import (
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func FuzzReadCSV(f *testing.F) {
	f.Add("a,b\n1,2\n")
	f.Add("")
	f.Add(",,,,\n")
	f.Add("a;b\n1;2\n")
	f.Add("a,b\n\"unterminated\n")
	f.Add("a,b\n1,NA\nNaN,null\n")
	f.Add("col\n2024-01-02\n")
	f.Fuzz(func(t *testing.T, input string) {
		df, err := pd.ReadCSVReader(strings.NewReader(input))
		if err != nil {
			return
		}
		// A successful parse must produce a self-consistent frame.
		rows, cols := df.Shape()
		if rows < 0 || cols < 0 {
			t.Fatalf("negative shape %d x %d", rows, cols)
		}
		_ = df.String()
	})
}

func FuzzReadJSON(f *testing.F) {
	f.Add(`[{"a": 1}]`)
	f.Add(`[]`)
	f.Add(`{"broken": `)
	f.Add(`[[1, 2], [3]]`)
	f.Fuzz(func(t *testing.T, input string) {
		df, err := pd.ReadJSONReader(strings.NewReader(input))
		if err != nil {
			return
		}
		_ = df.String()
	})
}

package pandas_test

import (
	"errors"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// TestErrorSentinelConsistency pins the sentinel each public failure
// mode wraps (the frozen error API, v0.10.2): every case must satisfy
// errors.Is against its documented sentinel.
func TestErrorSentinelConsistency(t *testing.T) {
	df, _ := pd.DataFrameFromRecords([]map[string]any{
		{"k": "a", "v": 1.0},
		{"k": "b", "v": 2.0},
	}, pd.WithColumnOrder("k", "v"))
	arr := pd.Array([]float64{1, 2, 3})

	cases := []struct {
		name     string
		sentinel error
		run      func() error
	}{
		{"bad column", errs.ErrColumnNotFound, func() error {
			_, err := df.Col("nope")
			return err
		}},
		{"bad dtype name", errs.ErrInvalidDType, func() error {
			_, err := pd.ParseDType("quaternion")
			return err
		}},
		{"bad astype", errs.ErrTypeMismatch, func() error {
			_, err := pd.ToDatetime(df.MustCol("k")) // "a"/"b" not dates
			return err
		}},
		{"bad shape", errs.ErrShapeMismatch, func() error {
			_, err := arr.Reshape(2, 2)
			return err
		}},
		{"broadcast mismatch", errs.ErrBroadcastMismatch, func() error {
			m, _ := pd.Arange(6).Reshape(2, 3)
			_, err := m.Add(pd.Array([]float64{1, 2}))
			return err
		}},
		{"bad index label", errs.ErrInvalidIndex, func() error {
			_, err := df.Loc().Rows("nope").Get()
			return err
		}},
		{"out of range", errs.ErrIndexOutOfBounds, func() error {
			_, err := df.Take([]int{99})
			return err
		}},
		{"out of range ndarray take", errs.ErrIndexOutOfBounds, func() error {
			_, err := arr.Take([]int{9}, 0)
			return err
		}},
		{"bad query", errs.ErrInvalidOperation, func() error {
			_, err := df.Query("v >>> 1")
			return err
		}},
		{"bad resample frequency", errs.ErrInvalidOperation, func() error {
			dates, err := pd.ToDatetime(pd.StringSeries("d", []string{"2026-01-01", "2026-01-02"}))
			if err != nil {
				return err
			}
			f, err := pd.NewDataFrame(dates, pd.FloatSeries("v", []float64{1, 2}))
			if err != nil {
				return err
			}
			indexed, err := f.SetIndex("d")
			if err != nil {
				return err
			}
			_, err = indexed.Resample("5min").Sum()
			return err
		}},
		{"resample without datetime index", errs.ErrInvalidIndex, func() error {
			_, err := df.Resample("D").Sum()
			return err
		}},
		{"bad pivot spec", errs.ErrInvalidOperation, func() error {
			_, err := df.PivotTable(pd.PivotTableOptions{Values: []string{"v"}})
			return err
		}},
		{"pivot multi columns keys", errs.ErrNotImplementedBase, func() error {
			_, err := df.PivotTable(pd.PivotTableOptions{
				Index: []string{"k"}, Values: []string{"v"}, Columns: []string{"k", "v"},
			})
			return err
		}},
		{"unstack flat index", errs.ErrInvalidIndex, func() error {
			_, err := df.Unstack()
			return err
		}},
		{"stack empty", errs.ErrInvalidOperation, func() error {
			empty, err := pd.NewDataFrame()
			if err != nil {
				return err
			}
			_, err = empty.Stack()
			return err
		}},
		{"bad searchsorted side", errs.ErrInvalidOperation, func() error {
			_, err := arr.SearchSorted([]float64{1}, "middle")
			return err
		}},
		{"searchsorted string array", errs.ErrTypeMismatch, func() error {
			_, err := ndarray.ArrayString([]string{"a"}).SearchSorted([]float64{1}, "left")
			return err
		}},
		{"strict categorical", errs.ErrTypeMismatch, func() error {
			_, err := pd.CategoricalSeries("c", []string{"x"}, pd.WithCategories("a"))
			return err
		}},
		{"unordered categorical compare", errs.ErrInvalidOperation, func() error {
			s, err := pd.CategoricalSeries("c", []string{"a", "b"})
			if err != nil {
				return err
			}
			cat, err := s.Cat()
			if err != nil {
				return err
			}
			_, err = cat.Gt("a")
			return err
		}},
		{"bad datetime directive", errs.ErrInvalidOperation, func() error {
			_, err := pd.ToDatetime(df.MustCol("k"), pd.WithDatetimeFormat("%Q"))
			return err
		}},
		{"string array arithmetic", errs.ErrTypeMismatch, func() error {
			_, err := ndarray.ArrayString([]string{"a"}).Add(arr)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !errors.Is(err, tc.sentinel) {
				t.Fatalf("error %v does not wrap %v", err, tc.sentinel)
			}
		})
	}
}

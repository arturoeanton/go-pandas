package compat_test

import (
	"math"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/ndarray"
)

var setopsCases = map[string]caseFn{
	"isin_numeric": func(t *testing.T) (any, error) {
		a := pd.Array([]float64{1, 2, 3, 2, math.NaN()})
		return a.IsIn([]any{2.0, 7.0, math.NaN()}), nil
	},
	"isin_string": func(t *testing.T) (any, error) {
		a := ndarray.ArrayString([]string{"a", "b", "c"})
		return a.IsIn([]any{"b", "z"}), nil
	},
	"searchsorted_left": func(t *testing.T) (any, error) {
		a := pd.Array([]float64{1, 2, 2, 4, 7})
		pos, err := a.SearchSorted([]float64{0, 2, 3, 9}, "left")
		if err != nil {
			return nil, err
		}
		return intsToArray(pos), nil
	},
	"searchsorted_right": func(t *testing.T) (any, error) {
		a := pd.Array([]float64{1, 2, 2, 4, 7})
		pos, err := a.SearchSorted([]float64{0, 2, 3, 9}, "right")
		if err != nil {
			return nil, err
		}
		return intsToArray(pos), nil
	},
	"take_1d": func(t *testing.T) (any, error) {
		a := pd.Array([]float64{10, 20, 30})
		return a.Take([]int{0, 2, 1}, 0)
	},
}

func intsToArray(values []int) *pd.NDArray {
	floats := make([]float64, len(values))
	for i, v := range values {
		floats[i] = float64(v)
	}
	return pd.Array(floats)
}

func init() {
	for name, fn := range setopsCases {
		numpyCases[name] = fn
	}
}

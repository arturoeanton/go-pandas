package ndarray

import (
	"fmt"
	"math"
	"sort"

	"github.com/arturoeanton/go-pandas/errs"
)

// IsIn reports element-wise membership in the candidate values, like
// np.isin (v0.10). Numeric and bool arrays test through the shared
// float view (so int 1 matches 1.0 and true matches 1); string arrays
// test string equality. NaN never matches (NaN != NaN, like NumPy with
// default settings).
func (a *NDArray) IsIn(values []any) *BoolArray {
	out := &BoolArray{data: make([]bool, a.Size()), shape: a.Shape()}
	if load := a.stringLoader(); load != nil {
		set := make(map[string]bool, len(values))
		for _, v := range values {
			if s, ok := v.(string); ok {
				set[s] = true
			}
		}
		i := 0
		a.iter(func(off int) {
			out.data[i] = set[load(off)]
			i++
		})
		return out
	}
	load := a.mustFloatLoader("isin")
	set := make(map[float64]bool, len(values))
	for _, v := range values {
		switch x := v.(type) {
		case bool:
			if x {
				set[1] = true
			} else {
				set[0] = true
			}
		default:
			if f, ok := toFloat(v); ok && !math.IsNaN(f) {
				set[f] = true
			}
		}
	}
	i := 0
	a.iter(func(off int) {
		x := load(off)
		out.data[i] = !math.IsNaN(x) && set[x]
		i++
	})
	return out
}

// SearchSorted returns, for each query value, the insertion index that
// keeps the 1-D numeric array sorted, like np.searchsorted (v0.10).
// side is "left" (first suitable position) or "right" (past the last
// equal element). The array MUST already be sorted ascending — this is
// a precondition, not checked (documented, like NumPy).
func (a *NDArray) SearchSorted(values []float64, side string) ([]int, error) {
	if len(a.Shape()) != 1 {
		return nil, fmt.Errorf("%w: SearchSorted needs a 1-D array", errs.ErrInvalidOperation)
	}
	if side != "left" && side != "right" {
		return nil, fmt.Errorf("%w: SearchSorted side must be \"left\" or \"right\", got %q", errs.ErrInvalidOperation, side)
	}
	load := a.floatLoader()
	if load == nil {
		return nil, fmt.Errorf("%w: SearchSorted on %s array", errs.ErrTypeMismatch, a.DType())
	}
	n := a.Size()
	// Materialize once through the logical order (handles views).
	data := make([]float64, 0, n)
	a.iter(func(off int) { data = append(data, load(off)) })

	out := make([]int, len(values))
	for i, v := range values {
		if side == "left" {
			out[i] = sort.Search(n, func(j int) bool { return data[j] >= v })
			continue
		}
		out[i] = sort.Search(n, func(j int) bool { return data[j] > v })
	}
	return out, nil
}

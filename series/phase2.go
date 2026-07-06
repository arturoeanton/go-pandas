package series

import (
	"fmt"
	"math"
	"sort"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
)

// ILoc returns the value at a position (alias of At, pandas s.iloc[i]).
func (s *Series) ILoc(pos int) (any, error) { return s.At(pos) }

// AtLabel returns the value for an index label (alias of Loc, pandas
// s.at[label]).
func (s *Series) AtLabel(label any) (any, error) { return s.Loc(label) }

// ReplaceNA is an alias of FillNA, mirroring Series.replace(NA, v) usage.
func (s *Series) ReplaceNA(v any) *Series { return s.FillNA(v) }

// Reindex conforms the series to a new index: labels present in the
// current index keep their value, new labels get NA.
func (s *Series) Reindex(idx index.Index) (*Series, error) {
	if idx == nil {
		return nil, fmt.Errorf("%w: nil index", errs.ErrInvalidIndex)
	}
	pos := make([]int, idx.Len())
	for i := 0; i < idx.Len(); i++ {
		if p, ok := s.index.Pos(idx.At(i)); ok {
			pos[i] = p
		} else {
			pos[i] = -1
		}
	}
	out, err := s.Take(pos)
	if err != nil {
		return nil, err
	}
	return out.WithIndexed(idx), nil
}

// Argsort returns the positions that would sort the series ascending,
// missing values last (like s.argsort()).
func (s *Series) Argsort() *Series {
	pos := make([]int, s.Len())
	for i := range pos {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		return lessAt(s, pos[a], pos[b], true)
	})
	values := make([]any, len(pos))
	for i, p := range pos {
		values[i] = p
	}
	return NewSeries(s.name, values, WithIndex(s.index))
}

// RankOptions configures Rank.
type RankOptions struct {
	// Method is average (default), min, max, first or dense.
	Method string
	// Ascending ranks smallest-first (default true).
	Ascending bool
}

// RankOption mutates RankOptions.
type RankOption func(*RankOptions)

// RankMethod sets the tie-breaking method.
func RankMethod(m string) RankOption { return func(o *RankOptions) { o.Method = m } }

// RankAscending sets the rank direction.
func RankAscending(v bool) RankOption { return func(o *RankOptions) { o.Ascending = v } }

// Rank returns the rank of each value (1-based, like s.rank()). Missing
// values get NA ranks.
func (s *Series) Rank(opts ...RankOption) (*Series, error) {
	o := RankOptions{Method: "average", Ascending: true}
	for _, f := range opts {
		f(&o)
	}
	switch o.Method {
	case "average", "min", "max", "first", "dense":
	default:
		return nil, fmt.Errorf("%w: rank method %q", errs.ErrInvalidOperation, o.Method)
	}
	type entry struct {
		pos int
		val float64
	}
	var entries []entry
	for i := range s.data {
		if s.mask[i] {
			continue
		}
		f, ok := dtype.AsFloat(s.data[i])
		if !ok {
			return nil, fmt.Errorf("%w: rank on non-numeric value %T", errs.ErrTypeMismatch, s.data[i])
		}
		entries = append(entries, entry{pos: i, val: f})
	}
	sort.SliceStable(entries, func(a, b int) bool {
		if o.Ascending {
			return entries[a].val < entries[b].val
		}
		return entries[a].val > entries[b].val
	})
	data := make([]any, s.Len())
	mask := make([]bool, s.Len())
	for i := range mask {
		mask[i] = true
	}
	dense := 0.0
	i := 0
	for i < len(entries) {
		j := i
		for j < len(entries) && entries[j].val == entries[i].val {
			j++
		}
		dense++
		for k := i; k < j; k++ {
			var rank float64
			switch o.Method {
			case "average":
				rank = float64(i+1+j) / 2 // mean of ranks i+1..j
			case "min":
				rank = float64(i + 1)
			case "max":
				rank = float64(j)
			case "first":
				rank = float64(k + 1)
			case "dense":
				rank = dense
			}
			data[entries[k].pos] = rank
			mask[entries[k].pos] = false
		}
		i = j
	}
	return &Series{name: s.name, dtype: dtype.Float64, data: data, mask: mask, index: s.index.Clone()}, nil
}

// Diff returns the difference with the value `periods` positions earlier
// (like s.diff()).
func (s *Series) Diff(periods int) (*Series, error) {
	return s.shiftCombine(periods, func(cur, prev float64) float64 { return cur - prev })
}

// PctChange returns the fractional change against the value `periods`
// positions earlier (like s.pct_change()).
func (s *Series) PctChange(periods int) (*Series, error) {
	return s.shiftCombine(periods, func(cur, prev float64) float64 { return cur/prev - 1 })
}

func (s *Series) shiftCombine(periods int, f func(cur, prev float64) float64) (*Series, error) {
	if periods == 0 {
		return nil, fmt.Errorf("%w: periods must be non-zero", errs.ErrInvalidOperation)
	}
	n := s.Len()
	data := make([]any, n)
	mask := make([]bool, n)
	for i := 0; i < n; i++ {
		prev := i - periods
		if prev < 0 || prev >= n || s.mask[i] || s.mask[prev] {
			mask[i] = true
			continue
		}
		cur, okC := dtype.AsFloat(s.data[i])
		pv, okP := dtype.AsFloat(s.data[prev])
		if !okC || !okP {
			return nil, fmt.Errorf("%w: diff/pct_change on non-numeric values", errs.ErrTypeMismatch)
		}
		data[i] = f(cur, pv)
	}
	return &Series{name: s.name, dtype: dtype.Float64, data: data, mask: mask, index: s.index.Clone()}, nil
}

// cumulative applies a running fold, propagating NA at missing positions
// without breaking the accumulation (pandas semantics).
func (s *Series) cumulative(init float64, f func(acc, x float64) float64) (*Series, error) {
	n := s.Len()
	data := make([]any, n)
	mask := make([]bool, n)
	acc := init
	started := false
	for i := 0; i < n; i++ {
		if s.mask[i] {
			mask[i] = true
			continue
		}
		x, ok := dtype.AsFloat(s.data[i])
		if !ok {
			return nil, fmt.Errorf("%w: cumulative op on non-numeric value %T", errs.ErrTypeMismatch, s.data[i])
		}
		if !started {
			acc = x
			started = true
		} else {
			acc = f(acc, x)
		}
		data[i] = acc
	}
	return &Series{name: s.name, dtype: dtype.Float64, data: data, mask: mask, index: s.index.Clone()}, nil
}

// Cumsum returns the cumulative sum (like s.cumsum()).
func (s *Series) Cumsum() (*Series, error) {
	return s.cumulative(0, func(acc, x float64) float64 { return acc + x })
}

// Cumprod returns the cumulative product.
func (s *Series) Cumprod() (*Series, error) {
	return s.cumulative(1, func(acc, x float64) float64 { return acc * x })
}

// Cummin returns the cumulative minimum.
func (s *Series) Cummin() (*Series, error) {
	return s.cumulative(math.Inf(1), math.Min)
}

// Cummax returns the cumulative maximum.
func (s *Series) Cummax() (*Series, error) {
	return s.cumulative(math.Inf(-1), math.Max)
}

// Clip limits values to [lower, upper]; pass nil to skip a bound.
func (s *Series) Clip(lower, upper any) (*Series, error) {
	var lo, hi float64
	hasLo, hasHi := false, false
	if lower != nil {
		f, ok := dtype.AsFloat(lower)
		if !ok {
			return nil, fmt.Errorf("%w: clip lower bound %T", errs.ErrTypeMismatch, lower)
		}
		lo, hasLo = f, true
	}
	if upper != nil {
		f, ok := dtype.AsFloat(upper)
		if !ok {
			return nil, fmt.Errorf("%w: clip upper bound %T", errs.ErrTypeMismatch, upper)
		}
		hi, hasHi = f, true
	}
	return s.mapNumeric(func(x float64) float64 {
		if hasLo && x < lo {
			return lo
		}
		if hasHi && x > hi {
			return hi
		}
		return x
	})
}

// Round rounds values half to even (banker's rounding, like pandas) to
// the given decimals.
func (s *Series) Round(decimals int) (*Series, error) {
	scale := math.Pow(10, float64(decimals))
	return s.mapNumeric(func(x float64) float64 {
		return math.RoundToEven(x*scale) / scale
	})
}

// Abs returns the absolute values.
func (s *Series) Abs() (*Series, error) {
	return s.mapNumeric(math.Abs)
}

// mapNumeric applies a float function, keeping integer dtype for integer
// input when the result stays integral.
func (s *Series) mapNumeric(f func(x float64) float64) (*Series, error) {
	n := s.Len()
	data := make([]any, n)
	keepInt := dtype.IsInteger(s.dtype)
	for i := 0; i < n; i++ {
		if s.mask[i] {
			continue
		}
		x, ok := dtype.AsFloat(s.data[i])
		if !ok {
			return nil, fmt.Errorf("%w: numeric op on %T", errs.ErrTypeMismatch, s.data[i])
		}
		r := f(x)
		if keepInt && r == math.Trunc(r) {
			data[i] = int64(r)
		} else {
			keepInt = false
			data[i] = r
		}
	}
	dt := dtype.Float64
	if keepInt {
		dt = dtype.Int64
	} else {
		// re-normalize earlier values written as int64
		for i := range data {
			if v, ok := data[i].(int64); ok {
				data[i] = float64(v)
			}
		}
	}
	return &Series{name: s.name, dtype: dt, data: data, mask: append([]bool(nil), s.mask...), index: s.index.Clone()}, nil
}

// Shift moves values by `periods` positions, filling the gap with NA.
func (s *Series) Shift(periods int) *Series {
	n := s.Len()
	data := make([]any, n)
	mask := make([]bool, n)
	for i := 0; i < n; i++ {
		src := i - periods
		if src < 0 || src >= n {
			mask[i] = true
			continue
		}
		data[i] = s.data[src]
		mask[i] = s.mask[src]
	}
	return &Series{name: s.name, dtype: s.dtype, data: data, mask: mask, index: s.index.Clone()}
}

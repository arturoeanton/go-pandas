package series

import (
	"fmt"
	"math"
	"sort"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
)

// ReduceOptions controls reductions; SkipNA defaults to true (pandas).
type ReduceOptions struct {
	SkipNA bool
}

// ReduceOption mutates ReduceOptions.
type ReduceOption func(*ReduceOptions)

// SkipNA sets whether reductions ignore missing values.
func SkipNA(v bool) ReduceOption {
	return func(o *ReduceOptions) { o.SkipNA = v }
}

func reduceOpts(opts []ReduceOption) ReduceOptions {
	o := ReduceOptions{SkipNA: true}
	for _, f := range opts {
		f(&o)
	}
	return o
}

// numericValues extracts the non-missing values as float64. When skipna is
// false and a missing value exists, ok is false (result must be NaN).
func (s *Series) numericValues(o ReduceOptions) ([]float64, bool, error) {
	var out []float64
	for i := range s.data {
		if s.mask[i] {
			if !o.SkipNA {
				return nil, false, nil
			}
			continue
		}
		f, ok := dtype.AsFloat(s.data[i])
		if !ok {
			return nil, false, fmt.Errorf("%w: non-numeric value %T in reduction", errs.ErrTypeMismatch, s.data[i])
		}
		out = append(out, f)
	}
	return out, true, nil
}

// Count returns the number of non-missing values.
func (s *Series) Count() int {
	n := 0
	for _, m := range s.mask {
		if !m {
			n++
		}
	}
	return n
}

// Sum returns the sum of the values (0 for an empty/all-NA series, like
// pandas).
func (s *Series) Sum(opts ...ReduceOption) (float64, error) {
	vals, ok, err := s.numericValues(reduceOpts(opts))
	if err != nil {
		return 0, err
	}
	if !ok {
		return math.NaN(), nil
	}
	acc := 0.0
	for _, v := range vals {
		acc += v
	}
	return acc, nil
}

// Mean returns the arithmetic mean (NaN when empty).
func (s *Series) Mean(opts ...ReduceOption) (float64, error) {
	vals, ok, err := s.numericValues(reduceOpts(opts))
	if err != nil {
		return 0, err
	}
	if !ok || len(vals) == 0 {
		return math.NaN(), nil
	}
	acc := 0.0
	for _, v := range vals {
		acc += v
	}
	return acc / float64(len(vals)), nil
}

// Median returns the middle value (mean of the two middle values for even
// counts).
func (s *Series) Median(opts ...ReduceOption) (float64, error) {
	return s.Quantile(0.5, opts...)
}

// Quantile returns the q-quantile with linear interpolation (pandas
// default).
func (s *Series) Quantile(q float64, opts ...ReduceOption) (float64, error) {
	if q < 0 || q > 1 {
		return 0, fmt.Errorf("%w: quantile %v not in [0, 1]", errs.ErrInvalidOperation, q)
	}
	vals, ok, err := s.numericValues(reduceOpts(opts))
	if err != nil {
		return 0, err
	}
	if !ok || len(vals) == 0 {
		return math.NaN(), nil
	}
	sort.Float64s(vals)
	return quantileSorted(vals, q), nil
}

func quantileSorted(vals []float64, q float64) float64 {
	n := len(vals)
	if n == 1 {
		return vals[0]
	}
	pos := q * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return vals[lo]
	}
	frac := pos - float64(lo)
	return vals[lo]*(1-frac) + vals[hi]*frac
}

// Min returns the smallest value; works on any orderable dtype (numbers,
// strings, times).
func (s *Series) Min(opts ...ReduceOption) (any, error) {
	return s.extreme(reduceOpts(opts), func(c int) bool { return c < 0 })
}

// Max returns the largest value.
func (s *Series) Max(opts ...ReduceOption) (any, error) {
	return s.extreme(reduceOpts(opts), func(c int) bool { return c > 0 })
}

func (s *Series) extreme(o ReduceOptions, better func(c int) bool) (any, error) {
	var best any
	found := false
	for i := range s.data {
		if s.mask[i] {
			if !o.SkipNA {
				return nil, nil
			}
			continue
		}
		if !found {
			best = s.data[i]
			found = true
			continue
		}
		c, ok := expr.CompareValues(s.data[i], best)
		if !ok {
			return nil, fmt.Errorf("%w: cannot order %T against %T", errs.ErrTypeMismatch, s.data[i], best)
		}
		if better(c) {
			best = s.data[i]
		}
	}
	if !found {
		return nil, nil
	}
	return best, nil
}

// Var returns the sample variance (ddof=1, pandas default).
func (s *Series) Var(opts ...ReduceOption) (float64, error) {
	vals, ok, err := s.numericValues(reduceOpts(opts))
	if err != nil {
		return 0, err
	}
	if !ok || len(vals) < 2 {
		return math.NaN(), nil
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	acc := 0.0
	for _, v := range vals {
		d := v - mean
		acc += d * d
	}
	return acc / float64(len(vals)-1), nil
}

// Std returns the sample standard deviation (ddof=1).
func (s *Series) Std(opts ...ReduceOption) (float64, error) {
	v, err := s.Var(opts...)
	if err != nil {
		return 0, err
	}
	return math.Sqrt(v), nil
}

// Describe returns count/mean/std/min/25%/50%/75%/max as a labeled series,
// mirroring Series.describe() for numeric data.
func (s *Series) Describe() (*Series, error) {
	count := float64(s.Count())
	mean, err := s.Mean()
	if err != nil {
		return nil, err
	}
	std, _ := s.Std()
	minV, err := s.Min()
	if err != nil {
		return nil, err
	}
	q25, _ := s.Quantile(0.25)
	q50, _ := s.Quantile(0.5)
	q75, _ := s.Quantile(0.75)
	maxV, _ := s.Max()
	minF, _ := dtype.AsFloat(minV)
	maxF, _ := dtype.AsFloat(maxV)
	if minV == nil {
		minF = math.NaN()
	}
	if maxV == nil {
		maxF = math.NaN()
	}
	values := []any{count, mean, std, minF, q25, q50, q75, maxF}
	labels := []string{"count", "mean", "std", "min", "25%", "50%", "75%", "max"}
	out := SeriesOf(s.name, values)
	return out.WithIndexed(index.NewStringIndex(labels)), nil
}

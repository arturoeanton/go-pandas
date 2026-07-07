package series

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// RollingOptions configures window behavior.
type RollingOptions struct {
	// MinPeriods is the minimum number of present values required for a
	// window to produce a value; defaults to the window size.
	MinPeriods int
	// Center labels each window at its center instead of its right edge.
	Center bool
}

// RollingOption mutates RollingOptions.
type RollingOption func(*RollingOptions)

// RollingMinPeriods sets the minimum observations per window.
func RollingMinPeriods(n int) RollingOption {
	return func(o *RollingOptions) { o.MinPeriods = n }
}

// RollingCenter centers the window labels.
func RollingCenter(v bool) RollingOption {
	return func(o *RollingOptions) { o.Center = v }
}

// RollingSeries is a fixed-size rolling window over a numeric series.
type RollingSeries struct {
	s      *Series
	window int
	opts   RollingOptions
}

// Rolling creates a rolling window of the given size.
func (s *Series) Rolling(window int, opts ...RollingOption) *RollingSeries {
	o := RollingOptions{MinPeriods: window}
	for _, f := range opts {
		f(&o)
	}
	return &RollingSeries{s: s, window: window, opts: o}
}

// numericBuffer extracts float values plus a present flag per position.
func numericBuffer(s *Series) ([]float64, []bool, error) {
	if fs, mask, ok := s.col.Float64s(); ok {
		present := make([]bool, len(fs))
		for i := range present {
			present[i] = !mask[i]
		}
		return fs, present, nil
	}
	n := s.Len()
	floats := make([]float64, n)
	present := make([]bool, n)
	for i := 0; i < n; i++ {
		if s.col.IsNA(i) {
			continue
		}
		v, ok := dtype.AsFloat(s.col.Value(i))
		if !ok {
			return nil, nil, fmt.Errorf("%w: window op on non-numeric value %T", errs.ErrTypeMismatch, s.col.Value(i))
		}
		floats[i] = v
		present[i] = true
	}
	return floats, present, nil
}

// aggregate slides the window and reduces each one with f (which receives
// only the present values in the window).
func (r *RollingSeries) aggregate(f func(window []float64) float64) (*Series, error) {
	if r.window <= 0 {
		return nil, fmt.Errorf("%w: rolling window must be positive", errs.ErrInvalidOperation)
	}
	src := r.s
	n := src.Len()
	floats, present, err := numericBuffer(src)
	if err != nil {
		return nil, err
	}
	data := make([]float64, n)
	mask := make([]bool, n)
	offset := 0
	if r.opts.Center {
		offset = r.window / 2
	}
	for i := 0; i < n; i++ {
		end := i + offset // inclusive right edge of the window
		start := end - r.window + 1
		// Windows are clipped at both edges; MinPeriods alone decides
		// whether a clipped window produces a value (pandas semantics,
		// including centered windows at the tail).
		var buf []float64
		for j := max(start, 0); j <= min(end, n-1); j++ {
			if present[j] {
				buf = append(buf, floats[j])
			}
		}
		if len(buf) < r.opts.MinPeriods {
			mask[i] = true
			continue
		}
		data[i] = f(buf)
	}
	return floatColumnSeries(src.name, data, mask, src.index), nil
}

// Sum returns the rolling sum.
func (r *RollingSeries) Sum() (*Series, error) {
	return r.aggregate(func(w []float64) float64 {
		acc := 0.0
		for _, v := range w {
			acc += v
		}
		return acc
	})
}

// Mean returns the rolling mean.
func (r *RollingSeries) Mean() (*Series, error) {
	return r.aggregate(func(w []float64) float64 {
		acc := 0.0
		for _, v := range w {
			acc += v
		}
		return acc / float64(len(w))
	})
}

// Min returns the rolling minimum.
func (r *RollingSeries) Min() (*Series, error) {
	return r.aggregate(func(w []float64) float64 {
		best := math.Inf(1)
		for _, v := range w {
			best = math.Min(best, v)
		}
		return best
	})
}

// Max returns the rolling maximum.
func (r *RollingSeries) Max() (*Series, error) {
	return r.aggregate(func(w []float64) float64 {
		best := math.Inf(-1)
		for _, v := range w {
			best = math.Max(best, v)
		}
		return best
	})
}

// Std returns the rolling sample standard deviation (ddof=1).
func (r *RollingSeries) Std() (*Series, error) {
	return r.aggregate(func(w []float64) float64 {
		if len(w) < 2 {
			return math.NaN()
		}
		mean := 0.0
		for _, v := range w {
			mean += v
		}
		mean /= float64(len(w))
		acc := 0.0
		for _, v := range w {
			d := v - mean
			acc += d * d
		}
		return math.Sqrt(acc / float64(len(w)-1))
	})
}

// ExpandingSeries is an expanding window (window i covers positions 0..i).
type ExpandingSeries struct {
	s          *Series
	minPeriods int
}

// Expanding creates an expanding window with the given minimum number of
// observations (1 when omitted).
func (s *Series) Expanding(minPeriods ...int) *ExpandingSeries {
	mp := 1
	if len(minPeriods) > 0 && minPeriods[0] > 0 {
		mp = minPeriods[0]
	}
	return &ExpandingSeries{s: s, minPeriods: mp}
}

func (e *ExpandingSeries) aggregate(f func(window []float64) float64) (*Series, error) {
	src := e.s
	n := src.Len()
	floats, present, err := numericBuffer(src)
	if err != nil {
		return nil, err
	}
	data := make([]float64, n)
	mask := make([]bool, n)
	var buf []float64
	for i := 0; i < n; i++ {
		if present[i] {
			buf = append(buf, floats[i])
		}
		if len(buf) < e.minPeriods {
			mask[i] = true
			continue
		}
		data[i] = f(buf)
	}
	return floatColumnSeries(src.name, data, mask, src.index), nil
}

// Sum returns the expanding sum.
func (e *ExpandingSeries) Sum() (*Series, error) {
	return e.aggregate(func(w []float64) float64 {
		acc := 0.0
		for _, v := range w {
			acc += v
		}
		return acc
	})
}

// Mean returns the expanding mean.
func (e *ExpandingSeries) Mean() (*Series, error) {
	return e.aggregate(func(w []float64) float64 {
		acc := 0.0
		for _, v := range w {
			acc += v
		}
		return acc / float64(len(w))
	})
}

package series

import (
	"math"
	"sort"
)

func windowSum(w []float64) float64 {
	acc := 0.0
	for _, v := range w {
		acc += v
	}
	return acc
}

func windowMedian(w []float64) float64 {
	c := append([]float64(nil), w...)
	sort.Float64s(c)
	return quantileSorted(c, 0.5)
}

func windowVar(w []float64) float64 {
	if len(w) < 2 {
		return math.NaN()
	}
	mean := windowSum(w) / float64(len(w))
	acc := 0.0
	for _, v := range w {
		d := v - mean
		acc += d * d
	}
	return acc / float64(len(w)-1)
}

// Count returns the rolling count of present values.
func (r *RollingSeries) Count() (*Series, error) {
	return r.aggregate(func(w []float64) float64 { return float64(len(w)) })
}

// Median returns the rolling median.
func (r *RollingSeries) Median() (*Series, error) {
	return r.aggregate(windowMedian)
}

// Var returns the rolling sample variance (ddof=1).
func (r *RollingSeries) Var() (*Series, error) {
	return r.aggregate(windowVar)
}

// Count returns the expanding count of present values.
func (e *ExpandingSeries) Count() (*Series, error) {
	return e.aggregate(func(w []float64) float64 { return float64(len(w)) })
}

// Median returns the expanding median.
func (e *ExpandingSeries) Median() (*Series, error) {
	return e.aggregate(windowMedian)
}

// Min returns the expanding minimum.
func (e *ExpandingSeries) Min() (*Series, error) {
	return e.aggregate(func(w []float64) float64 {
		best := math.Inf(1)
		for _, v := range w {
			best = math.Min(best, v)
		}
		return best
	})
}

// Max returns the expanding maximum.
func (e *ExpandingSeries) Max() (*Series, error) {
	return e.aggregate(func(w []float64) float64 {
		best := math.Inf(-1)
		for _, v := range w {
			best = math.Max(best, v)
		}
		return best
	})
}

// Var returns the expanding sample variance (ddof=1).
func (e *ExpandingSeries) Var() (*Series, error) {
	return e.aggregate(windowVar)
}

// Std returns the expanding sample standard deviation (ddof=1).
func (e *ExpandingSeries) Std() (*Series, error) {
	return e.aggregate(func(w []float64) float64 { return math.Sqrt(windowVar(w)) })
}

// MinPeriods is an alias of RollingMinPeriods matching the pandas keyword.
func MinPeriods(n int) RollingOption { return RollingMinPeriods(n) }

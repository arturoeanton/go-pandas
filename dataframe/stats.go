package dataframe

import (
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// ReduceOption re-exports the series reduction options.
type ReduceOption = series.ReduceOption

// numericColumns returns the columns with a numeric or bool dtype.
func (df *DataFrame) numericColumns() []*series.Series {
	var out []*series.Series
	for _, c := range df.columns {
		if dtype.IsNumeric(c.DType()) || dtype.IsBool(c.DType()) {
			out = append(out, c)
		}
	}
	return out
}

// Count returns the non-missing count per column.
func (df *DataFrame) Count() map[string]int {
	out := make(map[string]int, len(df.columns))
	for _, c := range df.columns {
		out[c.Name()] = c.Count()
	}
	return out
}

func (df *DataFrame) floatReduce(f func(c *series.Series) (float64, error)) map[string]float64 {
	out := make(map[string]float64)
	for _, c := range df.numericColumns() {
		v, err := f(c)
		if err != nil {
			continue
		}
		out[c.Name()] = v
	}
	return out
}

// Sum returns the per-column sums of numeric columns.
func (df *DataFrame) Sum(opts ...ReduceOption) map[string]float64 {
	return df.floatReduce(func(c *series.Series) (float64, error) { return c.Sum(opts...) })
}

// Mean returns the per-column means of numeric columns.
func (df *DataFrame) Mean(opts ...ReduceOption) map[string]float64 {
	return df.floatReduce(func(c *series.Series) (float64, error) { return c.Mean(opts...) })
}

// Median returns the per-column medians of numeric columns.
func (df *DataFrame) Median(opts ...ReduceOption) map[string]float64 {
	return df.floatReduce(func(c *series.Series) (float64, error) { return c.Median(opts...) })
}

// Var returns the per-column sample variances (ddof=1).
func (df *DataFrame) Var(opts ...ReduceOption) map[string]float64 {
	return df.floatReduce(func(c *series.Series) (float64, error) { return c.Var(opts...) })
}

// Std returns the per-column sample standard deviations (ddof=1).
func (df *DataFrame) Std(opts ...ReduceOption) map[string]float64 {
	return df.floatReduce(func(c *series.Series) (float64, error) { return c.Std(opts...) })
}

// Quantile returns the per-column q-quantiles of numeric columns.
func (df *DataFrame) Quantile(q float64, opts ...ReduceOption) map[string]float64 {
	return df.floatReduce(func(c *series.Series) (float64, error) { return c.Quantile(q, opts...) })
}

// Min returns the per-column minima (any orderable dtype).
func (df *DataFrame) Min(opts ...ReduceOption) map[string]any {
	out := make(map[string]any)
	for _, c := range df.columns {
		v, err := c.Min(opts...)
		if err == nil {
			out[c.Name()] = v
		}
	}
	return out
}

// Max returns the per-column maxima.
func (df *DataFrame) Max(opts ...ReduceOption) map[string]any {
	out := make(map[string]any)
	for _, c := range df.columns {
		v, err := c.Max(opts...)
		if err == nil {
			out[c.Name()] = v
		}
	}
	return out
}

// Describe summarizes numeric columns with count/mean/std/min/25%/50%/75%/
// max rows, like df.describe().
func (df *DataFrame) Describe() *DataFrame {
	numeric := df.numericColumns()
	labels := []string{"count", "mean", "std", "min", "25%", "50%", "75%", "max"}
	idx := index.NewStringIndex(labels)
	cols := make([]*series.Series, 0, len(numeric))
	for _, c := range numeric {
		count := float64(c.Count())
		mean, _ := c.Mean()
		std, _ := c.Std()
		minV, _ := c.Min()
		q25, _ := c.Quantile(0.25)
		q50, _ := c.Quantile(0.5)
		q75, _ := c.Quantile(0.75)
		maxV, _ := c.Max()
		minF, ok := dtype.AsFloat(minV)
		if !ok {
			minF = math.NaN()
		}
		maxF, ok := dtype.AsFloat(maxV)
		if !ok {
			maxF = math.NaN()
		}
		values := []any{count, mean, std, minF, q25, q50, q75, maxF}
		cols = append(cols, series.NewSeries(c.Name(), values, series.WithIndex(idx)))
	}
	out, _ := newFrame(cols, idx)
	return out
}

package dataframe

import (
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/series"
)

// RollingOption re-exports the series rolling options.
type RollingOption = series.RollingOption

// RollingDataFrame applies rolling windows to every numeric column.
type RollingDataFrame struct {
	df     *DataFrame
	window int
	opts   []RollingOption
}

// Rolling creates a rolling window over the frame's numeric columns.
func (df *DataFrame) Rolling(window int, opts ...RollingOption) *RollingDataFrame {
	return &RollingDataFrame{df: df, window: window, opts: opts}
}

func (r *RollingDataFrame) aggregate(f func(rs *series.RollingSeries) (*series.Series, error)) (*DataFrame, error) {
	var cols []*series.Series
	for _, c := range r.df.columns {
		if !dtype.IsNumeric(c.DType()) && !dtype.IsBool(c.DType()) {
			continue
		}
		out, err := f(c.Rolling(r.window, r.opts...))
		if err != nil {
			return nil, err
		}
		cols = append(cols, out)
	}
	return newFrame(cols, r.df.index.Clone())
}

// Sum returns per-column rolling sums.
func (r *RollingDataFrame) Sum() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Sum() })
}

// Mean returns per-column rolling means.
func (r *RollingDataFrame) Mean() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Mean() })
}

// Min returns per-column rolling minima.
func (r *RollingDataFrame) Min() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Min() })
}

// Max returns per-column rolling maxima.
func (r *RollingDataFrame) Max() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Max() })
}

// Std returns per-column rolling sample standard deviations.
func (r *RollingDataFrame) Std() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Std() })
}

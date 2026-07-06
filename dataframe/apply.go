package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// Apply reduces along an axis: axis=0 applies fn to each column's values
// (result indexed by column name); axis=1 applies fn to each row.
func (df *DataFrame) Apply(axis int, fn func(values []any) any) (*series.Series, error) {
	switch axis {
	case 0:
		values := make([]any, len(df.columns))
		names := make([]string, len(df.columns))
		for j, c := range df.columns {
			values[j] = fn(c.Values())
			names[j] = c.Name()
		}
		return series.NewSeries("", values, series.WithIndex(index.NewStringIndex(names))), nil
	case 1:
		rows := df.ToRows()
		values := make([]any, len(rows))
		for i, row := range rows {
			values[i] = fn(row)
		}
		return series.NewSeries("", values, series.WithIndex(df.index)), nil
	}
	return nil, fmt.Errorf("%w: axis %d", errs.ErrInvalidAxis, axis)
}

// Map applies fn to every cell, like df.map (formerly applymap).
func (df *DataFrame) Map(fn func(v any) any) *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.Apply(fn)
	}
	out, _ := newFrame(cols, df.index.Clone())
	return out
}

// Pipe threads the frame through fn, enabling method-chain style helpers.
func (df *DataFrame) Pipe(fn func(*DataFrame) (*DataFrame, error)) (*DataFrame, error) {
	return fn(df)
}

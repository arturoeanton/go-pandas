package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/series"
)

// Col returns a column by name (the pandas df["col"]).
func (df *DataFrame) Col(name string) (*series.Series, error) {
	i, ok := df.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, name)
	}
	return df.columns[i], nil
}

// Column is an alias of Col.
func (df *DataFrame) Column(name string) (*series.Series, error) { return df.Col(name) }

// MustCol is Col that panics when the column does not exist.
func (df *DataFrame) MustCol(name string) *series.Series {
	c, err := df.Col(name)
	if err != nil {
		panic(err)
	}
	return c
}

// Select returns a frame with only the named columns, in the given order
// (the pandas df[["a", "b"]]).
func (df *DataFrame) Select(names ...string) (*DataFrame, error) {
	cols := make([]*series.Series, len(names))
	for i, name := range names {
		c, err := df.Col(name)
		if err != nil {
			return nil, err
		}
		cols[i] = c.Copy()
	}
	return newFrame(cols, df.index.Clone())
}

// Drop returns a frame without the named columns.
func (df *DataFrame) Drop(names ...string) (*DataFrame, error) {
	dropped := make(map[string]bool, len(names))
	for _, name := range names {
		if _, ok := df.byName[name]; !ok {
			return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, name)
		}
		dropped[name] = true
	}
	var cols []*series.Series
	for _, c := range df.columns {
		if !dropped[c.Name()] {
			cols = append(cols, c.Copy())
		}
	}
	return newFrame(cols, df.index.Clone())
}

// Rename returns a frame with columns renamed by the mapping; unknown keys
// are ignored, like pandas.
func (df *DataFrame) Rename(mapping map[string]string) (*DataFrame, error) {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		if newName, ok := mapping[c.Name()]; ok {
			cols[i] = c.Rename(newName)
		} else {
			cols[i] = c.Copy()
		}
	}
	return newFrame(cols, df.index.Clone())
}

// AddColumn appends a column (equivalent to Assign).
func (df *DataFrame) AddColumn(name string, s *series.Series) (*DataFrame, error) {
	return df.Assign(name, s)
}

// Assign returns a frame with a column added or replaced by a series.
func (df *DataFrame) Assign(name string, s *series.Series) (*DataFrame, error) {
	if s.Len() != df.Len() {
		return nil, fmt.Errorf("%w: assigning %d values to %d rows", errs.ErrLengthMismatch, s.Len(), df.Len())
	}
	var cols []*series.Series
	replaced := false
	for _, c := range df.columns {
		if c.Name() == name {
			cols = append(cols, s.Rename(name))
			replaced = true
		} else {
			cols = append(cols, c.Copy())
		}
	}
	if !replaced {
		cols = append(cols, s.Rename(name))
	}
	return newFrame(cols, df.index.Clone())
}

// AssignValue adds or replaces a column with a repeated scalar.
func (df *DataFrame) AssignValue(name string, value any) (*DataFrame, error) {
	values := make([]any, df.Len())
	for i := range values {
		values[i] = value
	}
	return df.Assign(name, series.NewSeries(name, values))
}

// AssignFunc adds or replaces a column computed from each row map.
func (df *DataFrame) AssignFunc(name string, fn func(row map[string]any) any) (*DataFrame, error) {
	records := df.ToRecords()
	values := make([]any, len(records))
	for i, rec := range records {
		values[i] = fn(rec)
	}
	return df.Assign(name, series.NewSeries(name, values))
}

// AssignExpr adds or replaces a column computed from an expression:
//
//	df.AssignExpr("total", pd.Col("price").Mul(pd.Col("qty")))
//
// Typed columns run through the columnar engine (v0.4), producing a
// typed result column without boxing; otherwise the row fallback runs.
func (df *DataFrame) AssignExpr(name string, e expr.Expr) (*DataFrame, error) {
	if out, ok, err := df.assignColumnar(name, e); ok || err != nil {
		return out, err
	}
	return df.assignExprRows(name, e)
}

// assignExprRows is the row-map fallback evaluator (pre-v0.4 behavior).
func (df *DataFrame) assignExprRows(name string, e expr.Expr) (*DataFrame, error) {
	records := df.ToRecords()
	values := make([]any, len(records))
	for i, rec := range records {
		v, err := e.Eval(rec)
		if err != nil {
			return nil, fmt.Errorf("evaluating %s at row %d: %w", e, i, err)
		}
		values[i] = v
	}
	return df.Assign(name, series.NewSeries(name, values))
}

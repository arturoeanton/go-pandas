package dataframe

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/series"
)

// GroupBy holds a deferred grouping of a frame by key columns, like
// df.groupby(...).
type GroupBy struct {
	df     *DataFrame
	keys   []string
	dropNA bool
	sort   bool
	err    error
}

// GroupByOption customizes grouping.
type GroupByOption func(*GroupBy)

// GroupDropNA drops rows whose key values are missing (default true).
func GroupDropNA(v bool) GroupByOption { return func(g *GroupBy) { g.dropNA = v } }

// GroupSort sorts groups by key (default true, like pandas).
func GroupSort(v bool) GroupByOption { return func(g *GroupBy) { g.sort = v } }

// GroupBy groups the frame by one or more key columns.
func (df *DataFrame) GroupBy(keys ...string) *GroupBy {
	gb := &GroupBy{df: df, keys: keys, dropNA: true, sort: true}
	for _, k := range keys {
		if _, ok := df.byName[k]; !ok {
			gb.err = fmt.Errorf("%w: %s", errs.ErrColumnNotFound, k)
		}
	}
	return gb
}

// GroupByOpts is GroupBy with options.
func (df *DataFrame) GroupByOpts(opts []GroupByOption, keys ...string) *GroupBy {
	gb := df.GroupBy(keys...)
	for _, f := range opts {
		f(gb)
	}
	return gb
}

// group is one bucket: the key values and its member row positions.
type group struct {
	keyValues []any
	rows      []int
}

// buildGroups hash-groups rows by key tuple, preserving first-seen order.
func (gb *GroupBy) buildGroups() ([]group, error) {
	if gb.err != nil {
		return nil, gb.err
	}
	keyCols := make([][]any, len(gb.keys))
	for k, name := range gb.keys {
		c, _ := gb.df.Col(name)
		keyCols[k] = c.Values()
	}
	byKey := make(map[string]int)
	var groups []group
	for i := 0; i < gb.df.Len(); i++ {
		var sb strings.Builder
		hasNA := false
		keyValues := make([]any, len(gb.keys))
		for k := range gb.keys {
			v := keyCols[k][i]
			if dtype.IsNA(v) {
				hasNA = true
			}
			keyValues[k] = v
			sb.WriteString(fmt.Sprintf("%v\x00", v))
		}
		if hasNA && gb.dropNA {
			continue
		}
		key := sb.String()
		gi, ok := byKey[key]
		if !ok {
			gi = len(groups)
			byKey[key] = gi
			groups = append(groups, group{keyValues: keyValues})
		}
		groups[gi].rows = append(groups[gi].rows, i)
	}
	if gb.sort {
		sort.SliceStable(groups, func(a, b int) bool {
			for k := range gb.keys {
				c, ok := expr.CompareValues(groups[a].keyValues[k], groups[b].keyValues[k])
				if !ok || c == 0 {
					continue
				}
				return c < 0
			}
			return false
		})
	}
	return groups, nil
}

// valueColumns resolves the aggregation targets: the requested columns, or
// every non-key column (numeric-only when numericOnly).
func (gb *GroupBy) valueColumns(requested []string, numericOnly bool) ([]*series.Series, error) {
	isKey := make(map[string]bool, len(gb.keys))
	for _, k := range gb.keys {
		isKey[k] = true
	}
	if len(requested) > 0 {
		out := make([]*series.Series, len(requested))
		for i, name := range requested {
			c, err := gb.df.Col(name)
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	}
	var out []*series.Series
	for _, c := range gb.df.columns {
		if isKey[c.Name()] {
			continue
		}
		if numericOnly && !dtype.IsNumeric(c.DType()) && !dtype.IsBool(c.DType()) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// aggValue computes one named aggregation over the group's slice of a
// column.
func aggValue(c *series.Series, rows []int, agg string) (any, error) {
	sub, err := c.Take(rows)
	if err != nil {
		return nil, err
	}
	switch agg {
	case "count":
		return sub.Count(), nil
	case "size":
		return sub.Len(), nil
	case "sum":
		return sub.Sum()
	case "mean":
		return sub.Mean()
	case "median":
		return sub.Median()
	case "min":
		return sub.Min()
	case "max":
		return sub.Max()
	case "var":
		return sub.Var()
	case "std":
		return sub.Std()
	case "first":
		for i := 0; i < sub.Len(); i++ {
			if v, _ := sub.At(i); !dtype.IsNA(v) {
				return v, nil
			}
		}
		return nil, nil
	case "last":
		for i := sub.Len() - 1; i >= 0; i-- {
			if v, _ := sub.At(i); !dtype.IsNA(v) {
				return v, nil
			}
		}
		return nil, nil
	case "nunique":
		return sub.NUnique(true), nil
	}
	return nil, fmt.Errorf("%w: aggregation %q", errs.ErrInvalidOperation, agg)
}

// aggSpec is one output column: source column, aggregation and output name.
type aggSpec struct {
	column  string
	agg     string
	outName string
}

// runAgg executes a list of aggregation specs and assembles the result
// frame: key columns first, then one column per spec.
func (gb *GroupBy) runAgg(specs []aggSpec) (*DataFrame, error) {
	groups, err := gb.buildGroups()
	if err != nil {
		return nil, err
	}
	keyData := make([][]any, len(gb.keys))
	for k := range keyData {
		keyData[k] = make([]any, len(groups))
	}
	outData := make([][]any, len(specs))
	for j := range outData {
		outData[j] = make([]any, len(groups))
	}
	for gi, g := range groups {
		for k := range gb.keys {
			keyData[k][gi] = g.keyValues[k]
		}
		for j, spec := range specs {
			c, err := gb.df.Col(spec.column)
			if err != nil {
				return nil, err
			}
			v, err := aggValue(c, g.rows, spec.agg)
			if err != nil {
				return nil, fmt.Errorf("aggregating %s(%s): %w", spec.agg, spec.column, err)
			}
			outData[j][gi] = v
		}
	}
	var cols []*series.Series
	for k, name := range gb.keys {
		cols = append(cols, series.NewSeries(name, keyData[k]))
	}
	for j, spec := range specs {
		cols = append(cols, series.NewSeries(spec.outName, outData[j]))
	}
	return newFrame(cols, nil)
}

// simpleAgg applies one aggregation to the given (or all applicable)
// columns keeping original column names, like gb.mean().
func (gb *GroupBy) simpleAgg(agg string, columns []string) (*DataFrame, error) {
	numericOnly := agg == "sum" || agg == "mean" || agg == "median" || agg == "var" || agg == "std"
	targets, err := gb.valueColumns(columns, numericOnly)
	if err != nil {
		return nil, err
	}
	specs := make([]aggSpec, len(targets))
	for i, c := range targets {
		specs[i] = aggSpec{column: c.Name(), agg: agg, outName: c.Name()}
	}
	return gb.runAgg(specs)
}

// Count counts non-missing values per group.
func (gb *GroupBy) Count(columns ...string) (*DataFrame, error) {
	return gb.simpleAgg("count", columns)
}

// Size returns the number of rows per group in a "size" column.
func (gb *GroupBy) Size() (*DataFrame, error) {
	groups, err := gb.buildGroups()
	if err != nil {
		return nil, err
	}
	keyData := make([][]any, len(gb.keys))
	for k := range keyData {
		keyData[k] = make([]any, len(groups))
	}
	sizes := make([]any, len(groups))
	for gi, g := range groups {
		for k := range gb.keys {
			keyData[k][gi] = g.keyValues[k]
		}
		sizes[gi] = len(g.rows)
	}
	var cols []*series.Series
	for k, name := range gb.keys {
		cols = append(cols, series.NewSeries(name, keyData[k]))
	}
	cols = append(cols, series.NewSeries("size", sizes))
	return newFrame(cols, nil)
}

// Sum sums numeric columns per group.
func (gb *GroupBy) Sum(columns ...string) (*DataFrame, error) { return gb.simpleAgg("sum", columns) }

// Mean averages numeric columns per group.
func (gb *GroupBy) Mean(columns ...string) (*DataFrame, error) { return gb.simpleAgg("mean", columns) }

// Median computes per-group medians.
func (gb *GroupBy) Median(columns ...string) (*DataFrame, error) {
	return gb.simpleAgg("median", columns)
}

// Min computes per-group minima.
func (gb *GroupBy) Min(columns ...string) (*DataFrame, error) { return gb.simpleAgg("min", columns) }

// Max computes per-group maxima.
func (gb *GroupBy) Max(columns ...string) (*DataFrame, error) { return gb.simpleAgg("max", columns) }

// Var computes per-group sample variances.
func (gb *GroupBy) Var(columns ...string) (*DataFrame, error) { return gb.simpleAgg("var", columns) }

// Std computes per-group sample standard deviations.
func (gb *GroupBy) Std(columns ...string) (*DataFrame, error) { return gb.simpleAgg("std", columns) }

// NUnique counts distinct non-NA values per group.
func (gb *GroupBy) NUnique(columns ...string) (*DataFrame, error) {
	return gb.simpleAgg("nunique", columns)
}

// First takes the first non-missing value per group.
func (gb *GroupBy) First(columns ...string) (*DataFrame, error) {
	return gb.simpleAgg("first", columns)
}

// Last takes the last non-missing value per group.
func (gb *GroupBy) Last(columns ...string) (*DataFrame, error) { return gb.simpleAgg("last", columns) }

// Agg applies one aggregation per column; output columns are named
// column_agg, e.g. salary_mean:
//
//	gb.Agg(map[string]string{"salary": "mean", "age": "max"})
func (gb *GroupBy) Agg(spec map[string]string) (*DataFrame, error) {
	names := make([]string, 0, len(spec))
	for name := range spec {
		names = append(names, name)
	}
	sort.Strings(names)
	specs := make([]aggSpec, 0, len(spec))
	for _, name := range names {
		agg := spec[name]
		specs = append(specs, aggSpec{column: name, agg: agg, outName: name + "_" + agg})
	}
	return gb.runAgg(specs)
}

// AggList applies several aggregations per column:
//
//	gb.AggList(map[string][]string{"salary": {"mean", "max"}})
func (gb *GroupBy) AggList(spec map[string][]string) (*DataFrame, error) {
	names := make([]string, 0, len(spec))
	for name := range spec {
		names = append(names, name)
	}
	sort.Strings(names)
	var specs []aggSpec
	for _, name := range names {
		for _, agg := range spec[name] {
			specs = append(specs, aggSpec{column: name, agg: agg, outName: name + "_" + agg})
		}
	}
	return gb.runAgg(specs)
}

// Apply runs a function over each group's sub-frame and vertically
// concatenates the results.
func (gb *GroupBy) Apply(fn func(*DataFrame) (*DataFrame, error)) (*DataFrame, error) {
	groups, err := gb.buildGroups()
	if err != nil {
		return nil, err
	}
	var frames []*DataFrame
	for _, g := range groups {
		sub, err := gb.df.Take(g.rows)
		if err != nil {
			return nil, err
		}
		out, err := fn(sub)
		if err != nil {
			return nil, err
		}
		if out != nil {
			frames = append(frames, out)
		}
	}
	if len(frames) == 0 {
		return newFrame(nil, nil)
	}
	return Concat(frames, ConcatIgnoreIndex(true))
}

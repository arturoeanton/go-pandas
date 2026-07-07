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

// MeltOptions mirrors pd.melt keyword arguments.
type MeltOptions struct {
	IDVars    []string
	ValueVars []string
	VarName   string
	ValueName string
}

// Melt unpivots a frame from wide to long format.
func (df *DataFrame) Melt(opts MeltOptions) (*DataFrame, error) {
	varName := opts.VarName
	if varName == "" {
		varName = "variable"
	}
	valueName := opts.ValueName
	if valueName == "" {
		valueName = "value"
	}
	isID := make(map[string]bool, len(opts.IDVars))
	for _, name := range opts.IDVars {
		if _, ok := df.byName[name]; !ok {
			return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, name)
		}
		isID[name] = true
	}
	valueVars := opts.ValueVars
	if len(valueVars) == 0 {
		for _, c := range df.columns {
			if !isID[c.Name()] {
				valueVars = append(valueVars, c.Name())
			}
		}
	}
	n := df.Len()
	idData := make([][]any, len(opts.IDVars))
	for k, name := range opts.IDVars {
		c, _ := df.Col(name)
		idData[k] = c.Values()
	}
	var outID = make([][]any, len(opts.IDVars))
	var outVar, outValue []any
	for _, vv := range valueVars {
		c, err := df.Col(vv)
		if err != nil {
			return nil, err
		}
		values := c.Values()
		for i := 0; i < n; i++ {
			for k := range opts.IDVars {
				outID[k] = append(outID[k], idData[k][i])
			}
			outVar = append(outVar, vv)
			outValue = append(outValue, values[i])
		}
	}
	var cols []*series.Series
	for k, name := range opts.IDVars {
		cols = append(cols, series.NewSeries(name, outID[k]))
	}
	cols = append(cols, series.NewSeries(varName, outVar))
	cols = append(cols, series.NewSeries(valueName, outValue))
	return newFrame(cols, nil)
}

// PivotOptions mirrors pd.pivot keyword arguments.
type PivotOptions struct {
	Index   string
	Columns string
	Values  string
}

// Pivot reshapes long to wide: unique Index values become rows, unique
// Columns values become columns. Duplicate (index, column) pairs are an
// error, like pandas.
func (df *DataFrame) Pivot(opts PivotOptions) (*DataFrame, error) {
	return df.pivotWith(opts.Index, opts.Columns, opts.Values, "", nil)
}

// PivotTableOptions mirrors pd.pivot_table (single value column and single
// aggregation in v0.1).
type PivotTableOptions struct {
	Index   []string
	Columns []string
	Values  []string
	AggFunc string
	// AggFuncs runs several aggregations at once (v0.10); when set it
	// overrides AggFunc.
	AggFuncs  []string
	FillValue any
}

// PivotTable aggregates duplicate cells (default mean). Since v0.10 it
// supports multiple values, multiple aggfuncs, multi-key index and
// (one) columns dimension, built on the typed GroupBy engine plus
// Unstack. Output column names are deterministic (documented in
// docs/reshape rules): parts joined with "_" in value, agg, label
// order — the value part appears only with several values, the agg
// part only with several aggfuncs, the label part only with a columns
// dimension. The single value/agg/columns case keeps the historical
// output shape (index keys as regular columns, label columns).
func (df *DataFrame) PivotTable(opts PivotTableOptions) (*DataFrame, error) {
	aggs := opts.AggFuncs
	if len(aggs) == 0 {
		agg := opts.AggFunc
		if agg == "" {
			agg = "mean"
		}
		aggs = []string{agg}
	}
	if len(opts.Index) == 0 || len(opts.Values) == 0 {
		return nil, fmt.Errorf("%w: PivotTable needs Index and Values", errs.ErrInvalidOperation)
	}
	if len(opts.Columns) > 1 {
		return nil, errs.NotImplemented("DataFrame.PivotTable with multiple Columns keys")
	}
	// Historical fast path: 1 index x 1 columns x 1 value x 1 agg.
	if len(opts.Index) == 1 && len(opts.Columns) == 1 && len(opts.Values) == 1 && len(aggs) == 1 {
		return df.pivotWith(opts.Index[0], opts.Columns[0], opts.Values[0], aggs[0], opts.FillValue)
	}
	return df.pivotTableMulti(opts, aggs)
}

// pivotOutName applies the deterministic naming rule.
func pivotOutName(value, agg, label string, manyValues, manyAggs, hasColumns bool) string {
	var parts []string
	if manyValues {
		parts = append(parts, value)
	}
	if manyAggs {
		parts = append(parts, agg)
	}
	if hasColumns {
		parts = append(parts, label)
	}
	if len(parts) == 0 {
		return value
	}
	return strings.Join(parts, "_")
}

// pivotTableMulti is the v0.10 engine: one typed groupby over
// index+columns keys, then (with a columns dimension) one unstack per
// value x agg output.
func (df *DataFrame) pivotTableMulti(opts PivotTableOptions, aggs []string) (*DataFrame, error) {
	manyValues := len(opts.Values) > 1
	manyAggs := len(aggs) > 1
	hasColumns := len(opts.Columns) == 1

	keys := append(append([]string(nil), opts.Index...), opts.Columns...)
	var specs []aggSpec
	for _, v := range opts.Values {
		for _, a := range aggs {
			specs = append(specs, aggSpec{column: v, agg: a, outName: v + "\x00" + a})
		}
	}
	flat, err := df.GroupBy(keys...).runAgg(specs)
	if err != nil {
		return nil, err
	}

	if !hasColumns {
		// Index keys stay regular columns; rename aggregate outputs.
		cols := make([]*series.Series, 0, len(flat.columns))
		for _, c := range flat.columns {
			name := c.Name()
			if v, a, ok := strings.Cut(name, "\x00"); ok {
				name = pivotOutName(v, a, "", manyValues, manyAggs, false)
				c = c.Rename(name)
			}
			cols = append(cols, c)
		}
		return newFrame(cols, flat.index)
	}

	indexed, err := flat.SetIndex(keys...)
	if err != nil {
		return nil, err
	}
	var out *DataFrame
	for _, spec := range specs {
		s, err := indexed.Col(spec.outName)
		if err != nil {
			return nil, err
		}
		part, err := UnstackSeries(s)
		if err != nil {
			return nil, err
		}
		renamed := make([]*series.Series, len(part.columns))
		for j, pc := range part.columns {
			name := pivotOutName(spec.column, spec.agg, pc.Name(), manyValues, manyAggs, true)
			renamed[j] = pc.Rename(name)
		}
		partFrame, err := newFrame(renamed, part.index)
		if err != nil {
			return nil, err
		}
		if out == nil {
			out = partFrame
			continue
		}
		if out, err = concatColumns([]*DataFrame{out, partFrame}); err != nil {
			return nil, err
		}
	}
	// Historical shape: index keys as regular columns + RangeIndex.
	out = out.ResetIndex()
	if opts.FillValue != nil {
		isKey := make(map[string]bool, len(opts.Index))
		for _, k := range opts.Index {
			isKey[k] = true
		}
		cols := make([]*series.Series, len(out.columns))
		for j, c := range out.columns {
			if isKey[c.Name()] {
				cols[j] = c
				continue
			}
			cols[j] = c.FillNA(opts.FillValue)
		}
		return newFrame(cols, out.index)
	}
	return out, nil
}

// pivotWith implements pivot (agg == "": duplicates are an error) and
// pivot_table (agg != "": duplicates are aggregated).
func (df *DataFrame) pivotWith(indexCol, columnsCol, valuesCol, agg string, fillValue any) (*DataFrame, error) {
	idxSeries, err := df.Col(indexCol)
	if err != nil {
		return nil, err
	}
	colSeries, err := df.Col(columnsCol)
	if err != nil {
		return nil, err
	}
	valSeries, err := df.Col(valuesCol)
	if err != nil {
		return nil, err
	}
	idxValues := idxSeries.Values()
	colValues := colSeries.Values()

	var rowKeys, colKeys []any
	rowPos := map[string]int{}
	colPos := map[string]int{}
	for i := 0; i < df.Len(); i++ {
		rk := fmt.Sprintf("%v", idxValues[i])
		ck := fmt.Sprintf("%v", colValues[i])
		if _, ok := rowPos[rk]; !ok {
			rowPos[rk] = 0
			rowKeys = append(rowKeys, idxValues[i])
		}
		if _, ok := colPos[ck]; !ok {
			colPos[ck] = 0
			colKeys = append(colKeys, colValues[i])
		}
	}
	// pandas pivot sorts both the index and the column labels.
	sortAnyValues(rowKeys)
	sortAnyValues(colKeys)
	for i, k := range rowKeys {
		rowPos[fmt.Sprintf("%v", k)] = i
	}
	for i, k := range colKeys {
		colPos[fmt.Sprintf("%v", k)] = i
	}
	cells := map[[2]int][]int{}
	for i := 0; i < df.Len(); i++ {
		ri := rowPos[fmt.Sprintf("%v", idxValues[i])]
		ci := colPos[fmt.Sprintf("%v", colValues[i])]
		cells[[2]int{ri, ci}] = append(cells[[2]int{ri, ci}], i)
	}

	var cols []*series.Series
	cols = append(cols, series.NewSeries(indexCol, rowKeys))
	for ci, ckAny := range colKeys {
		values := make([]any, len(rowKeys))
		for ri := range rowKeys {
			rows := cells[[2]int{ri, ci}]
			switch {
			case len(rows) == 0:
				values[ri] = fillValue
			case agg == "":
				if len(rows) > 1 {
					return nil, fmt.Errorf("%w: duplicate entries for index=%v columns=%v; use PivotTable", errs.ErrInvalidOperation, rowKeys[ri], ckAny)
				}
				v, _ := valSeries.At(rows[0])
				values[ri] = v
			default:
				v, err := aggValue(valSeries, rows, agg)
				if err != nil {
					return nil, err
				}
				if dtype.IsNA(v) && fillValue != nil {
					v = fillValue
				}
				values[ri] = v
			}
		}
		cols = append(cols, series.NewSeries(fmt.Sprint(ckAny), values))
	}
	return newFrame(cols, nil)
}

// sortAnyValues orders labels with the shared value comparator (numbers,
// strings, times); incomparable values keep their relative order.
func sortAnyValues(values []any) {
	sort.SliceStable(values, func(a, b int) bool {
		c, ok := expr.CompareValues(values[a], values[b])
		return ok && c < 0
	})
}

// Stack and Unstack live in stack.go (real engines since v0.10; the
// v0.1 placeholders returned ErrNotImplemented — Stack now returns the
// pandas-like Series).

// Resample lives in resample.go (real engine since v0.9; the v0.1
// placeholder returned ErrNotImplemented and a second error value —
// errors now surface from the aggregation calls, like GroupBy).

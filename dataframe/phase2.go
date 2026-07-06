package dataframe

import (
	"fmt"
	"math"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// Axis constants for APIs that operate along rows or columns.
const (
	AxisRows    = 0
	AxisColumns = 1
)

// Reindex conforms the frame to a new row index: known labels keep their
// row, new labels get NA rows.
func (df *DataFrame) Reindex(idx index.Index) (*DataFrame, error) {
	if idx == nil {
		return nil, fmt.Errorf("%w: nil index", errs.ErrInvalidIndex)
	}
	pos := make([]int, idx.Len())
	for i := 0; i < idx.Len(); i++ {
		if p, ok := df.index.Pos(idx.At(i)); ok {
			pos[i] = p
		} else {
			pos[i] = -1
		}
	}
	taken, err := df.Take(pos)
	if err != nil {
		return nil, err
	}
	cols := make([]*series.Series, len(taken.columns))
	for i, c := range taken.columns {
		cols[i] = c.WithIndexed(idx)
	}
	return newFrame(cols, idx)
}

// ReindexColumns conforms the frame to a new column list: unknown columns
// are created full of NA, like df.reindex(columns=[...]).
func (df *DataFrame) ReindexColumns(columns ...string) (*DataFrame, error) {
	cols := make([]*series.Series, len(columns))
	for i, name := range columns {
		if j, ok := df.byName[name]; ok {
			cols[i] = df.columns[j].Copy()
			continue
		}
		values := make([]any, df.Len())
		cols[i] = series.NewSeries(name, values, series.WithIndex(df.index))
	}
	return newFrame(cols, df.index.Clone())
}

// rowKey builds a hashable key from the subset columns of a row.
func rowKey(cols [][]any, row int) string {
	var sb strings.Builder
	for _, col := range cols {
		v := col[row]
		if dtype.IsNA(v) {
			sb.WriteString("\x00<NA>\x00")
			continue
		}
		if f, ok := dtype.AsFloat(v); ok {
			sb.WriteString(fmt.Sprintf("%v\x00", f))
		} else {
			sb.WriteString(fmt.Sprintf("%v\x00", v))
		}
	}
	return sb.String()
}

func (df *DataFrame) subsetValues(subset []string) ([][]any, error) {
	if len(subset) == 0 {
		subset = df.Columns()
	}
	cols := make([][]any, len(subset))
	for i, name := range subset {
		c, err := df.Col(name)
		if err != nil {
			return nil, err
		}
		cols[i] = c.Values()
	}
	return cols, nil
}

// Duplicated marks rows that repeat an earlier row (keep="first"), like
// df.duplicated(subset).
func (df *DataFrame) Duplicated(subset ...string) (*series.Series, error) {
	cols, err := df.subsetValues(subset)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	values := make([]any, df.Len())
	for i := 0; i < df.Len(); i++ {
		k := rowKey(cols, i)
		values[i] = seen[k]
		seen[k] = true
	}
	return series.NewSeries("duplicated", values, series.WithIndex(df.index)), nil
}

// DropDuplicates keeps the first occurrence of each distinct row, like
// df.drop_duplicates(subset).
func (df *DataFrame) DropDuplicates(subset ...string) (*DataFrame, error) {
	dup, err := df.Duplicated(subset...)
	if err != nil {
		return nil, err
	}
	mask := dup.AsMask()
	var pos []int
	for i, isDup := range mask {
		if !isDup {
			pos = append(pos, i)
		}
	}
	return df.Take(pos)
}

// NUnique counts distinct non-NA values per column (all columns, or the
// given subset), like df.nunique().
func (df *DataFrame) NUnique(columns ...string) map[string]int {
	if len(columns) == 0 {
		columns = df.Columns()
	}
	out := make(map[string]int, len(columns))
	for _, name := range columns {
		if i, ok := df.byName[name]; ok {
			out[name] = df.columns[i].NUnique(true)
		}
	}
	return out
}

// ValueCounts counts distinct row combinations of the given columns (all
// columns when omitted), sorted by count descending, like
// df.value_counts().
func (df *DataFrame) ValueCounts(columns ...string) (*DataFrame, error) {
	if len(columns) == 0 {
		columns = df.Columns()
	}
	gb := df.GroupBy(columns...)
	sized, err := gb.Size()
	if err != nil {
		return nil, err
	}
	renamed, err := sized.Rename(map[string]string{"size": "count"})
	if err != nil {
		return nil, err
	}
	return renamed.SortValues("count", false)
}

// pairwiseStat computes a statistic over pairwise-complete observations of
// two float columns.
func pairwiseStat(x, y []float64, xok, yok []bool, f func(x, y []float64) float64) float64 {
	var xs, ys []float64
	for i := range x {
		if xok[i] && yok[i] {
			xs = append(xs, x[i])
			ys = append(ys, y[i])
		}
	}
	if len(xs) < 2 {
		return math.NaN()
	}
	return f(xs, ys)
}

func covariance(x, y []float64) float64 {
	n := float64(len(x))
	mx, my := 0.0, 0.0
	for i := range x {
		mx += x[i]
		my += y[i]
	}
	mx /= n
	my /= n
	acc := 0.0
	for i := range x {
		acc += (x[i] - mx) * (y[i] - my)
	}
	return acc / (n - 1) // ddof=1, pandas default
}

func correlation(x, y []float64) float64 {
	cxy := covariance(x, y)
	cxx := covariance(x, x)
	cyy := covariance(y, y)
	return cxy / math.Sqrt(cxx*cyy)
}

func (df *DataFrame) pairwiseMatrix(columns []string, f func(x, y []float64) float64) (*DataFrame, error) {
	var targets []*series.Series
	if len(columns) > 0 {
		for _, name := range columns {
			c, err := df.Col(name)
			if err != nil {
				return nil, err
			}
			targets = append(targets, c)
		}
	} else {
		targets = df.numericColumns()
	}
	n := len(targets)
	floats := make([][]float64, n)
	present := make([][]bool, n)
	names := make([]string, n)
	for i, c := range targets {
		names[i] = c.Name()
		floats[i] = make([]float64, c.Len())
		present[i] = make([]bool, c.Len())
		for j, v := range c.Values() {
			if f, ok := dtype.AsFloat(v); ok && !dtype.IsNA(v) {
				floats[i][j] = f
				present[i][j] = true
			}
		}
	}
	idx := index.NewStringIndex(names)
	cols := make([]*series.Series, n)
	for j := 0; j < n; j++ {
		values := make([]any, n)
		for i := 0; i < n; i++ {
			values[i] = pairwiseStat(floats[i], floats[j], present[i], present[j], f)
		}
		cols[j] = series.NewSeries(names[j], values, series.WithIndex(idx))
	}
	return newFrame(cols, idx)
}

// Corr returns the pairwise Pearson correlation matrix of numeric columns
// (pairwise-complete observations), like df.corr().
func (df *DataFrame) Corr(columns ...string) (*DataFrame, error) {
	return df.pairwiseMatrix(columns, correlation)
}

// Cov returns the pairwise sample covariance matrix (ddof=1), like
// df.cov().
func (df *DataFrame) Cov(columns ...string) (*DataFrame, error) {
	return df.pairwiseMatrix(columns, covariance)
}

// mapNumericColumns applies a Series transform to numeric columns (all, or
// the given subset), keeping other columns unchanged.
func (df *DataFrame) mapNumericColumns(columns []string, f func(c *series.Series) (*series.Series, error)) (*DataFrame, error) {
	target := make(map[string]bool)
	if len(columns) > 0 {
		for _, name := range columns {
			if _, ok := df.byName[name]; !ok {
				return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, name)
			}
			target[name] = true
		}
	} else {
		for _, c := range df.numericColumns() {
			target[c.Name()] = true
		}
	}
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		if !target[c.Name()] {
			cols[i] = c.Copy()
			continue
		}
		out, err := f(c)
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", c.Name(), err)
		}
		cols[i] = out
	}
	return newFrame(cols, df.index.Clone())
}

// Clip limits numeric values to [lower, upper], like df.clip().
func (df *DataFrame) Clip(lower, upper float64, columns ...string) (*DataFrame, error) {
	return df.mapNumericColumns(columns, func(c *series.Series) (*series.Series, error) {
		return c.Clip(lower, upper)
	})
}

// Round rounds numeric values to the given decimals, like df.round().
func (df *DataFrame) Round(decimals int, columns ...string) (*DataFrame, error) {
	return df.mapNumericColumns(columns, func(c *series.Series) (*series.Series, error) {
		return c.Round(decimals)
	})
}

// Abs takes absolute values of numeric columns, like df.abs().
func (df *DataFrame) Abs(columns ...string) (*DataFrame, error) {
	return df.mapNumericColumns(columns, func(c *series.Series) (*series.Series, error) {
		return c.Abs()
	})
}

// Astype converts columns to new dtypes, like df.astype({...}).
func (df *DataFrame) Astype(types map[string]dtype.DType) (*DataFrame, error) {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		if dt, ok := types[c.Name()]; ok {
			out, err := c.Astype(dt)
			if err != nil {
				return nil, fmt.Errorf("column %q: %w", c.Name(), err)
			}
			cols[i] = out
		} else {
			cols[i] = c.Copy()
		}
	}
	return newFrame(cols, df.index.Clone())
}

// SelectDTypesOptions configures SelectDTypes.
type SelectDTypesOptions struct {
	Include []dtype.DType
	Exclude []dtype.DType
}

// SelectDTypesOption mutates SelectDTypesOptions.
type SelectDTypesOption func(*SelectDTypesOptions)

// Include selects columns whose dtype matches any of the given dtypes
// (dtype.Number matches every numeric dtype).
func Include(dts ...dtype.DType) SelectDTypesOption {
	return func(o *SelectDTypesOptions) { o.Include = append(o.Include, dts...) }
}

// Exclude removes columns whose dtype matches any of the given dtypes.
func Exclude(dts ...dtype.DType) SelectDTypesOption {
	return func(o *SelectDTypesOptions) { o.Exclude = append(o.Exclude, dts...) }
}

// SelectDTypes filters columns by dtype, like df.select_dtypes():
//
//	numeric, _ := df.SelectDTypes(pd.Include(pd.Number))
func (df *DataFrame) SelectDTypes(opts ...SelectDTypesOption) (*DataFrame, error) {
	var o SelectDTypesOptions
	for _, f := range opts {
		f(&o)
	}
	if len(o.Include) == 0 && len(o.Exclude) == 0 {
		return nil, fmt.Errorf("%w: SelectDTypes needs Include or Exclude", errs.ErrInvalidOperation)
	}
	matchesAny := func(sel []dtype.DType, t dtype.DType) bool {
		for _, s := range sel {
			if dtype.Matches(s, t) {
				return true
			}
		}
		return false
	}
	var cols []*series.Series
	for _, c := range df.columns {
		if len(o.Include) > 0 && !matchesAny(o.Include, c.DType()) {
			continue
		}
		if len(o.Exclude) > 0 && matchesAny(o.Exclude, c.DType()) {
			continue
		}
		cols = append(cols, c.Copy())
	}
	return newFrame(cols, df.index.Clone())
}

// ReplaceNA fills missing values in every column with one value.
func (df *DataFrame) ReplaceNA(v any) *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.FillNA(v)
	}
	out, _ := newFrame(cols, df.index.Clone())
	return out
}

// ExpandingDataFrame applies expanding windows to numeric columns.
type ExpandingDataFrame struct {
	df         *DataFrame
	minPeriods int
}

// Expanding creates an expanding window over the frame's numeric columns.
func (df *DataFrame) Expanding(minPeriods ...int) *ExpandingDataFrame {
	mp := 1
	if len(minPeriods) > 0 && minPeriods[0] > 0 {
		mp = minPeriods[0]
	}
	return &ExpandingDataFrame{df: df, minPeriods: mp}
}

func (e *ExpandingDataFrame) aggregate(f func(es *series.ExpandingSeries) (*series.Series, error)) (*DataFrame, error) {
	var cols []*series.Series
	for _, c := range e.df.columns {
		if !dtype.IsNumeric(c.DType()) && !dtype.IsBool(c.DType()) {
			continue
		}
		out, err := f(c.Expanding(e.minPeriods))
		if err != nil {
			return nil, err
		}
		cols = append(cols, out)
	}
	return newFrame(cols, e.df.index.Clone())
}

// Sum returns per-column expanding sums.
func (e *ExpandingDataFrame) Sum() (*DataFrame, error) {
	return e.aggregate(func(es *series.ExpandingSeries) (*series.Series, error) { return es.Sum() })
}

// Mean returns per-column expanding means.
func (e *ExpandingDataFrame) Mean() (*DataFrame, error) {
	return e.aggregate(func(es *series.ExpandingSeries) (*series.Series, error) { return es.Mean() })
}

// Min returns per-column expanding minima.
func (e *ExpandingDataFrame) Min() (*DataFrame, error) {
	return e.aggregate(func(es *series.ExpandingSeries) (*series.Series, error) { return es.Min() })
}

// Max returns per-column expanding maxima.
func (e *ExpandingDataFrame) Max() (*DataFrame, error) {
	return e.aggregate(func(es *series.ExpandingSeries) (*series.Series, error) { return es.Max() })
}

// Median returns per-column rolling medians.
func (r *RollingDataFrame) Median() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Median() })
}

// Var returns per-column rolling sample variances.
func (r *RollingDataFrame) Var() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Var() })
}

// Count returns per-column rolling counts.
func (r *RollingDataFrame) Count() (*DataFrame, error) {
	return r.aggregate(func(rs *series.RollingSeries) (*series.Series, error) { return rs.Count() })
}

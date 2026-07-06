// Package pandas (import as pd) is the root of go-pandas: pandas and
// NumPy style data analysis for Go.
//
//	import pd "github.com/arturoeanton/go-pandas"
//
//	df, _ := pd.DataFrameFromRecords(records)
//	adults, _ := df.Where(pd.Col("age").Gt(18))
//	a := pd.Array([]float64{1, 2, 3}).AddScalar(10)
//
// If you know pandas and NumPy, the concepts should transfer immediately.
package pandas

import (
	stdio "io"
	"time"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
	pdio "github.com/arturoeanton/go-pandas/io"
	"github.com/arturoeanton/go-pandas/ndarray"
	"github.com/arturoeanton/go-pandas/series"
)

// Core type re-exports.
type (
	DataFrame    = dataframe.DataFrame
	Series       = series.Series
	NDArray      = ndarray.NDArray
	BoolArray    = ndarray.BoolArray
	Index        = index.Index
	DType        = dtype.DType
	GroupBy      = dataframe.GroupBy
	MergeOptions = dataframe.MergeOptions
	JoinOptions  = dataframe.JoinOptions
	CSVOptions   = pdio.CSVOptions
	JSONOptions  = pdio.JSONOptions
	Expr         = expr.Expr
	Predicate    = expr.Predicate

	SeriesOption    = series.SeriesOption
	DataFrameOption = dataframe.DataFrameOption
	CSVOption       = pdio.CSVOption
	JSONOption      = pdio.JSONOption
	ConcatOption    = dataframe.ConcatOption
	GroupByOption   = dataframe.GroupByOption
	RollingOption   = series.RollingOption
	ReduceOption    = series.ReduceOption
	DropNAOption    = dataframe.DropNAOption
	SliceSpec       = ndarray.SliceSpec

	MeltOptions       = dataframe.MeltOptions
	PivotOptions      = dataframe.PivotOptions
	PivotTableOptions = dataframe.PivotTableOptions
	ConcatOptions     = dataframe.ConcatOptions
	SampleOption      = dataframe.SampleOption
	ValueCountOption  = series.ValueCountOption
	StringAccessor    = series.StringAccessor
	DatetimeAccessor  = series.DatetimeAccessor
	RollingSeries     = series.RollingSeries
	RollingDataFrame  = dataframe.RollingDataFrame
	ExpandingSeries   = series.ExpandingSeries
	LocIndexer        = dataframe.LocIndexer
	ILocIndexer       = dataframe.ILocIndexer
	MultiIndex        = index.MultiIndex
	NullableDType     = dtype.NullableDType
)

// DType constant re-exports.
const (
	Bool      = dtype.Bool
	Int       = dtype.Int
	Int8      = dtype.Int8
	Int16     = dtype.Int16
	Int32     = dtype.Int32
	Int64     = dtype.Int64
	UInt      = dtype.UInt
	UInt8     = dtype.UInt8
	UInt16    = dtype.UInt16
	UInt32    = dtype.UInt32
	UInt64    = dtype.UInt64
	Float32   = dtype.Float32
	Float64   = dtype.Float64
	String    = dtype.String
	Time      = dtype.Time
	Timedelta = dtype.Timedelta
	Category  = dtype.Category
	Object    = dtype.Object
)

// Series constructors -----------------------------------------------------

// NewSeries builds a series from untyped values.
func NewSeries(name string, values []any, opts ...SeriesOption) *Series {
	return series.NewSeries(name, values, opts...)
}

// SeriesOf builds a series from a typed slice.
func SeriesOf[T any](name string, values []T, opts ...SeriesOption) *Series {
	return series.SeriesOf(name, values, opts...)
}

// IntSeries, FloatSeries, StringSeries, BoolSeries, TimeSeries build typed
// series.
func IntSeries(name string, values []int) *Series       { return series.IntSeries(name, values) }
func Int64Series(name string, values []int64) *Series   { return series.Int64Series(name, values) }
func FloatSeries(name string, values []float64) *Series { return series.FloatSeries(name, values) }
func StringSeries(name string, values []string) *Series { return series.StringSeries(name, values) }
func BoolSeries(name string, values []bool) *Series     { return series.BoolSeries(name, values) }
func TimeSeries(name string, values []time.Time) *Series {
	return series.TimeSeries(name, values)
}

// WithIndex, WithDType and WithName re-export series options.
func WithIndex(idx Index) SeriesOption  { return series.WithIndex(idx) }
func WithDType(dt DType) SeriesOption   { return series.WithDType(dt) }
func WithName(name string) SeriesOption { return series.WithName(name) }

// DataFrame constructors --------------------------------------------------

// NewDataFrame builds a frame from Series columns.
func NewDataFrame(cols ...*Series) (*DataFrame, error) { return dataframe.NewDataFrame(cols...) }

// DataFrameFromRecords builds a frame from row maps.
func DataFrameFromRecords(records []map[string]any, opts ...DataFrameOption) (*DataFrame, error) {
	return dataframe.DataFrameFromRecords(records, opts...)
}

// DataFrameFromRows builds a frame from column names and row slices.
func DataFrameFromRows(columns []string, rows [][]any, opts ...DataFrameOption) (*DataFrame, error) {
	return dataframe.DataFrameFromRows(columns, rows, opts...)
}

// DataFrameFromMap builds a frame from column name -> values.
func DataFrameFromMap(data map[string][]any, opts ...DataFrameOption) (*DataFrame, error) {
	return dataframe.DataFrameFromMap(data, opts...)
}

// DataFrameFromStructs builds a frame from a slice of structs.
func DataFrameFromStructs(v any, opts ...DataFrameOption) (*DataFrame, error) {
	return dataframe.DataFrameFromStructs(v, opts...)
}

// DataFrameFromNDArray converts a 2-D array into a frame.
func DataFrameFromNDArray(a *NDArray, columns []string) (*DataFrame, error) {
	return dataframe.DataFrameFromNDArray(a, columns)
}

// WithDataFrameIndex and WithColumnOrder re-export frame options.
func WithDataFrameIndex(idx Index) DataFrameOption { return dataframe.WithDataFrameIndex(idx) }
func WithColumnOrder(columns ...string) DataFrameOption {
	return dataframe.WithColumnOrder(columns...)
}

// Index constructors ------------------------------------------------------

// NewRangeIndex builds the default positional index over [0, n).
func NewRangeIndex(n int) Index { return index.NewRangeIndex(n) }

// RangeIndexFrom builds a range index over [start, stop) with a step.
func RangeIndexFrom(start, stop, step int) Index { return index.RangeIndexFrom(start, stop, step) }

// NewStringIndex builds a label index from strings.
func NewStringIndex(values []string, name ...string) Index {
	return index.NewStringIndex(values, name...)
}

// NewDatetimeIndex builds a label index from timestamps.
func NewDatetimeIndex(values []time.Time, name ...string) Index {
	return index.NewDatetimeIndex(values, name...)
}

// NewMultiIndexFromArrays builds a hierarchical index from label arrays.
func NewMultiIndexFromArrays(arrays [][]any, names []string) (*MultiIndex, error) {
	return index.NewMultiIndexFromArrays(arrays, names)
}

// NDArray constructors ----------------------------------------------------

// Array builds a 1-D array from float64 values (np.array).
func Array(data []float64) *NDArray { return ndarray.Array(data) }

// ArrayOf builds a 1-D array from any numeric slice.
func ArrayOf[T ndarray.Number](data []T) *NDArray { return ndarray.ArrayOf(data) }

// Array2D builds a 2-D array from rows.
func Array2D(data [][]float64) (*NDArray, error) { return ndarray.Array2D(data) }

// FromSlice builds an array with an explicit shape.
func FromSlice(data []float64, shape ...int) (*NDArray, error) {
	return ndarray.FromSlice(data, shape...)
}

// MustFromSlice is FromSlice that panics on shape mismatch.
func MustFromSlice(data []float64, shape ...int) *NDArray {
	return ndarray.MustFromSlice(data, shape...)
}

// Zeros, Ones, Full, Empty, Arange, Linspace, Logspace, Eye, Identity,
// Diag, Rand and Randn mirror their NumPy namesakes.
func Zeros(shape ...int) *NDArray               { return ndarray.Zeros(shape...) }
func Ones(shape ...int) *NDArray                { return ndarray.Ones(shape...) }
func Full(value float64, shape ...int) *NDArray { return ndarray.Full(value, shape...) }
func Empty(shape ...int) *NDArray               { return ndarray.Empty(shape...) }
func Arange(args ...float64) *NDArray           { return ndarray.Arange(args...) }
func Linspace(start, stop float64, num int) *NDArray {
	return ndarray.Linspace(start, stop, num)
}
func Logspace(start, stop float64, num int) *NDArray {
	return ndarray.Logspace(start, stop, num)
}
func Eye(n int) *NDArray                { return ndarray.Eye(n) }
func Identity(n int) *NDArray           { return ndarray.Identity(n) }
func Diag(v *NDArray) (*NDArray, error) { return ndarray.Diag(v) }
func Rand(shape ...int) *NDArray        { return ndarray.Rand(shape...) }
func Randn(shape ...int) *NDArray       { return ndarray.Randn(shape...) }

// Dot and MatMul mirror np.dot / np.matmul.
func Dot(a, b *NDArray) (*NDArray, error)    { return ndarray.Dot(a, b) }
func MatMul(a, b *NDArray) (*NDArray, error) { return ndarray.MatMul(a, b) }

// Slice, SliceStep and All build positional slice specs for NDArray.Slice
// and df.ILoc().
func Slice(start, stop int) SliceSpec           { return ndarray.Slice(start, stop) }
func SliceStep(start, stop, step int) SliceSpec { return ndarray.SliceStep(start, stop, step) }
func All() SliceSpec                            { return ndarray.All() }

// IO -----------------------------------------------------------------------

// ReadCSV reads a CSV file into a DataFrame (pd.read_csv).
func ReadCSV(path string, opts ...CSVOption) (*DataFrame, error) {
	return dataframe.ReadCSV(path, opts...)
}

// ReadCSVReader reads CSV from any reader.
func ReadCSVReader(r stdio.Reader, opts ...CSVOption) (*DataFrame, error) {
	return dataframe.ReadCSVReader(r, opts...)
}

// ReadJSON reads a JSON array of objects (pd.read_json).
func ReadJSON(path string, opts ...JSONOption) (*DataFrame, error) {
	return dataframe.ReadJSON(path, opts...)
}

// ReadNDJSON reads newline-delimited JSON.
func ReadNDJSON(path string, opts ...JSONOption) (*DataFrame, error) {
	return dataframe.ReadNDJSON(path, opts...)
}

// CSV/JSON option re-exports.
func WithHeader(v bool) CSVOption             { return pdio.WithHeader(v) }
func WithComma(r rune) CSVOption              { return pdio.WithComma(r) }
func WithInferTypes(v bool) CSVOption         { return pdio.WithInferTypes(v) }
func WithNAValues(values ...string) CSVOption { return pdio.WithNAValues(values...) }
func WithParseDates(columns ...string) CSVOption {
	return pdio.WithParseDates(columns...)
}
func WithDateFormat(format string) CSVOption { return pdio.WithDateFormat(format) }
func WithTrimSpace(v bool) CSVOption         { return pdio.WithTrimSpace(v) }
func WithLimit(n int) CSVOption              { return pdio.WithLimit(n) }
func WithOrient(orient string) JSONOption    { return pdio.WithOrient(orient) }

// Combining -----------------------------------------------------------------

// Concat concatenates frames (pd.concat).
func Concat(frames []*DataFrame, opts ...ConcatOption) (*DataFrame, error) {
	return dataframe.Concat(frames, opts...)
}

// ConcatAxis, ConcatJoin and ConcatIgnoreIndex re-export concat options.
func ConcatAxis(axis int) ConcatOption      { return dataframe.ConcatAxis(axis) }
func ConcatJoin(join string) ConcatOption   { return dataframe.ConcatJoin(join) }
func ConcatIgnoreIndex(v bool) ConcatOption { return dataframe.ConcatIgnoreIndex(v) }

// Merge joins two frames on key columns (pd.merge).
func Merge(left, right *DataFrame, opts MergeOptions) (*DataFrame, error) {
	return dataframe.Merge(left, right, opts)
}

// Expressions ----------------------------------------------------------------

// Col references a column inside an expression: pd.Col("age").Gt(30).
func Col(name string) expr.ColumnExpr { return expr.Col(name) }

// Lit wraps a constant value as an expression.
func Lit(v any) expr.LiteralExpr { return expr.Lit(v) }

// And, Or, Not combine predicates; Where selects between two expressions.
func And(preds ...Predicate) Predicate        { return expr.And(preds...) }
func Or(preds ...Predicate) Predicate         { return expr.Or(preds...) }
func Not(pred Predicate) Predicate            { return expr.Not(pred) }
func Where(cond Predicate, x any, y any) Expr { return expr.Where(cond, x, y) }

// Expression functions.
func Abs(e Expr) Expr   { return expr.Abs(e) }
func Sqrt(e Expr) Expr  { return expr.Sqrt(e) }
func Log(e Expr) Expr   { return expr.Log(e) }
func Exp(e Expr) Expr   { return expr.Exp(e) }
func Lower(e Expr) Expr { return expr.Lower(e) }
func Upper(e Expr) Expr { return expr.Upper(e) }
func Len(e Expr) Expr   { return expr.Len(e) }

// Window and groupby options ----------------------------------------------------

// RollingMinPeriods sets the minimum observations per rolling window.
func RollingMinPeriods(n int) RollingOption { return series.RollingMinPeriods(n) }

// RollingCenter centers rolling window labels.
func RollingCenter(v bool) RollingOption { return series.RollingCenter(v) }

// GroupDropNA controls whether rows with missing group keys are dropped.
func GroupDropNA(v bool) GroupByOption { return dataframe.GroupDropNA(v) }

// GroupSort controls whether groups are sorted by key.
func GroupSort(v bool) GroupByOption { return dataframe.GroupSort(v) }

// SkipNA controls whether reductions ignore missing values.
func SkipNA(v bool) ReduceOption { return series.SkipNA(v) }

// DropNAHow sets DropNA row behavior ("any" or "all").
func DropNAHow(how string) DropNAOption { return dataframe.DropNAHow(how) }

// DropNASubset restricts DropNA to a subset of columns.
func DropNASubset(columns ...string) DropNAOption { return dataframe.DropNASubset(columns...) }

// Missing values ---------------------------------------------------------------

// NA returns the generic missing value marker (pd.NA).
func NA() any { return dtype.NA() }

// NaT returns the missing datetime marker (pd.NaT).
func NaT() any { return dtype.NaT() }

// IsNA, NotNA, IsNull and NotNull test scalar missingness.
func IsNA(v any) bool    { return dtype.IsNA(v) }
func NotNA(v any) bool   { return dtype.NotNA(v) }
func IsNull(v any) bool  { return dtype.IsNull(v) }
func NotNull(v any) bool { return dtype.NotNull(v) }

// DType helpers.
func InferDType(values []any) DType { return dtype.InferDType(values) }
func Promote(a, b DType) DType      { return dtype.Promote(a, b) }

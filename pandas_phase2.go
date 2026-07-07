package pandas

import (
	"time"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
	pdio "github.com/arturoeanton/go-pandas/io"
	"github.com/arturoeanton/go-pandas/ndarray"
	"github.com/arturoeanton/go-pandas/series"
)

// DType helpers -------------------------------------------------------------

// Number is the pseudo-dtype matching every numeric dtype, for use with
// SelectDTypes (pandas include=["number"]).
const Number = dtype.Number

// ParseDType parses a pandas/NumPy dtype name like "int64",
// "datetime64[ns]" or "category".
func ParseDType(name string) (DType, error) { return dtype.ParseDType(name) }

// Include and Exclude build SelectDTypes filters.
func Include(dts ...DType) dataframe.SelectDTypesOption { return dataframe.Include(dts...) }
func Exclude(dts ...DType) dataframe.SelectDTypesOption { return dataframe.Exclude(dts...) }

// Axis names a reduction axis, purely for readability:
// a.Sum(pd.Axis(0)) reads like a.sum(axis=0).
func Axis(i int) int { return i }

// AxisRows / AxisColumns re-export the DataFrame axis constants.
const (
	AxisRows    = dataframe.AxisRows
	AxisColumns = dataframe.AxisColumns
)

// Datetime helpers ------------------------------------------------------------

// ToDatetime converts a series to typed datetime storage, like
// pd.to_datetime(s, format=..., errors=...). Since v0.9 it accepts
// pandas-style options:
//
//	pd.ToDatetime(s, pd.WithDatetimeFormat("%Y-%m-%d"))
//	pd.ToDatetime(s, pd.WithDatetimeErrors("coerce"))
func ToDatetime(s *Series, opts ...DatetimeOption) (*Series, error) {
	return series.ToDatetime(s, opts...)
}

// ParseDatetime parses a single datetime string with the common layouts.
func ParseDatetime(v string) (time.Time, error) { return dtype.ParseTime(v) }

// Typed NDArray constructors ---------------------------------------------------

// ArrayInt, ArrayInt64, ArrayFloat32, ArrayFloat64 and ArrayBool build
// 1-D arrays recording their logical dtype (float64 storage in v0.2).
func ArrayInt(data []int) *NDArray         { return ndarray.ArrayInt(data) }
func ArrayInt64(data []int64) *NDArray     { return ndarray.ArrayInt64(data) }
func ArrayFloat32(data []float32) *NDArray { return ndarray.ArrayFloat32(data) }
func ArrayFloat64(data []float64) *NDArray { return ndarray.ArrayFloat64(data) }
func ArrayBool(data []bool) *NDArray       { return ndarray.ArrayBool(data) }

// AsArray copies any numeric slice into a 1-D array (np.asarray).
func AsArray[T ndarray.Number](data []T) *NDArray { return ndarray.AsArray(data) }

// NDArray joining and set operations ---------------------------------------------

// Concatenate joins arrays along an existing axis (np.concatenate).
func Concatenate(arrays []*NDArray, axis int) (*NDArray, error) {
	return ndarray.Concatenate(arrays, axis)
}

// Stack joins arrays along a new axis (np.stack).
func Stack(arrays []*NDArray, axis int) (*NDArray, error) {
	return ndarray.StackArrays(arrays, axis)
}

// HStack and VStack mirror np.hstack / np.vstack.
func HStack(arrays []*NDArray) (*NDArray, error) { return ndarray.HStack(arrays) }
func VStack(arrays []*NDArray) (*NDArray, error) { return ndarray.VStack(arrays) }

// Unique returns the sorted distinct values of an array (np.unique).
func Unique(a *NDArray) *NDArray { return ndarray.Unique(a) }

// NDArray ufuncs (root forms; every one also exists as a method) -------------------

func Abs(a *NDArray) *NDArray   { return a.Abs() }
func Sqrt(a *NDArray) *NDArray  { return a.Sqrt() }
func Exp(a *NDArray) *NDArray   { return a.Exp() }
func Log(a *NDArray) *NDArray   { return a.Log() }
func Log10(a *NDArray) *NDArray { return a.Log10() }
func Sin(a *NDArray) *NDArray   { return a.Sin() }
func Cos(a *NDArray) *NDArray   { return a.Cos() }
func Tan(a *NDArray) *NDArray   { return a.Tan() }
func Floor(a *NDArray) *NDArray { return a.Floor() }
func Ceil(a *NDArray) *NDArray  { return a.Ceil() }
func Round(a *NDArray) *NDArray { return a.Round() }

// Clip limits array values to [min, max] (np.clip).
func Clip(a *NDArray, min, max float64) *NDArray { return a.Clip(min, max) }

// IsNaN, IsFinite and IsInf mirror np.isnan / np.isfinite / np.isinf.
func IsNaN(a *NDArray) *BoolArray    { return a.IsNaN() }
func IsFinite(a *NDArray) *BoolArray { return a.IsFinite() }
func IsInf(a *NDArray) *BoolArray    { return a.IsInf() }

// Binary NDArray functions (np.add, np.subtract, ...) ---------------------------------

func Add(a, b *NDArray) (*NDArray, error)      { return a.Add(b) }
func Subtract(a, b *NDArray) (*NDArray, error) { return a.Sub(b) }
func Multiply(a, b *NDArray) (*NDArray, error) { return a.Mul(b) }
func Divide(a, b *NDArray) (*NDArray, error)   { return a.Div(b) }
func Power(a, b *NDArray) (*NDArray, error)    { return a.Pow(b) }
func Maximum(a, b *NDArray) (*NDArray, error)  { return ndarray.Maximum(a, b) }
func Minimum(a, b *NDArray) (*NDArray, error)  { return ndarray.Minimum(a, b) }

// WhereArray selects from x where mask is true, else from y (np.where).
// The name avoids clashing with the expression pd.Where.
func WhereArray(mask *BoolArray, x, y *NDArray) (*NDArray, error) {
	return ndarray.Where(mask, x, y)
}

// WhereScalar selects from a where mask is true, else the scalar.
func WhereScalar(mask *BoolArray, a *NDArray, other float64) (*NDArray, error) {
	return ndarray.WhereScalar(mask, a, other)
}

// Window and concat aliases -------------------------------------------------------

// MinPeriods is the pandas-named alias of RollingMinPeriods.
func MinPeriods(n int) RollingOption { return series.MinPeriods(n) }

// IgnoreIndex is the pandas-named alias of ConcatIgnoreIndex.
func IgnoreIndex(v bool) ConcatOption { return dataframe.ConcatIgnoreIndex(v) }

// Join sets outer/inner column handling for Concat (pandas join=).
func Join(how string) ConcatOption { return dataframe.ConcatJoin(how) }

// DropNAThresh keeps rows with at least n non-NA values (pandas thresh=).
func DropNAThresh(n int) DropNAOption { return dataframe.DropNAThresh(n) }

// DropNAAxis selects the DropNA axis: 0 drops rows, 1 drops columns.
func DropNAAxis(axis int) DropNAOption { return dataframe.DropNAAxis(axis) }

// Indexing helpers -------------------------------------------------------------------

// LabelSlice builds an inclusive label range for df.Loc().Rows, matching
// pandas' inclusive df.loc["a":"z"] semantics.
func LabelSlice(start, stop any) dataframe.LabelRange {
	return dataframe.LabelSlice(start, stop)
}

// Rank options ------------------------------------------------------------------------

// RankMethod and RankAscending re-export the Series rank options.
func RankMethod(m string) series.RankOption  { return series.RankMethod(m) }
func RankAscending(v bool) series.RankOption { return series.RankAscending(v) }

// SampleOption re-export.
func WithSampleSeed(seed int64) dataframe.SampleOption { return dataframe.WithSampleSeed(seed) }

// IO option re-exports added in Phase 2.
func WithUseCols(columns ...string) CSVOption { return pdio.WithUseCols(columns...) }
func WithNRows(n int) CSVOption               { return pdio.WithNRows(n) }
func WithKeepDefaultNA(v bool) CSVOption      { return pdio.WithKeepDefaultNA(v) }
func JSONOrient(orient string) JSONOption     { return pdio.WithOrient(orient) }

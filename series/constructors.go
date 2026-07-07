package series

import (
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// SeriesOption customizes construction.
type SeriesOption func(*Series)

// WithIndex attaches an explicit index (must match the data length).
func WithIndex(idx index.Index) SeriesOption {
	return func(s *Series) {
		if idx != nil && idx.Len() == s.Len() {
			s.index = idx.Clone()
		}
	}
}

// WithDType forces the dtype: the column is rebuilt with the requested
// storage when the values convert cleanly; otherwise the data stays
// object-backed with the requested logical dtype.
func WithDType(dt dtype.DType) SeriesOption {
	return func(s *Series) {
		if s.DType() == dt {
			return
		}
		rebuilt := column.FromAny(s.col.Values(), dt)
		if column.IsObjectBacked(rebuilt) && dt != dtype.Object {
			// conversion failed: keep values, carry the forced dtype
			rebuilt = column.FromAny(s.col.Values(), dtype.Object)
		}
		s.col = rebuilt
	}
}

// WithName renames the series.
func WithName(name string) SeriesOption {
	return func(s *Series) { s.name = name }
}

// NewSeries builds a series from untyped values, inferring the dtype and
// building typed storage when the data is homogeneous (nil, NA(), NaT()
// and NaN are missing).
func NewSeries(name string, values []any, opts ...SeriesOption) *Series {
	s := fromColumn(name, column.Infer(values), nil)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SeriesOf builds a series from a typed slice. Supported element types
// (bool, int, int64, float32, float64, string, time.Time) go straight
// into typed columns without boxing.
func SeriesOf[T any](name string, values []T, opts ...SeriesOption) *Series {
	if col := column.FromTyped(any(values)); col != nil {
		s := fromColumn(name, col, nil)
		for _, opt := range opts {
			opt(s)
		}
		return s
	}
	anyValues := make([]any, len(values))
	for i, v := range values {
		anyValues[i] = v
	}
	return NewSeries(name, anyValues, opts...)
}

// IntSeries builds an Int series backed by []int.
func IntSeries(name string, values []int) *Series { return SeriesOf(name, values) }

// Int64Series builds an Int64 series backed by []int64.
func Int64Series(name string, values []int64) *Series { return SeriesOf(name, values) }

// FloatSeries builds a Float64 series backed by []float64 (NaN entries
// are missing).
func FloatSeries(name string, values []float64) *Series { return SeriesOf(name, values) }

// StringSeries builds a String series backed by []string.
func StringSeries(name string, values []string) *Series { return SeriesOf(name, values) }

// BoolSeries builds a Bool series backed by []bool.
func BoolSeries(name string, values []bool) *Series { return SeriesOf(name, values) }

// TimeSeries builds a datetime series backed by []time.Time.
func TimeSeries(name string, values []time.Time) *Series { return SeriesOf(name, values) }

// FromValuesMask builds a series from pre-computed data and mask; used by
// internal operations that already know which entries are missing.
func FromValuesMask(name string, data []any, mask []bool, dt dtype.DType, idx index.Index) *Series {
	values := dataWithMask(data, mask)
	if dt == dtype.Invalid {
		dt = dtype.InferDType(values)
	}
	return fromColumn(name, column.FromAny(values, dt), idx)
}

// dataWithMask boxes data with masked entries normalized to nil.
func dataWithMask(data []any, mask []bool) []any {
	out := make([]any, len(data))
	for i, v := range data {
		if mask[i] {
			out[i] = nil
		} else {
			out[i] = v
		}
	}
	return out
}

// floatColumn assembles a Float64-backed series from raw data+mask —
// the common shape of numeric kernels (rolling, diff, rank...).
func floatColumnSeries(name string, data []float64, mask []bool, idx index.Index) *Series {
	if idx != nil {
		idx = idx.Clone()
	}
	return fromColumn(name, column.NewFloat64(data, mask), idx)
}

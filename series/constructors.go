package series

import (
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
)

// SeriesOption customizes construction.
type SeriesOption func(*Series)

// WithIndex attaches an explicit index (must match the data length).
func WithIndex(idx index.Index) SeriesOption {
	return func(s *Series) {
		if idx != nil && idx.Len() == len(s.data) {
			s.index = idx.Clone()
		}
	}
}

// WithDType forces the dtype instead of inferring it. Values are not
// converted; use Astype for that.
func WithDType(dt dtype.DType) SeriesOption {
	return func(s *Series) { s.dtype = dt }
}

// WithName renames the series.
func WithName(name string) SeriesOption {
	return func(s *Series) { s.name = name }
}

// NewSeries builds a series from untyped values, inferring the dtype and
// building the missing mask (nil, NA(), NaT() and NaN are missing).
func NewSeries(name string, values []any, opts ...SeriesOption) *Series {
	data := make([]any, len(values))
	mask := make([]bool, len(values))
	for i, v := range values {
		if dtype.IsNA(v) {
			mask[i] = true
			// Keep NaT/float markers so formatting can distinguish them.
			if _, ok := v.(dtype.NaTMarker); ok {
				data[i] = dtype.NaTMarker{}
			}
			continue
		}
		data[i] = v
	}
	s := &Series{
		name:  name,
		data:  data,
		mask:  mask,
		dtype: dtype.InferDType(values),
		index: index.NewRangeIndex(len(values)),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SeriesOf builds a series from a typed slice.
func SeriesOf[T any](name string, values []T, opts ...SeriesOption) *Series {
	anyValues := make([]any, len(values))
	for i, v := range values {
		anyValues[i] = v
	}
	return NewSeries(name, anyValues, opts...)
}

// IntSeries builds an Int series.
func IntSeries(name string, values []int) *Series { return SeriesOf(name, values) }

// Int64Series builds an Int64 series.
func Int64Series(name string, values []int64) *Series { return SeriesOf(name, values) }

// FloatSeries builds a Float64 series (NaN entries are missing).
func FloatSeries(name string, values []float64) *Series { return SeriesOf(name, values) }

// StringSeries builds a String series.
func StringSeries(name string, values []string) *Series { return SeriesOf(name, values) }

// BoolSeries builds a Bool series.
func BoolSeries(name string, values []bool) *Series { return SeriesOf(name, values) }

// TimeSeries builds a datetime series.
func TimeSeries(name string, values []time.Time) *Series { return SeriesOf(name, values) }

// FromValuesMask builds a series from pre-computed data and mask; used by
// internal operations that already know which entries are missing.
func FromValuesMask(name string, data []any, mask []bool, dt dtype.DType, idx index.Index) *Series {
	if idx == nil || idx.Len() != len(data) {
		idx = index.NewRangeIndex(len(data))
	}
	if dt == dtype.Invalid {
		dt = dtype.InferDType(dataWithMask(data, mask))
	}
	return &Series{
		name:  name,
		data:  append([]any(nil), data...),
		mask:  append([]bool(nil), mask...),
		dtype: dt,
		index: idx.Clone(),
	}
}

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

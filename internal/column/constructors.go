package column

import (
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
)

// Typed constructors. Each takes pre-separated data+mask (mask may be nil
// for fully-present data); the slices are used directly (callers hand
// over ownership).

func normalizeMask[T any](data []T, mask []bool) []bool {
	if mask == nil {
		mask = make([]bool, len(data))
	}
	return mask
}

// NewBool builds a bool column.
func NewBool(data []bool, mask []bool) Column {
	return &typedColumn[bool]{
		dt: dtype.Bool, data: data, mask: normalizeMask(data, mask),
		conv: func(v any) (bool, bool) { b, ok := v.(bool); return b, ok },
		toFloat: func(v bool) (float64, bool) {
			if v {
				return 1, true
			}
			return 0, true
		},
	}
}

// NewInt builds an int column.
func NewInt(data []int, mask []bool) Column {
	return &typedColumn[int]{
		dt: dtype.Int, data: data, mask: normalizeMask(data, mask),
		conv: func(v any) (int, bool) {
			i, ok := dtype.AsInt(v)
			return int(i), ok
		},
		toFloat: func(v int) (float64, bool) { return float64(v), true },
	}
}

// NewInt64 builds an int64 column.
func NewInt64(data []int64, mask []bool) Column {
	return &typedColumn[int64]{
		dt: dtype.Int64, data: data, mask: normalizeMask(data, mask),
		conv:    dtype.AsInt,
		toFloat: func(v int64) (float64, bool) { return float64(v), true },
	}
}

// NewFloat32 builds a float32 column.
func NewFloat32(data []float32, mask []bool) Column {
	return &typedColumn[float32]{
		dt: dtype.Float32, data: data, mask: normalizeMask(data, mask),
		conv: func(v any) (float32, bool) {
			f, ok := dtype.AsFloat(v)
			return float32(f), ok
		},
		toFloat: func(v float32) (float64, bool) { return float64(v), true },
	}
}

// NewFloat64 builds a float64 column.
func NewFloat64(data []float64, mask []bool) Column {
	return &typedColumn[float64]{
		dt: dtype.Float64, data: data, mask: normalizeMask(data, mask),
		conv:    dtype.AsFloat,
		toFloat: func(v float64) (float64, bool) { return v, true },
	}
}

// NewString builds a string column.
func NewString(data []string, mask []bool) Column {
	return &typedColumn[string]{
		dt: dtype.String, data: data, mask: normalizeMask(data, mask),
		conv: func(v any) (string, bool) { s, ok := v.(string); return s, ok },
	}
}

// NewTime builds a datetime column; masked slots surface as NaT.
func NewTime(data []time.Time, mask []bool) Column {
	return &typedColumn[time.Time]{
		dt: dtype.Time, data: data, mask: normalizeMask(data, mask),
		conv:    func(v any) (time.Time, bool) { t, ok := v.(time.Time); return t, ok },
		naValue: dtype.NaTMarker{},
	}
}

// NewObject builds the []any fallback column. The dt parameter lets a
// caller keep a forced logical dtype (defaults to Object).
func NewObject(data []any, mask []bool, dt dtype.DType) Column {
	if dt == dtype.Invalid {
		dt = dtype.Object
	}
	return &typedColumn[any]{
		dt: dt, data: data, mask: normalizeMask(data, mask),
		conv: func(v any) (any, bool) { return v, true },
		toFloat: func(v any) (float64, bool) {
			return dtype.AsFloat(v)
		},
	}
}

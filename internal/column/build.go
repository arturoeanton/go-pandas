package column

import (
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
)

// FromTyped builds a typed column directly from a homogeneous Go slice
// without boxing. NA detection follows the dtype rules (NaN floats are
// missing). Returns nil when the element type has no typed column.
func FromTyped(values any) Column {
	switch data := values.(type) {
	case []bool:
		return NewBool(append([]bool(nil), data...), nil)
	case []int:
		return NewInt(append([]int(nil), data...), nil)
	case []int64:
		return NewInt64(append([]int64(nil), data...), nil)
	case []float32:
		d := append([]float32(nil), data...)
		mask := make([]bool, len(d))
		for i, v := range d {
			if v != v { // NaN
				mask[i] = true
				d[i] = 0
			}
		}
		return NewFloat32(d, mask)
	case []float64:
		d := append([]float64(nil), data...)
		mask := make([]bool, len(d))
		for i, v := range d {
			if v != v { // NaN
				mask[i] = true
				d[i] = 0
			}
		}
		return NewFloat64(d, mask)
	case []string:
		return NewString(append([]string(nil), data...), nil)
	case []time.Time:
		return NewTime(append([]time.Time(nil), data...), nil)
	}
	return nil
}

// empty builds a zero-length column of the requested dtype.
func empty(dt dtype.DType, capacity int) Column {
	switch dt {
	case dtype.Bool:
		return NewBool(make([]bool, 0, capacity), make([]bool, 0, capacity))
	case dtype.Int:
		return NewInt(make([]int, 0, capacity), make([]bool, 0, capacity))
	case dtype.Int64:
		return NewInt64(make([]int64, 0, capacity), make([]bool, 0, capacity))
	case dtype.Float32:
		return NewFloat32(make([]float32, 0, capacity), make([]bool, 0, capacity))
	case dtype.Float64:
		return NewFloat64(make([]float64, 0, capacity), make([]bool, 0, capacity))
	case dtype.String:
		return NewString(make([]string, 0, capacity), make([]bool, 0, capacity))
	case dtype.Time:
		return NewTime(make([]time.Time, 0, capacity), make([]bool, 0, capacity))
	default:
		return NewObject(make([]any, 0, capacity), make([]bool, 0, capacity), dt)
	}
}

// FromAny builds a column of the requested dtype from boxed values.
// Values that cannot be converted downgrade the column to Object (the
// documented mixed-value fallback); NA-like values become masked slots.
func FromAny(values []any, dt dtype.DType) Column {
	if dt == dtype.Category {
		cat, err := Factorize(values, nil, false)
		if err != nil {
			return buildObject(values)
		}
		return cat
	}
	col := empty(dt, len(values))
	for _, v := range values {
		if err := col.AppendValue(v); err != nil {
			return buildObject(values)
		}
	}
	return col
}

// Infer builds the best typed column for boxed values using the shared
// dtype inference rules: homogeneous values get typed storage, mixed
// int/float promotes to Float64, anything else falls back to Object.
func Infer(values []any) Column {
	return FromAny(values, dtype.InferDType(values))
}

// buildObject stores values as-is with NA-likes masked.
func buildObject(values []any) Column {
	col := empty(dtype.Object, len(values))
	for _, v := range values {
		_ = col.AppendValue(v) // object append never fails
	}
	return col
}

// IsObjectBacked reports whether a column uses []any storage.
func IsObjectBacked(c Column) bool {
	_, ok := c.(*typedColumn[any])
	return ok
}

// StorageDType returns the dtype of the physical storage: the column
// dtype for typed columns and Object for []any-backed columns even when
// they carry a forced logical dtype.
func StorageDType(c Column) dtype.DType {
	if IsObjectBacked(c) {
		return dtype.Object
	}
	return c.DType()
}

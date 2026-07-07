// Package column implements the typed storage engine behind Series and
// DataFrame columns (v0.3). Each column stores a homogeneous Go slice
// plus a missing-value mask, so common dtypes never box values into
// []any. ObjectColumn remains the fallback for mixed or unsupported
// values.
package column

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// Column is the storage interface shared by every typed column.
// Mask convention: masked entries are missing; Value returns nil for
// them (TimeColumn returns the NaT marker so formatting can distinguish
// missing datetimes).
type Column interface {
	DType() dtype.DType
	Len() int
	IsNA(i int) bool
	// Value returns the boxed value at i, nil (or NaT) when missing.
	Value(i int) any
	// SetValue converts and stores v; NA-like values mark the slot
	// missing. Unconvertible values return ErrTypeMismatch.
	SetValue(i int, v any) error
	// AppendValue grows the column by one element.
	AppendValue(v any) error
	// Take gathers positions into a new column; negative positions
	// produce missing entries.
	Take(indices []int) (Column, error)
	// Slice copies the [start, stop) range into a new column.
	Slice(start, stop int) (Column, error)
	Copy() Column
	// Values boxes the whole column into []any (nil for missing).
	Values() []any
	// Float64s extracts the column as float64s plus the mask without
	// per-element boxing. ok is false for non-numeric columns. The
	// returned slices must be treated as read-only: for Float64 columns
	// they alias the internal storage.
	Float64s() (values []float64, mask []bool, ok bool)
}

// typedColumn is the single generic implementation behind every
// concrete dtype.
type typedColumn[T any] struct {
	dt   dtype.DType
	data []T
	mask []bool
	// conv converts an arbitrary value into T (reports false when
	// impossible). naValue is what Value returns for masked slots.
	conv    func(v any) (T, bool)
	toFloat func(v T) (float64, bool)
	naValue any
}

func (c *typedColumn[T]) DType() dtype.DType { return c.dt }
func (c *typedColumn[T]) Len() int           { return len(c.data) }
func (c *typedColumn[T]) IsNA(i int) bool    { return c.mask[i] }

func (c *typedColumn[T]) Value(i int) any {
	if c.mask[i] {
		return c.naValue
	}
	return c.data[i]
}

func (c *typedColumn[T]) SetValue(i int, v any) error {
	if i < 0 || i >= len(c.data) {
		return fmt.Errorf("%w: position %d for column of length %d", errs.ErrIndexOutOfBounds, i, len(c.data))
	}
	if dtype.IsNA(v) {
		var zero T
		c.data[i] = zero
		c.mask[i] = true
		return nil
	}
	t, ok := c.conv(v)
	if !ok {
		return fmt.Errorf("%w: cannot store %T in %s column", errs.ErrTypeMismatch, v, c.dt)
	}
	c.data[i] = t
	c.mask[i] = false
	return nil
}

func (c *typedColumn[T]) AppendValue(v any) error {
	var zero T
	c.data = append(c.data, zero)
	c.mask = append(c.mask, true)
	return c.SetValue(len(c.data)-1, v)
}

func (c *typedColumn[T]) Take(indices []int) (Column, error) {
	data := make([]T, len(indices))
	mask := make([]bool, len(indices))
	for out, src := range indices {
		if src < 0 {
			mask[out] = true
			continue
		}
		if src >= len(c.data) {
			return nil, fmt.Errorf("%w: take position %d for column of length %d", errs.ErrIndexOutOfBounds, src, len(c.data))
		}
		data[out] = c.data[src]
		mask[out] = c.mask[src]
	}
	return c.with(data, mask), nil
}

func (c *typedColumn[T]) Slice(start, stop int) (Column, error) {
	if start < 0 || stop < start || stop > len(c.data) {
		return nil, fmt.Errorf("%w: slice [%d:%d] for column of length %d", errs.ErrIndexOutOfBounds, start, stop, len(c.data))
	}
	return c.with(
		append([]T(nil), c.data[start:stop]...),
		append([]bool(nil), c.mask[start:stop]...),
	), nil
}

func (c *typedColumn[T]) Copy() Column {
	return c.with(
		append([]T(nil), c.data...),
		append([]bool(nil), c.mask...),
	)
}

// with builds a sibling column sharing the dtype and converters.
func (c *typedColumn[T]) with(data []T, mask []bool) *typedColumn[T] {
	return &typedColumn[T]{
		dt: c.dt, data: data, mask: mask,
		conv: c.conv, toFloat: c.toFloat, naValue: c.naValue,
	}
}

func (c *typedColumn[T]) Values() []any {
	out := make([]any, len(c.data))
	for i := range c.data {
		if c.mask[i] {
			out[i] = nil
			continue
		}
		out[i] = c.data[i]
	}
	return out
}

func (c *typedColumn[T]) Float64s() ([]float64, []bool, bool) {
	if c.toFloat == nil {
		return nil, nil, false
	}
	// Float64 columns expose their storage directly (read-only).
	if direct, ok := any(c.data).([]float64); ok {
		return direct, c.mask, true
	}
	out := make([]float64, len(c.data))
	for i, v := range c.data {
		if c.mask[i] {
			continue
		}
		f, ok := c.toFloat(v)
		if !ok {
			return nil, nil, false
		}
		out[i] = f
	}
	return out, c.mask, true
}

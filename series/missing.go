package series

import (
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// boolSeries builds a Bool-backed series aligned with s.
func (s *Series) boolSeries(name string, f func(i int) bool) *Series {
	data := make([]bool, s.Len())
	for i := range data {
		data[i] = f(i)
	}
	return &Series{
		name:  name,
		col:   column.NewBool(data, nil),
		index: s.index.Clone(),
	}
}

// IsNA returns a boolean series marking missing entries.
func (s *Series) IsNA() *Series {
	return s.boolSeries(s.name, func(i int) bool { return s.col.IsNA(i) })
}

// NotNA returns a boolean series marking present entries.
func (s *Series) NotNA() *Series {
	return s.boolSeries(s.name, func(i int) bool { return !s.col.IsNA(i) })
}

// IsNull is an alias of IsNA.
func (s *Series) IsNull() *Series { return s.IsNA() }

// NotNull is an alias of NotNA.
func (s *Series) NotNull() *Series { return s.NotNA() }

// DropNA returns the series without its missing entries.
func (s *Series) DropNA() *Series {
	var keep []int
	for i := 0; i < s.Len(); i++ {
		if !s.col.IsNA(i) {
			keep = append(keep, i)
		}
	}
	out, _ := s.Take(keep)
	return out
}

// FillNA replaces missing entries with a value. When the value fits the
// typed column the storage stays typed; otherwise the series rebuilds
// with a promoted or object column (e.g. filling an int column with a
// string).
func (s *Series) FillNA(v any) *Series {
	c := s.Copy()
	for i := 0; i < c.Len(); i++ {
		if !c.col.IsNA(i) {
			continue
		}
		if err := c.col.SetValue(i, v); err != nil {
			// value does not fit the typed storage: rebuild boxed
			values := s.col.Values()
			for j := range values {
				if s.col.IsNA(j) {
					values[j] = v
				}
			}
			return fromColumn(s.name, column.Infer(values), s.index.Clone())
		}
	}
	return c
}

// Astype converts every value to the target dtype, changing the real
// storage type (v0.3).
func (s *Series) Astype(dt dtype.DType) (*Series, error) {
	if dt == dtype.Category {
		return s.asCategorical()
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.col.IsNA(i) {
			continue
		}
		v, err := dtype.CastValue(s.col.Value(i), dt)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return fromColumn(s.name, column.FromAny(values, dt), s.index.Clone()), nil
}

// InferObjects re-infers the dtype (and storage) of an object-backed
// series, e.g. after IO or FillNA changed the value kinds.
func (s *Series) InferObjects() *Series {
	return fromColumn(s.name, column.Infer(s.col.Values()), s.index.Clone())
}

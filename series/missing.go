package series

import (
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
)

// boolSeries builds a Bool series aligned with s.
func (s *Series) boolSeries(name string, f func(i int) bool) *Series {
	data := make([]any, s.Len())
	mask := make([]bool, s.Len())
	for i := range data {
		data[i] = f(i)
	}
	return &Series{
		name:  name,
		dtype: dtype.Bool,
		data:  data,
		mask:  mask,
		index: s.index.Clone(),
	}
}

// IsNA returns a boolean series marking missing entries.
func (s *Series) IsNA() *Series {
	return s.boolSeries(s.name, func(i int) bool { return s.mask[i] })
}

// NotNA returns a boolean series marking present entries.
func (s *Series) NotNA() *Series {
	return s.boolSeries(s.name, func(i int) bool { return !s.mask[i] })
}

// IsNull is an alias of IsNA.
func (s *Series) IsNull() *Series { return s.IsNA() }

// NotNull is an alias of NotNA.
func (s *Series) NotNull() *Series { return s.NotNA() }

// DropNA returns the series without its missing entries.
func (s *Series) DropNA() *Series {
	var keep []int
	for i, m := range s.mask {
		if !m {
			keep = append(keep, i)
		}
	}
	out, _ := s.Take(keep)
	return out
}

// FillNA replaces missing entries with a value.
func (s *Series) FillNA(v any) *Series {
	c := s.Copy()
	for i, m := range c.mask {
		if m {
			c.data[i] = v
			c.mask[i] = false
		}
	}
	c.dtype = dtype.InferDType(c.Values())
	return c
}

// Astype converts every value to the target dtype.
func (s *Series) Astype(dt dtype.DType) (*Series, error) {
	data := make([]any, s.Len())
	for i := range s.data {
		if s.mask[i] {
			continue
		}
		v, err := dtype.CastValue(s.data[i], dt)
		if err != nil {
			return nil, err
		}
		data[i] = v
	}
	return &Series{
		name:  s.name,
		dtype: dt,
		data:  data,
		mask:  append([]bool(nil), s.mask...),
		index: s.index.Clone(),
	}, nil
}

// InferObjects re-infers the dtype of an Object series (e.g. after IO or
// FillNA changed the value kinds).
func (s *Series) InferObjects() *Series {
	c := s.Copy()
	c.dtype = dtype.InferDType(c.Values())
	return c
}

// ensureAligned pads/reorders other to s's index. v0.1 requires equal
// lengths; equal indexes are aligned by position.
func ensureIndex(s *Series) index.Index { return s.index }

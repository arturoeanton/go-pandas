package series

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
)

// At returns the value at a position (nil when missing).
func (s *Series) At(pos int) (any, error) {
	if pos < 0 || pos >= s.Len() {
		return nil, fmt.Errorf("%w: position %d for series of length %d", errs.ErrIndexOutOfBounds, pos, s.Len())
	}
	return s.valueAt(pos), nil
}

// IAt is an alias of At (pandas .iat).
func (s *Series) IAt(pos int) (any, error) { return s.At(pos) }

// Loc returns the value for an index label.
func (s *Series) Loc(label any) (any, error) {
	pos, ok := s.index.Pos(label)
	if !ok {
		return nil, fmt.Errorf("%w: label %v", errs.ErrInvalidIndex, label)
	}
	return s.valueAt(pos), nil
}

// Set writes a value at a position; NA-like values mark it missing.
// Values that cannot be stored in the typed column return
// ErrTypeMismatch (v0.3: previously any value was accepted into the
// boxed storage).
func (s *Series) Set(pos int, v any) error {
	if pos < 0 || pos >= s.Len() {
		return fmt.Errorf("%w: position %d for series of length %d", errs.ErrIndexOutOfBounds, pos, s.Len())
	}
	return s.col.SetValue(pos, v)
}

// Head returns the first n elements (all when n exceeds the length).
func (s *Series) Head(n int) *Series {
	if n > s.Len() {
		n = s.Len()
	}
	if n < 0 {
		n = 0
	}
	out, _ := s.Slice(0, n)
	return out
}

// Tail returns the last n elements.
func (s *Series) Tail(n int) *Series {
	if n > s.Len() {
		n = s.Len()
	}
	if n < 0 {
		n = 0
	}
	out, _ := s.Slice(s.Len()-n, s.Len())
	return out
}

// Slice returns positions [start, stop) as a new series.
func (s *Series) Slice(start, stop int) (*Series, error) {
	if start < 0 || stop < start || stop > s.Len() {
		return nil, fmt.Errorf("%w: slice [%d:%d] for series of length %d", errs.ErrIndexOutOfBounds, start, stop, s.Len())
	}
	positions := make([]int, 0, stop-start)
	for i := start; i < stop; i++ {
		positions = append(positions, i)
	}
	return s.Take(positions)
}

// Take selects positions into a new series. Negative positions produce
// missing values (used by joins/alignment).
func (s *Series) Take(pos []int) (*Series, error) {
	col, err := s.col.Take(pos)
	if err != nil {
		return nil, err
	}
	return &Series{
		name:  s.name,
		col:   col,
		index: index.Take(s.index, pos),
	}, nil
}

// WithIndexed returns a copy of the series with a replaced index.
func (s *Series) WithIndexed(idx index.Index) *Series {
	c := s.Copy()
	if idx != nil && idx.Len() == c.Len() {
		c.index = idx.Clone()
	}
	return c
}

// ResetIndex returns a copy with a fresh RangeIndex.
func (s *Series) ResetIndex() *Series {
	c := s.Copy()
	c.index = index.NewRangeIndex(c.Len())
	return c
}

package series

import (
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// Storage exposes the backing column for the columnar expression engine
// (v0.4). The column must be treated as read-only: expression kernels
// never mutate source storage.
func (s *Series) Storage() column.Column { return s.col }

// FromColumn assembles a series directly around a typed column, avoiding
// any boxing. The column is owned by the new series; pass a Copy if the
// caller keeps a reference.
func FromColumn(name string, col column.Column, idx index.Index) *Series {
	if idx != nil {
		idx = idx.Clone()
	}
	return fromColumn(name, col, idx)
}

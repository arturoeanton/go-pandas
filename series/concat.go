package series

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// Concat stacks series vertically with typed storage (v0.6.1):
// same-dtype inputs append into one typed buffer, compatible numeric
// dtypes promote, incompatible mixes fall back to object. The result
// carries a fresh RangeIndex and the first series' name.
func Concat(ss ...*Series) (*Series, error) {
	if len(ss) == 0 {
		return nil, fmt.Errorf("%w: ConcatSeries needs at least one series", errs.ErrInvalidOperation)
	}
	parts := make([]column.ConcatPart, len(ss))
	for i, s := range ss {
		if s == nil {
			return nil, fmt.Errorf("%w: nil series in ConcatSeries", errs.ErrInvalidOperation)
		}
		parts[i] = column.ConcatPart{Col: s.col, Len: s.Len()}
	}
	return fromColumn(ss[0].name, column.ConcatParts(parts), nil), nil
}

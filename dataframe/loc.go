package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// LocIndexer selects rows by index label and columns by name, like df.loc:
//
//	df.Loc().Rows("a", "b").Cols("name", "age").Get()
//	df.Loc().RowsBetween("a", "d").Get()   // inclusive label slice
type LocIndexer struct {
	df      *DataFrame
	rowPos  []int
	cols    []string
	hasRows bool
	hasCols bool
	err     error
}

// Loc starts a label-based selection.
func (df *DataFrame) Loc() *LocIndexer { return &LocIndexer{df: df} }

// Row returns a single row by index label as a map.
func (ix *LocIndexer) Row(label any) (map[string]any, error) {
	pos, ok := ix.df.index.Pos(label)
	if !ok {
		return nil, fmt.Errorf("%w: label %v", errs.ErrInvalidIndex, label)
	}
	return ix.df.Row(pos)
}

// Rows selects rows by explicit labels (duplicates select every match).
func (ix *LocIndexer) Rows(labels ...any) *LocIndexer {
	for _, label := range labels {
		positions := ix.df.index.Positions(label)
		if len(positions) == 0 {
			ix.err = fmt.Errorf("%w: label %v", errs.ErrInvalidIndex, label)
			return ix
		}
		ix.rowPos = append(ix.rowPos, positions...)
	}
	ix.hasRows = true
	return ix
}

// RowsBetween selects the inclusive label slice [start, stop], like
// df.loc["a":"d"].
func (ix *LocIndexer) RowsBetween(start, stop any) *LocIndexer {
	positions, err := ix.df.index.Slice(start, stop)
	if err != nil {
		ix.err = err
		return ix
	}
	ix.rowPos = append(ix.rowPos, positions...)
	ix.hasRows = true
	return ix
}

// Cols selects columns by name.
func (ix *LocIndexer) Cols(names ...string) *LocIndexer {
	ix.cols = append(ix.cols, names...)
	ix.hasCols = true
	return ix
}

// Get materializes the selection as a new frame.
func (ix *LocIndexer) Get() (*DataFrame, error) {
	if ix.err != nil {
		return nil, ix.err
	}
	out := ix.df
	if ix.hasCols {
		selected, err := out.Select(ix.cols...)
		if err != nil {
			return nil, err
		}
		out = selected
	}
	if ix.hasRows {
		taken, err := out.Take(ix.rowPos)
		if err != nil {
			return nil, err
		}
		out = taken
	}
	if out == ix.df {
		out = ix.df.Copy()
	}
	return out, nil
}

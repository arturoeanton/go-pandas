package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
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

// LabelRange is an inclusive label slice for Loc().Rows, built with
// LabelSlice(start, stop). Unlike positional slicing, BOTH endpoints are
// included, matching pandas df.loc["a":"z"].
type LabelRange struct {
	Start any
	Stop  any
}

// LabelSlice builds an inclusive label range.
func LabelSlice(start, stop any) LabelRange { return LabelRange{Start: start, Stop: stop} }

// Rows selects rows by explicit labels (duplicates select every match).
// A LabelSlice selector expands to the inclusive label range.
func (ix *LocIndexer) Rows(labels ...any) *LocIndexer {
	for _, label := range labels {
		if r, ok := label.(LabelRange); ok {
			positions, err := ix.df.index.Slice(r.Start, r.Stop)
			if err != nil {
				ix.err = err
				return ix
			}
			ix.rowPos = append(ix.rowPos, positions...)
			continue
		}
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

// Tuple selects the rows whose full MultiIndex label matches the tuple,
// like df.loc[("AR", "Buenos Aires")]. Duplicate tuples select every
// match; a nil component matches an NA tuple component. Accepts either
// spread components or a single pd.Tuple/[]any value.
func (ix *LocIndexer) Tuple(components ...any) *LocIndexer {
	mi, t, ok := ix.tupleArgs(components)
	if !ok {
		return ix
	}
	positions := mi.PositionsTuple(t)
	if len(positions) == 0 {
		ix.err = fmt.Errorf("%w: tuple %v", errs.ErrInvalidIndex, index.Tuple(t))
		return ix
	}
	ix.rowPos = append(ix.rowPos, positions...)
	ix.hasRows = true
	return ix
}

// TuplePrefix selects the rows whose leading MultiIndex components match
// the prefix, like df.loc[("AR", slice(None))]. v0.8 scans the code
// arrays (documented; full tuples use the lookup map).
func (ix *LocIndexer) TuplePrefix(components ...any) *LocIndexer {
	mi, t, ok := ix.tupleArgs(components)
	if !ok {
		return ix
	}
	positions := mi.PositionsPrefix(t)
	if len(positions) == 0 {
		ix.err = fmt.Errorf("%w: tuple prefix %v", errs.ErrInvalidIndex, index.Tuple(t))
		return ix
	}
	ix.rowPos = append(ix.rowPos, positions...)
	ix.hasRows = true
	return ix
}

// tupleArgs validates a tuple selection: the frame must carry a
// MultiIndex, and a single Tuple/[]any argument unwraps to components.
func (ix *LocIndexer) tupleArgs(components []any) (*index.MultiIndex, []any, bool) {
	mi, ok := ix.df.index.(*index.MultiIndex)
	if !ok {
		ix.err = fmt.Errorf("%w: Loc tuple selection needs a MultiIndex (index is %T)", errs.ErrInvalidIndex, ix.df.index)
		return nil, nil, false
	}
	if len(components) == 1 {
		switch t := components[0].(type) {
		case index.Tuple:
			return mi, t, true
		case []any:
			return mi, t, true
		}
	}
	return mi, components, true
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

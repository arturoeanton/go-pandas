package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// ILocIndexer selects rows and columns by position, like df.iloc. Build a
// selection with Rows/Cols — both accept ints, pd.Slice(...)/
// pd.SliceStep(...) specs and pd.All() — then materialize it with Get:
//
//	df.ILoc().Rows(0, 2, 4).Get()
//	df.ILoc().Rows(pd.Slice(0, 10)).Cols(pd.Slice(1, 3)).Get()
//	df.ILoc().Rows(pd.All()).Cols(0, 2).Get()
//
// Positional slices follow the Go convention: stop is exclusive.
type ILocIndexer struct {
	df      *DataFrame
	rowPos  []int
	colPos  []int
	hasRows bool
	hasCols bool
	err     error
}

// ILoc starts a positional selection.
func (df *DataFrame) ILoc() *ILocIndexer { return &ILocIndexer{df: df} }

// Row returns a single row by position as a map. Negative positions count
// from the end.
func (ix *ILocIndexer) Row(pos int) (map[string]any, error) {
	if pos < 0 {
		pos += ix.df.Len()
	}
	return ix.df.Row(pos)
}

// resolveSpec expands a SliceSpec over an axis of size n.
func resolveSpec(spec ndarray.SliceSpec, n int) ([]int, error) {
	start, stop, step := 0, n, 1
	if spec.Step != 0 {
		step = spec.Step
	}
	if step <= 0 {
		return nil, errs.NotImplemented("negative iloc step")
	}
	if spec.Start != nil {
		start = *spec.Start
		if start < 0 {
			start += n
		}
	}
	if spec.Stop != nil {
		stop = *spec.Stop
		if stop < 0 {
			stop += n
		}
	}
	if stop > n {
		stop = n
	}
	var out []int
	for i := start; i < stop; i += step {
		if i >= 0 {
			out = append(out, i)
		}
	}
	return out, nil
}

// appendPositions expands mixed selectors (int, SliceSpec) over an axis.
func appendPositions(dst []int, selectors []any, n int, what string) ([]int, error) {
	for _, sel := range selectors {
		switch v := sel.(type) {
		case int:
			p := v
			if p < 0 {
				p += n
			}
			if p < 0 || p >= n {
				return nil, fmt.Errorf("%w: %s %d out of range [0, %d)", errs.ErrIndexOutOfBounds, what, v, n)
			}
			dst = append(dst, p)
		case ndarray.SliceSpec:
			pos, err := resolveSpec(v, n)
			if err != nil {
				return nil, err
			}
			dst = append(dst, pos...)
		default:
			return nil, fmt.Errorf("%w: iloc selector must be int or SliceSpec, got %T", errs.ErrInvalidOperation, sel)
		}
	}
	return dst, nil
}

// Rows selects rows by position: ints, pd.Slice specs and pd.All() may be
// mixed.
func (ix *ILocIndexer) Rows(selectors ...any) *ILocIndexer {
	pos, err := appendPositions(ix.rowPos, selectors, ix.df.Len(), "row")
	if err != nil {
		ix.err = err
		return ix
	}
	ix.rowPos = pos
	ix.hasRows = true
	return ix
}

// RowsAt selects explicit row positions (alias kept for clarity).
func (ix *ILocIndexer) RowsAt(positions ...int) *ILocIndexer {
	selectors := make([]any, len(positions))
	for i, p := range positions {
		selectors[i] = p
	}
	return ix.Rows(selectors...)
}

// Cols selects columns by position: ints, pd.Slice specs and pd.All() may
// be mixed.
func (ix *ILocIndexer) Cols(selectors ...any) *ILocIndexer {
	pos, err := appendPositions(ix.colPos, selectors, len(ix.df.columns), "column")
	if err != nil {
		ix.err = err
		return ix
	}
	ix.colPos = pos
	ix.hasCols = true
	return ix
}

// ColsRange selects a positional range of columns (alias kept for
// clarity).
func (ix *ILocIndexer) ColsRange(spec ndarray.SliceSpec) *ILocIndexer {
	return ix.Cols(spec)
}

// Get materializes the selection as a new frame.
func (ix *ILocIndexer) Get() (*DataFrame, error) {
	if ix.err != nil {
		return nil, ix.err
	}
	out := ix.df
	if ix.hasCols {
		names := make([]string, len(ix.colPos))
		for i, p := range ix.colPos {
			names[i] = out.columns[p].Name()
		}
		selected, err := out.Select(names...)
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

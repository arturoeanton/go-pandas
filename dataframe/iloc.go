package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// ILocIndexer selects rows and columns by position, like df.iloc. Build a
// selection with Rows/RowsAt/Cols/ColsRange and materialize it with Get:
//
//	df.ILoc().Rows(pd.Slice(0, 10)).Cols(1, 2).Get()
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

// Row returns a single row by position as a map.
func (ix *ILocIndexer) Row(pos int) (map[string]any, error) {
	if pos < 0 {
		pos += ix.df.Len()
	}
	return ix.df.Row(pos)
}

// Rows selects a positional range of rows.
func (ix *ILocIndexer) Rows(spec ndarray.SliceSpec) *ILocIndexer {
	n := ix.df.Len()
	start, stop, step := 0, n, 1
	if spec.Step != 0 {
		step = spec.Step
	}
	if step <= 0 {
		ix.err = errs.NotImplemented("negative iloc step")
		return ix
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
	for i := start; i < stop; i += step {
		if i >= 0 {
			ix.rowPos = append(ix.rowPos, i)
		}
	}
	ix.hasRows = true
	return ix
}

// RowsAt selects explicit row positions.
func (ix *ILocIndexer) RowsAt(positions ...int) *ILocIndexer {
	n := ix.df.Len()
	for _, p := range positions {
		if p < 0 {
			p += n
		}
		if p < 0 || p >= n {
			ix.err = fmt.Errorf("%w: row %d for frame of length %d", errs.ErrIndexOutOfBounds, p, n)
			return ix
		}
		ix.rowPos = append(ix.rowPos, p)
	}
	ix.hasRows = true
	return ix
}

// Cols selects explicit column positions.
func (ix *ILocIndexer) Cols(positions ...int) *ILocIndexer {
	n := len(ix.df.columns)
	for _, p := range positions {
		if p < 0 {
			p += n
		}
		if p < 0 || p >= n {
			ix.err = fmt.Errorf("%w: column %d for frame with %d columns", errs.ErrIndexOutOfBounds, p, n)
			return ix
		}
		ix.colPos = append(ix.colPos, p)
	}
	ix.hasCols = true
	return ix
}

// ColsRange selects a positional range of columns.
func (ix *ILocIndexer) ColsRange(spec ndarray.SliceSpec) *ILocIndexer {
	n := len(ix.df.columns)
	start, stop, step := 0, n, 1
	if spec.Step != 0 {
		step = spec.Step
	}
	if step <= 0 {
		ix.err = errs.NotImplemented("negative iloc step")
		return ix
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
	for i := start; i < stop; i += step {
		if i >= 0 {
			ix.colPos = append(ix.colPos, i)
		}
	}
	ix.hasCols = true
	return ix
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

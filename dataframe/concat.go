package dataframe

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	"github.com/arturoeanton/go-pandas/series"
)

// ConcatOptions mirrors pd.concat keyword arguments.
type ConcatOptions struct {
	// Axis 0 stacks rows (default); axis 1 concatenates columns.
	Axis int
	// Join is "outer" (column union, default) or "inner" (intersection).
	Join string
	// IgnoreIndex resets the result to a RangeIndex.
	IgnoreIndex bool
}

// ConcatOption mutates ConcatOptions.
type ConcatOption func(*ConcatOptions)

// ConcatAxis sets the concatenation axis.
func ConcatAxis(axis int) ConcatOption { return func(o *ConcatOptions) { o.Axis = axis } }

// ConcatJoin sets outer/inner column handling for axis 0.
func ConcatJoin(join string) ConcatOption { return func(o *ConcatOptions) { o.Join = join } }

// ConcatIgnoreIndex resets the result index.
func ConcatIgnoreIndex(v bool) ConcatOption {
	return func(o *ConcatOptions) { o.IgnoreIndex = v }
}

// Concat concatenates frames vertically (axis 0) or horizontally
// (axis 1). Since v0.6.1 vertical concat is typed: same-dtype columns
// append into one typed buffer, compatible numeric dtypes promote once,
// columns missing from a frame become NA gaps, and only genuinely
// incompatible columns fall back to object storage.
func Concat(frames []*DataFrame, opts ...ConcatOption) (*DataFrame, error) {
	o := ConcatOptions{Join: "outer"}
	for _, f := range opts {
		f(&o)
	}
	var nonEmpty []*DataFrame
	for _, f := range frames {
		if f != nil {
			nonEmpty = append(nonEmpty, f)
		}
	}
	if len(nonEmpty) == 0 {
		return newFrame(nil, nil)
	}
	if o.Axis == 1 {
		return concatColumns(nonEmpty)
	}
	return concatRows(nonEmpty, o)
}

// concatRows implements the typed axis=0 concat.
func concatRows(frames []*DataFrame, o ConcatOptions) (*DataFrame, error) {
	// Column order: union in first-seen order, or intersection for inner.
	var names []string
	seen := map[string]bool{}
	for _, f := range frames {
		for _, name := range f.Columns() {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	if o.Join == "inner" {
		var kept []string
		for _, name := range names {
			inAll := true
			for _, f := range frames {
				if _, ok := f.byName[name]; !ok {
					inAll = false
					break
				}
			}
			if inAll {
				kept = append(kept, name)
			}
		}
		names = kept
	}

	total := 0
	for _, f := range frames {
		total += f.Len()
	}
	var idx index.Index
	if o.IgnoreIndex {
		idx = index.NewRangeIndex(total)
	} else {
		idx = concatIndexes(frames, total)
	}

	cols := make([]*series.Series, len(names))
	parts := make([]column.ConcatPart, len(frames))
	for j, name := range names {
		for i, f := range frames {
			if k, ok := f.byName[name]; ok {
				parts[i] = column.ConcatPart{Col: f.columns[k].Storage(), Len: f.Len()}
			} else {
				parts[i] = column.ConcatPart{Col: nil, Len: f.Len()}
			}
		}
		cols[j] = series.Assemble(name, column.ConcatParts(parts), idx)
	}
	return newFrame(cols, idx)
}

// concatIndexes stacks the row labels, staying typed when every frame
// carries the same label family (integer, string or datetime); mixed
// families keep the boxed labels as-is.
func concatIndexes(frames []*DataFrame, total int) index.Index {
	allInt, allString, allTime := true, true, true
	for _, f := range frames {
		switch f.index.(type) {
		case *index.RangeIndex, *index.Int64Index:
			allString, allTime = false, false
		case *index.StringIndex:
			allInt, allTime = false, false
		case *index.DatetimeIndex:
			allInt, allString = false, false
		default:
			allInt, allString, allTime = false, false, false
		}
	}
	switch {
	case allInt:
		labels := make([]int64, 0, total)
		for _, f := range frames {
			switch ix := f.index.(type) {
			case *index.RangeIndex:
				for i := 0; i < ix.Len(); i++ {
					labels = append(labels, int64(ix.Start+i*ix.Step))
				}
			case *index.Int64Index:
				labels = append(labels, ix.Int64s()...)
			}
		}
		return index.NewInt64Index(labels)
	case allString:
		labels := make([]string, 0, total)
		for _, f := range frames {
			labels = append(labels, f.index.(*index.StringIndex).Strings()...)
		}
		return index.NewStringIndex(labels)
	case allTime:
		labels := make([]time.Time, 0, total)
		for _, f := range frames {
			labels = append(labels, f.index.(*index.DatetimeIndex).Times()...)
		}
		return index.NewDatetimeIndex(labels)
	}
	labels := make([]any, 0, total)
	for _, f := range frames {
		labels = append(labels, f.index.Values()...)
	}
	return indexFromLabels(labels)
}

// concatColumns concatenates frames side by side; row counts must match
// (no index alignment — a documented limitation).
func concatColumns(frames []*DataFrame) (*DataFrame, error) {
	n := frames[0].Len()
	idx := frames[0].index.Clone()
	var cols []*series.Series
	seen := map[string]int{}
	for _, f := range frames {
		if f.Len() != n {
			return nil, fmt.Errorf("%w: concat axis=1 with row counts %d and %d", errs.ErrLengthMismatch, n, f.Len())
		}
		for _, c := range f.columns {
			name := c.Name()
			if k, dup := seen[name]; dup {
				seen[name] = k + 1
				name = fmt.Sprintf("%s_%d", name, k+1)
			} else {
				seen[name] = 0
			}
			cols = append(cols, series.Assemble(name, c.Storage().Copy(), idx))
		}
	}
	return newFrame(cols, idx)
}

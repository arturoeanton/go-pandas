package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
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

// Concat concatenates frames vertically (axis 0) or horizontally (axis 1).
// Vertical concat takes the column union and fills missing cells with NA.
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

	// Column order: union in first-seen order, or intersection for inner.
	var names []string
	seen := map[string]bool{}
	for _, f := range nonEmpty {
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
			for _, f := range nonEmpty {
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
	for _, f := range nonEmpty {
		total += f.Len()
	}
	colData := make([][]any, len(names))
	for j := range colData {
		colData[j] = make([]any, 0, total)
	}
	var labels []any
	for _, f := range nonEmpty {
		values := make(map[string][]any, len(f.columns))
		for _, c := range f.columns {
			values[c.Name()] = c.Values()
		}
		for i := 0; i < f.Len(); i++ {
			labels = append(labels, f.index.At(i))
		}
		for j, name := range names {
			if vs, ok := values[name]; ok {
				colData[j] = append(colData[j], vs...)
			} else {
				for i := 0; i < f.Len(); i++ {
					colData[j] = append(colData[j], nil)
				}
			}
		}
	}
	cols := make([]*series.Series, len(names))
	for j, name := range names {
		cols[j] = series.NewSeries(name, colData[j])
	}
	out, err := newFrame(cols, nil)
	if err != nil {
		return nil, err
	}
	if !o.IgnoreIndex {
		idx := indexFromLabels(labels)
		adjusted := make([]*series.Series, len(out.columns))
		for i, c := range out.columns {
			adjusted[i] = c.WithIndexed(idx)
		}
		return newFrame(adjusted, idx)
	}
	return out, nil
}

// concatColumns concatenates frames side by side; row counts must match.
func concatColumns(frames []*DataFrame) (*DataFrame, error) {
	n := frames[0].Len()
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
			cols = append(cols, c.Copy().Rename(name))
		}
	}
	return newFrame(cols, frames[0].index.Clone())
}

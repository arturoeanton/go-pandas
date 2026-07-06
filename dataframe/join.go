package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// JoinOptions mirrors df.join keyword arguments.
type JoinOptions struct {
	// On joins on a left column against the other frame's index; empty
	// means join index-on-index.
	On string
	// How is left (default), inner or outer.
	How string
	// LSuffix/RSuffix disambiguate overlapping column names.
	LSuffix string
	RSuffix string
}

// Join combines two frames on their indexes (or a left column against the
// other frame's index).
func (df *DataFrame) Join(other *DataFrame, opts JoinOptions) (*DataFrame, error) {
	how := opts.How
	if how == "" {
		how = "left"
	}
	switch how {
	case "left", "inner", "outer":
	default:
		return nil, fmt.Errorf("%w: join how=%q", errs.ErrInvalidJoin, how)
	}
	// Left labels: either a column's values or the index labels.
	var leftLabels []any
	if opts.On != "" {
		c, err := df.Col(opts.On)
		if err != nil {
			return nil, err
		}
		leftLabels = c.Values()
	} else {
		leftLabels = df.index.Values()
	}

	var leftPos, rightPos []int
	matchedRight := make([]bool, other.Len())
	for i, label := range leftLabels {
		positions := other.index.Positions(label)
		if len(positions) == 0 {
			if how == "left" || how == "outer" {
				leftPos = append(leftPos, i)
				rightPos = append(rightPos, -1)
			}
			continue
		}
		for _, j := range positions {
			matchedRight[j] = true
			leftPos = append(leftPos, i)
			rightPos = append(rightPos, j)
		}
	}
	if how == "outer" {
		for j := range matchedRight {
			if !matchedRight[j] {
				leftPos = append(leftPos, -1)
				rightPos = append(rightPos, j)
			}
		}
	}

	dup := func(name string) bool {
		_, inLeft := df.byName[name]
		_, inRight := other.byName[name]
		return inLeft && inRight
	}
	var cols []*series.Series
	for _, c := range df.columns {
		name := c.Name()
		if dup(name) {
			if opts.LSuffix == "" && opts.RSuffix == "" {
				return nil, fmt.Errorf("%w: overlapping column %q requires LSuffix/RSuffix", errs.ErrInvalidJoin, name)
			}
			name += opts.LSuffix
		}
		out, err := c.Take(leftPos)
		if err != nil {
			return nil, err
		}
		cols = append(cols, out.Rename(name))
	}
	for _, c := range other.columns {
		name := c.Name()
		if dup(name) {
			name += opts.RSuffix
		}
		out, err := c.Take(rightPos)
		if err != nil {
			return nil, err
		}
		cols = append(cols, out.Rename(name))
	}
	// Result index: left labels where available, else right labels.
	idx := index.Take(df.index, leftPos)
	if opts.On == "" {
		labels := make([]any, len(leftPos))
		for i := range leftPos {
			if leftPos[i] >= 0 {
				labels[i] = df.index.At(leftPos[i])
			} else {
				labels[i] = other.index.At(rightPos[i])
			}
		}
		idx = indexFromLabels(labels)
	}
	adjusted := make([]*series.Series, len(cols))
	for i, c := range cols {
		adjusted[i] = c.WithIndexed(idx)
	}
	return newFrame(adjusted, idx)
}

// indexFromLabels rebuilds an index from raw labels via a string index
// fallback.
func indexFromLabels(labels []any) index.Index {
	strs := make([]string, len(labels))
	allStrings := true
	for i, v := range labels {
		if s, ok := v.(string); ok {
			strs[i] = s
		} else {
			allStrings = false
			break
		}
	}
	if allStrings {
		return index.NewStringIndex(strs)
	}
	strs = make([]string, len(labels))
	for i, v := range labels {
		strs[i] = fmt.Sprint(v)
	}
	return index.NewStringIndex(strs)
}

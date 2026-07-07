package dataframe

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	join "github.com/arturoeanton/go-pandas/internal/join"
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

// indexKeyColumn converts an index's labels into a typed key column for
// the join engine (RangeIndex labels are generated arithmetically;
// heterogeneous indexes fall back to boxed storage).
func indexKeyColumn(ix index.Index) column.Column {
	switch typed := ix.(type) {
	case *index.RangeIndex:
		values := make([]int64, typed.Len())
		for i := range values {
			values[i] = int64(typed.Start + i*typed.Step)
		}
		return column.NewInt64(values, nil)
	case *index.Int64Index:
		return column.NewInt64(append([]int64(nil), typed.Int64s()...), nil)
	case *index.StringIndex:
		return column.NewString(append([]string(nil), typed.Strings()...), nil)
	case *index.DatetimeIndex:
		return column.NewTime(append([]time.Time(nil), typed.Times()...), nil)
	}
	return column.FromAny(ix.Values(), 0)
}

// Join combines two frames on their indexes (or a left column against the
// other frame's index) through the typed join engine (v0.6).
func (df *DataFrame) Join(other *DataFrame, opts JoinOptions) (*DataFrame, error) {
	how := opts.How
	if how == "" {
		how = "left"
	}
	var jhow join.How
	switch how {
	case "left":
		jhow = join.Left
	case "inner":
		jhow = join.Inner
	case "outer":
		jhow = join.Outer
	default:
		return nil, fmt.Errorf("%w: join how=%q", errs.ErrInvalidJoin, how)
	}

	// Left keys: a column's storage or the index labels.
	var leftKey column.Column
	if opts.On != "" {
		c, err := df.Col(opts.On)
		if err != nil {
			return nil, err
		}
		leftKey = c.Storage()
	} else {
		leftKey = indexKeyColumn(df.index)
	}
	rightKey := indexKeyColumn(other.index)

	lids, rids, count := join.PairIDs(
		[]column.Column{leftKey}, []column.Column{rightKey})
	plan := join.Build(jhow, lids, rids, count)

	dup := func(name string) bool {
		_, inLeft := df.byName[name]
		_, inRight := other.byName[name]
		return inLeft && inRight
	}
	n := len(plan.LeftRows)

	// Result index: left labels where available, else right labels.
	var idx index.Index
	if opts.On != "" {
		idx = index.Take(df.index, plan.LeftRows)
	} else if jhow == join.Outer {
		labels := make([]any, n)
		for i := range plan.LeftRows {
			if plan.LeftRows[i] >= 0 {
				labels[i] = df.index.At(plan.LeftRows[i])
			} else {
				labels[i] = other.index.At(plan.RightRows[i])
			}
		}
		idx = indexFromLabels(labels)
	} else {
		idx = index.Take(df.index, plan.LeftRows)
	}

	var cols []*series.Series
	appendSide := func(src *DataFrame, rows []int, suffix string, left bool) error {
		for _, c := range src.columns {
			name := c.Name()
			if dup(name) {
				if opts.LSuffix == "" && opts.RSuffix == "" {
					return fmt.Errorf("%w: overlapping column %q requires LSuffix/RSuffix", errs.ErrInvalidJoin, name)
				}
				name += suffix
			}
			out, err := c.Storage().Take(rows)
			if err != nil {
				return err
			}
			cols = append(cols, series.Assemble(name, out, idx))
		}
		return nil
	}
	if err := appendSide(df, plan.LeftRows, opts.LSuffix, true); err != nil {
		return nil, err
	}
	if err := appendSide(other, plan.RightRows, opts.RSuffix, false); err != nil {
		return nil, err
	}
	return newFrame(cols, idx)
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

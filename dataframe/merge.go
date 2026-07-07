package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	join "github.com/arturoeanton/go-pandas/internal/join"
	"github.com/arturoeanton/go-pandas/series"
)

// MergeOptions mirrors pd.merge keyword arguments.
type MergeOptions struct {
	// On names key columns present in both frames.
	On []string
	// LeftOn/RightOn name differing key columns per side.
	LeftOn  []string
	RightOn []string
	// How is inner (default), left, right, outer or cross.
	How string
	// Suffixes disambiguate duplicated non-key columns (default _x, _y).
	Suffixes [2]string
	// Validate is "", one_to_one, one_to_many, many_to_one or many_to_many.
	Validate string
	// Indicator adds a _merge column with both/left_only/right_only.
	Indicator bool
}

// Merge is the package-level merge (pd.merge).
func Merge(left, right *DataFrame, opts MergeOptions) (*DataFrame, error) {
	return left.Merge(right, opts)
}

// Merge joins two frames on key columns through the typed hash-join
// engine (v0.6): keys map into a shared typed id space, pair vectors are
// built once, and output columns materialize via typed gather. NA keys
// never match (see known differences).
func (df *DataFrame) Merge(right *DataFrame, opts MergeOptions) (*DataFrame, error) {
	how := opts.How
	if how == "" {
		how = "inner"
	}
	if opts.Suffixes == [2]string{} {
		opts.Suffixes = [2]string{"_x", "_y"}
	}

	if how == "cross" {
		plan := join.Cross(df.Len(), right.Len())
		return materializeCross(df, right, plan, opts)
	}

	var jhow join.How
	switch how {
	case "inner":
		jhow = join.Inner
	case "left":
		jhow = join.Left
	case "right":
		jhow = join.Right
	case "outer":
		jhow = join.Outer
	default:
		return nil, fmt.Errorf("%w: how=%q", errs.ErrInvalidJoin, how)
	}

	leftKeys, rightKeys := opts.LeftOn, opts.RightOn
	if len(opts.On) > 0 {
		leftKeys, rightKeys = opts.On, opts.On
	}
	if len(leftKeys) == 0 || len(leftKeys) != len(rightKeys) {
		return nil, fmt.Errorf("%w: merge requires matching On or LeftOn/RightOn keys", errs.ErrInvalidJoin)
	}
	leftCols := make([]column.Column, len(leftKeys))
	rightCols := make([]column.Column, len(rightKeys))
	for i := range leftKeys {
		lc, err := df.Col(leftKeys[i])
		if err != nil {
			return nil, err
		}
		rc, err := right.Col(rightKeys[i])
		if err != nil {
			return nil, err
		}
		leftCols[i] = lc.Storage()
		rightCols[i] = rc.Storage()
	}

	lids, rids, count := join.PairIDs(leftCols, rightCols)
	if err := join.Validate(opts.Validate, lids, rids, count); err != nil {
		return nil, err
	}
	plan := join.Build(jhow, lids, rids, count)
	return materializeMerge(df, right, leftKeys, rightKeys, plan, opts)
}

// materializeMerge assembles the output frame from the pair vectors:
// key columns (typed coalesce of both sides when key names match), left
// non-keys, right non-keys — with suffixes on collisions — plus the
// optional indicator, all through typed gathers sharing one index.
func materializeMerge(left, right *DataFrame, leftKeys, rightKeys []string, plan *join.Plan, opts MergeOptions) (*DataFrame, error) {
	isLeftKey := make(map[string]bool, len(leftKeys))
	for _, k := range leftKeys {
		isLeftKey[k] = true
	}
	isRightKey := make(map[string]bool, len(rightKeys))
	for _, k := range rightKeys {
		isRightKey[k] = true
	}
	sameKeyNames := len(opts.On) > 0

	n := len(plan.LeftRows)
	idx := index.NewRangeIndex(n)
	var cols []*series.Series

	// Key columns: left values, filled from the right when the left side
	// has no matching row (outer/right joins with shared key names).
	for ki, k := range leftKeys {
		lc := left.MustCol(k).Storage()
		if sameKeyNames {
			rc := right.MustCol(rightKeys[ki]).Storage()
			out, ok := column.GatherCoalesce(lc, rc, plan.LeftRows, plan.RightRows)
			if !ok {
				out = column.GatherCoalesceBoxed(lc, rc, plan.LeftRows, plan.RightRows)
			}
			cols = append(cols, series.Assemble(k, out, idx))
			continue
		}
		out, err := lc.Take(plan.LeftRows)
		if err != nil {
			return nil, err
		}
		cols = append(cols, series.Assemble(k, out, idx))
	}

	dupName := func(name string) bool {
		if sameKeyNames && isLeftKey[name] {
			return false
		}
		_, inLeft := left.byName[name]
		_, inRight := right.byName[name]
		return inLeft && inRight
	}

	appendSide := func(src *DataFrame, rows []int, suffix string, skip map[string]bool) error {
		for _, c := range src.columns {
			if skip[c.Name()] {
				continue
			}
			name := c.Name()
			if dupName(name) {
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
	if err := appendSide(left, plan.LeftRows, opts.Suffixes[0], isLeftKey); err != nil {
		return nil, err
	}
	if err := appendSide(right, plan.RightRows, opts.Suffixes[1], isRightKey); err != nil {
		return nil, err
	}

	if opts.Indicator {
		values := make([]string, n)
		for i, m := range plan.Match {
			switch m {
			case join.MatchLeftOnly:
				values[i] = "left_only"
			case join.MatchRightOnly:
				values[i] = "right_only"
			default:
				values[i] = "both"
			}
		}
		cols = append(cols, series.Assemble("_merge", column.NewString(values, nil), idx))
	}
	return newFrame(cols, idx)
}

// materializeCross assembles the cartesian product with suffixes on every
// duplicated column name.
func materializeCross(left, right *DataFrame, plan *join.Plan, opts MergeOptions) (*DataFrame, error) {
	n := len(plan.LeftRows)
	idx := index.NewRangeIndex(n)
	dup := func(name string) bool {
		_, inLeft := left.byName[name]
		_, inRight := right.byName[name]
		return inLeft && inRight
	}
	var cols []*series.Series
	appendSide := func(src *DataFrame, rows []int, suffix string) error {
		for _, c := range src.columns {
			name := c.Name()
			if dup(name) {
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
	if err := appendSide(left, plan.LeftRows, opts.Suffixes[0]); err != nil {
		return nil, err
	}
	if err := appendSide(right, plan.RightRows, opts.Suffixes[1]); err != nil {
		return nil, err
	}
	return newFrame(cols, idx)
}

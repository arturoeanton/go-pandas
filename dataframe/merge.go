package dataframe

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
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

// Merge joins two frames on key columns using a hash join.
func (df *DataFrame) Merge(right *DataFrame, opts MergeOptions) (*DataFrame, error) {
	how := opts.How
	if how == "" {
		how = "inner"
	}
	switch how {
	case "inner", "left", "right", "outer", "cross":
	default:
		return nil, fmt.Errorf("%w: how=%q", errs.ErrInvalidJoin, how)
	}
	if opts.Suffixes == [2]string{} {
		opts.Suffixes = [2]string{"_x", "_y"}
	}

	if how == "cross" {
		return crossJoin(df, right, opts)
	}

	leftKeys, rightKeys := opts.LeftOn, opts.RightOn
	if len(opts.On) > 0 {
		leftKeys, rightKeys = opts.On, opts.On
	}
	if len(leftKeys) == 0 || len(leftKeys) != len(rightKeys) {
		return nil, fmt.Errorf("%w: merge requires matching On or LeftOn/RightOn keys", errs.ErrInvalidJoin)
	}
	leftKeyCols := make([][]any, len(leftKeys))
	for i, k := range leftKeys {
		c, err := df.Col(k)
		if err != nil {
			return nil, err
		}
		leftKeyCols[i] = c.Values()
	}
	rightKeyCols := make([][]any, len(rightKeys))
	for i, k := range rightKeys {
		c, err := right.Col(k)
		if err != nil {
			return nil, err
		}
		rightKeyCols[i] = c.Values()
	}

	makeKey := func(cols [][]any, row int) (string, bool) {
		var sb strings.Builder
		for _, col := range cols {
			v := col[row]
			if dtype.IsNA(v) {
				return "", false
			}
			// Normalize numerics so int 1 matches float 1.0 across frames.
			if f, ok := dtype.AsFloat(v); ok {
				sb.WriteString(fmt.Sprintf("%v\x00", f))
			} else {
				sb.WriteString(fmt.Sprintf("%v\x00", v))
			}
		}
		return sb.String(), true
	}

	// Hash the right side: key -> row positions.
	rightRows := make(map[string][]int)
	for i := 0; i < right.Len(); i++ {
		if key, ok := makeKey(rightKeyCols, i); ok {
			rightRows[key] = append(rightRows[key], i)
		}
	}

	if err := validateMerge(opts.Validate, df, right, leftKeyCols, rightKeyCols, makeKey); err != nil {
		return nil, err
	}

	// Build the matched row position pairs (-1 = no match on that side).
	var leftPos, rightPos []int
	var indicator []string
	matchedRight := make([]bool, right.Len())
	for i := 0; i < df.Len(); i++ {
		key, ok := makeKey(leftKeyCols, i)
		var matches []int
		if ok {
			matches = rightRows[key]
		}
		if len(matches) == 0 {
			if how == "left" || how == "outer" {
				leftPos = append(leftPos, i)
				rightPos = append(rightPos, -1)
				indicator = append(indicator, "left_only")
			}
			continue
		}
		for _, j := range matches {
			matchedRight[j] = true
			leftPos = append(leftPos, i)
			rightPos = append(rightPos, j)
			indicator = append(indicator, "both")
		}
	}
	if how == "right" || how == "outer" {
		for j := 0; j < right.Len(); j++ {
			if !matchedRight[j] {
				leftPos = append(leftPos, -1)
				rightPos = append(rightPos, j)
				indicator = append(indicator, "right_only")
			}
		}
	}
	if how == "right" {
		// Keep only matched pairs and right-only rows, ordered by right row.
		var lp, rp []int
		var ind []string
		for j := 0; j < right.Len(); j++ {
			found := false
			for k, r := range rightPos {
				if r == j {
					lp = append(lp, leftPos[k])
					rp = append(rp, j)
					ind = append(ind, indicator[k])
					found = true
				}
			}
			if !found {
				lp = append(lp, -1)
				rp = append(rp, j)
				ind = append(ind, "right_only")
			}
		}
		leftPos, rightPos, indicator = lp, rp, ind
	}

	return assembleMerge(df, right, leftKeys, rightKeys, leftPos, rightPos, indicator, opts)
}

// validateMerge enforces the Validate cardinality constraint.
func validateMerge(validate string, left, right *DataFrame, leftKeyCols, rightKeyCols [][]any, makeKey func([][]any, int) (string, bool)) error {
	if validate == "" || validate == "many_to_many" {
		return nil
	}
	uniqueSide := func(frame *DataFrame, cols [][]any) bool {
		seen := make(map[string]bool)
		for i := 0; i < frame.Len(); i++ {
			key, ok := makeKey(cols, i)
			if !ok {
				continue
			}
			if seen[key] {
				return false
			}
			seen[key] = true
		}
		return true
	}
	switch validate {
	case "one_to_one":
		if !uniqueSide(left, leftKeyCols) || !uniqueSide(right, rightKeyCols) {
			return fmt.Errorf("%w: merge keys are not one_to_one", errs.ErrInvalidJoin)
		}
	case "one_to_many":
		if !uniqueSide(left, leftKeyCols) {
			return fmt.Errorf("%w: left merge keys are not unique for one_to_many", errs.ErrInvalidJoin)
		}
	case "many_to_one":
		if !uniqueSide(right, rightKeyCols) {
			return fmt.Errorf("%w: right merge keys are not unique for many_to_one", errs.ErrInvalidJoin)
		}
	default:
		return fmt.Errorf("%w: validate=%q", errs.ErrInvalidJoin, validate)
	}
	return nil
}

// assembleMerge builds the output frame from the matched position pairs:
// key columns (coalesced), left non-keys, right non-keys, with suffixes on
// name collisions.
func assembleMerge(left, right *DataFrame, leftKeys, rightKeys []string, leftPos, rightPos []int, indicator []string, opts MergeOptions) (*DataFrame, error) {
	isLeftKey := make(map[string]bool, len(leftKeys))
	for _, k := range leftKeys {
		isLeftKey[k] = true
	}
	isRightKey := make(map[string]bool, len(rightKeys))
	for _, k := range rightKeys {
		isRightKey[k] = true
	}
	sameKeyNames := len(opts.On) > 0

	var cols []*series.Series

	// Key columns: values from the left, filled from the right when the
	// left side has no match (outer/right joins).
	for ki, k := range leftKeys {
		lc, _ := left.Col(k)
		values := make([]any, len(leftPos))
		for i := range leftPos {
			if leftPos[i] >= 0 {
				v, _ := lc.At(leftPos[i])
				values[i] = v
			} else if sameKeyNames {
				rc, _ := right.Col(rightKeys[ki])
				v, _ := rc.At(rightPos[i])
				values[i] = v
			}
		}
		cols = append(cols, series.NewSeries(k, values))
	}

	dupName := func(name string) bool {
		_, inLeft := left.byName[name]
		_, inRight := right.byName[name]
		if isLeftKey[name] && sameKeyNames {
			return false
		}
		return inLeft && inRight
	}

	takeCol := func(c *series.Series, pos []int, suffix string) *series.Series {
		name := c.Name()
		if dupName(name) {
			name += suffix
		}
		out, _ := c.Take(pos)
		return out.Rename(name)
	}

	for _, c := range left.columns {
		if sameKeyNames && isLeftKey[c.Name()] {
			continue
		}
		if !sameKeyNames && isLeftKey[c.Name()] {
			continue
		}
		cols = append(cols, takeCol(c, leftPos, opts.Suffixes[0]))
	}
	for _, c := range right.columns {
		if isRightKey[c.Name()] {
			continue
		}
		cols = append(cols, takeCol(c, rightPos, opts.Suffixes[1]))
	}
	if opts.Indicator {
		values := make([]any, len(indicator))
		for i, v := range indicator {
			values[i] = v
		}
		cols = append(cols, series.NewSeries("_merge", values))
	}
	return newFrame(cols, nil)
}

// crossJoin returns the cartesian product of both frames.
func crossJoin(left, right *DataFrame, opts MergeOptions) (*DataFrame, error) {
	var leftPos, rightPos []int
	for i := 0; i < left.Len(); i++ {
		for j := 0; j < right.Len(); j++ {
			leftPos = append(leftPos, i)
			rightPos = append(rightPos, j)
		}
	}
	var cols []*series.Series
	dup := func(name string) bool {
		_, inLeft := left.byName[name]
		_, inRight := right.byName[name]
		return inLeft && inRight
	}
	for _, c := range left.columns {
		name := c.Name()
		if dup(name) {
			name += opts.Suffixes[0]
		}
		out, _ := c.Take(leftPos)
		cols = append(cols, out.Rename(name))
	}
	for _, c := range right.columns {
		name := c.Name()
		if dup(name) {
			name += opts.Suffixes[1]
		}
		out, _ := c.Take(rightPos)
		cols = append(cols, out.Rename(name))
	}
	return newFrame(cols, nil)
}

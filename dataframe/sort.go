package dataframe

import (
	"fmt"
	"sort"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// SortValues sorts rows by one column; missing values go last. Stable.
func (df *DataFrame) SortValues(column string, ascending bool) (*DataFrame, error) {
	return df.SortValuesBy([]string{column}, []bool{ascending})
}

// SortValuesBy sorts rows by multiple columns with per-column direction.
func (df *DataFrame) SortValuesBy(columns []string, ascending []bool) (*DataFrame, error) {
	if len(columns) == 0 {
		return df.Copy(), nil
	}
	if len(ascending) != len(columns) {
		return nil, fmt.Errorf("%w: %d columns but %d directions", errs.ErrLengthMismatch, len(columns), len(ascending))
	}
	type sortKey struct {
		na  func(i int) bool
		cmp func(a, b int) (int, bool)
	}
	keys := make([]sortKey, len(columns))
	for k, name := range columns {
		c, err := df.Col(name)
		if err != nil {
			return nil, err
		}
		if cc, ok := column.AsCategorical(c.Storage()); ok {
			// Categorical keys order by category rank on raw codes —
			// no boxing, no value comparisons (v0.7).
			codes, mask := cc.RawCodes()
			keys[k] = sortKey{
				na: func(i int) bool { return mask[i] },
				cmp: func(a, b int) (int, bool) {
					switch {
					case codes[a] < codes[b]:
						return -1, true
					case codes[a] > codes[b]:
						return 1, true
					}
					return 0, true
				},
			}
			continue
		}
		vals := c.Values()
		keys[k] = sortKey{
			na:  func(i int) bool { return dtype.IsNA(vals[i]) },
			cmp: func(a, b int) (int, bool) { return expr.CompareValues(vals[a], vals[b]) },
		}
	}
	pos := make([]int, df.Len())
	for i := range pos {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		for k := range keys {
			naA, naB := keys[k].na(pos[a]), keys[k].na(pos[b])
			if naA || naB {
				if naA && naB {
					continue
				}
				return naB // NA always last
			}
			c, ok := keys[k].cmp(pos[a], pos[b])
			if !ok || c == 0 {
				continue
			}
			if ascending[k] {
				return c < 0
			}
			return c > 0
		}
		return false
	})
	return df.Take(pos)
}

// SortIndex sorts rows by the index labels.
func (df *DataFrame) SortIndex(ascending bool) (*DataFrame, error) {
	pos := make([]int, df.Len())
	for i := range pos {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		c, ok := expr.CompareValues(df.index.At(pos[a]), df.index.At(pos[b]))
		if !ok {
			return false
		}
		if ascending {
			return c < 0
		}
		return c > 0
	})
	return df.Take(pos)
}

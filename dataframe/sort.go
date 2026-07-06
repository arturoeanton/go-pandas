package dataframe

import (
	"fmt"
	"sort"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
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
	keyCols := make([][]any, len(columns))
	for k, name := range columns {
		c, err := df.Col(name)
		if err != nil {
			return nil, err
		}
		keyCols[k] = c.Values()
	}
	pos := make([]int, df.Len())
	for i := range pos {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		for k := range columns {
			va, vb := keyCols[k][pos[a]], keyCols[k][pos[b]]
			naA, naB := dtype.IsNA(va), dtype.IsNA(vb)
			if naA || naB {
				if naA && naB {
					continue
				}
				return naB // NA always last
			}
			c, ok := expr.CompareValues(va, vb)
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

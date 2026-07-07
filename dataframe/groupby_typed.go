package dataframe

import (
	"sort"

	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	gby "github.com/arturoeanton/go-pandas/internal/groupby"
	"github.com/arturoeanton/go-pandas/series"
)

// groupPlan couples the typed group ids with the output ordering.
type groupPlan struct {
	plan  *gby.Plan
	order []int // group ids in output order
	rows  [][]int
}

// buildPlan runs the typed key builders once and computes the output
// group order (sorted by key labels with NA groups last, or first-seen).
func (gb *GroupBy) buildPlan() (*groupPlan, error) {
	if gb.err != nil {
		return nil, gb.err
	}
	keyCols := make([]column.Column, len(gb.keys))
	for k, name := range gb.keys {
		keyCols[k] = gb.df.MustCol(name).Storage()
	}
	plan := gby.Build(keyCols, gb.dropNA)
	order := make([]int, plan.Count)
	for i := range order {
		order[i] = i
	}
	if gb.sort {
		sort.SliceStable(order, func(x, y int) bool {
			ra, rb := plan.FirstRow[order[x]], plan.FirstRow[order[y]]
			for _, kc := range keyCols {
				aNA, bNA := kc.IsNA(ra), kc.IsNA(rb)
				if aNA || bNA {
					if aNA && bNA {
						continue
					}
					return bNA // NA groups sort last
				}
				c, ok := expr.CompareValues(kc.Value(ra), kc.Value(rb))
				if !ok || c == 0 {
					continue
				}
				return c < 0
			}
			return false
		})
	}
	return &groupPlan{plan: plan, order: order}, nil
}

// groupRows lazily expands per-group row lists (Apply and object
// fallbacks only).
func (gp *groupPlan) groupRows() [][]int {
	if gp.rows == nil {
		gp.rows = gp.plan.Rows()
	}
	return gp.rows
}

// keyLabelSeries gathers the typed key columns at each group's first row,
// in output order, sharing one RangeIndex.
func (gb *GroupBy) keyLabelSeries(gp *groupPlan, idx index.Index) ([]*series.Series, error) {
	firstRows := make([]int, len(gp.order))
	for i, g := range gp.order {
		firstRows[i] = gp.plan.FirstRow[g]
	}
	out := make([]*series.Series, len(gb.keys))
	for k, name := range gb.keys {
		taken, err := gb.df.MustCol(name).Storage().Take(firstRows)
		if err != nil {
			return nil, err
		}
		out[k] = series.Assemble(name, taken, idx)
	}
	return out, nil
}

// reorder helpers map per-group-id arrays into output order.

func reorderInts(vals []int, order []int) column.Column {
	out := make([]int, len(order))
	for i, g := range order {
		out[i] = vals[g]
	}
	return column.NewInt(out, nil)
}

func reorderFloats(vals []float64, na []bool, order []int) column.Column {
	out := make([]float64, len(order))
	var mask []bool
	for i, g := range order {
		out[i] = vals[g]
		if na != nil && na[g] {
			if mask == nil {
				mask = make([]bool, len(order))
			}
			mask[i] = true
		}
	}
	return column.NewFloat64(out, mask)
}

func reorderTake(c column.Column, indices []int, order []int) (column.Column, error) {
	pos := make([]int, len(order))
	for i, g := range order {
		pos[i] = indices[g] // -1 (no value) becomes NA through Take
	}
	return c.Take(pos)
}

// maskOf extracts a column's NA mask without boxing values.
func maskOf(c column.Column) []bool {
	out := make([]bool, c.Len())
	for i := range out {
		out[i] = c.IsNA(i)
	}
	return out
}

// aggregateColumn runs one aggregation over a value column using the
// typed segment reducers, falling back to the boxed per-group path for
// object-backed data.
func (gb *GroupBy) aggregateColumn(gp *groupPlan, s *series.Series, agg string) (column.Column, error) {
	c := s.Storage()
	ids, count, order := gp.plan.GroupIDs, gp.plan.Count, gp.order

	switch agg {
	case "size":
		return reorderInts(gby.Sizes(ids, count), order), nil
	case "count":
		return reorderInts(gby.CountNonNA(maskOf(c), ids, count), order), nil
	case "first":
		return reorderTake(c, gby.FirstIdx(maskOf(c), ids, count), order)
	case "last":
		return reorderTake(c, gby.LastIdx(maskOf(c), ids, count), order)
	}

	// dtype-specific kernels
	if vals, mask, ok := column.Strings(c); ok {
		switch agg {
		case "min":
			return reorderTake(c, gby.MinIdxString(vals, mask, ids, count), order)
		case "max":
			return reorderTake(c, gby.MaxIdxString(vals, mask, ids, count), order)
		case "nunique":
			return reorderInts(gby.NUniqueString(vals, mask, ids, count), order), nil
		}
		return gb.fallbackAgg(gp, s, agg)
	}
	if vals, mask, ok := column.Times(c); ok {
		switch agg {
		case "min":
			return reorderTake(c, gby.MinIdxTime(vals, mask, ids, count), order)
		case "max":
			return reorderTake(c, gby.MaxIdxTime(vals, mask, ids, count), order)
		case "nunique":
			return reorderInts(gby.NUniqueTime(vals, mask, ids, count), order), nil
		}
		return gb.fallbackAgg(gp, s, agg)
	}
	if !column.IsObjectBacked(c) {
		if vals, mask, ok := c.Float64s(); ok {
			switch agg {
			case "sum":
				return reorderFloats(gby.SumFloat(vals, mask, ids, count), nil, order), nil
			case "mean":
				m, na := gby.MeanFloat(vals, mask, ids, count)
				return reorderFloats(m, na, order), nil
			case "median":
				m, na := gby.MedianFloat(vals, mask, ids, count)
				return reorderFloats(m, na, order), nil
			case "var":
				v, na := gby.VarFloat(vals, mask, ids, count, false)
				return reorderFloats(v, na, order), nil
			case "std":
				v, na := gby.VarFloat(vals, mask, ids, count, true)
				return reorderFloats(v, na, order), nil
			case "min":
				return reorderTake(c, gby.MinIdxFloat(vals, mask, ids, count), order)
			case "max":
				return reorderTake(c, gby.MaxIdxFloat(vals, mask, ids, count), order)
			case "nunique":
				return reorderInts(gby.NUniqueFloat(vals, mask, ids, count), order), nil
			}
		}
	}
	return gb.fallbackAgg(gp, s, agg)
}

// fallbackAgg preserves the pre-v0.5 boxed behavior for object-backed
// columns (and any aggregation without a typed kernel): per-group Take
// plus the Series reduction, then dtype inference.
func (gb *GroupBy) fallbackAgg(gp *groupPlan, s *series.Series, agg string) (column.Column, error) {
	rows := gp.groupRows()
	values := make([]any, len(gp.order))
	for i, g := range gp.order {
		v, err := aggValue(s, rows[g], agg)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return column.Infer(values), nil
}

// assembleAgg builds the output frame: typed key label columns followed
// by one column per aggregation spec, all sharing one RangeIndex.
func (gb *GroupBy) assembleAgg(gp *groupPlan, specs []aggSpec) (*DataFrame, error) {
	idx := index.NewRangeIndex(len(gp.order))
	cols, err := gb.keyLabelSeries(gp, idx)
	if err != nil {
		return nil, err
	}
	for _, spec := range specs {
		s, err := gb.df.Col(spec.column)
		if err != nil {
			return nil, err
		}
		out, err := gb.aggregateColumn(gp, s, spec.agg)
		if err != nil {
			return nil, err
		}
		cols = append(cols, series.Assemble(spec.outName, out, idx))
	}
	return newFrame(cols, idx)
}

// sizeFrame builds the Size() result (key labels + "size" column).
func (gb *GroupBy) sizeFrame(gp *groupPlan) (*DataFrame, error) {
	idx := index.NewRangeIndex(len(gp.order))
	cols, err := gb.keyLabelSeries(gp, idx)
	if err != nil {
		return nil, err
	}
	sizes := reorderInts(gby.Sizes(gp.plan.GroupIDs, gp.plan.Count), gp.order)
	cols = append(cols, series.Assemble("size", sizes, idx))
	return newFrame(cols, idx)
}

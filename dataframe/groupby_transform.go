package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	gby "github.com/arturoeanton/go-pandas/internal/groupby"
	"github.com/arturoeanton/go-pandas/series"
)

// Transform broadcasts a group aggregate back to every row (v0.10),
// pandas' df.groupby(key)[column].transform(agg): the output series has
// the input length, row order and index; each row carries its group's
// aggregate. Rows whose key is NA (dropped groups) become NA. Any
// aggregation the typed reducers support works (mean, sum, count, min,
// max, first, last, nunique, ...).
func (gb *GroupBy) Transform(column, agg string) (*series.Series, error) {
	gp, err := gb.buildPlan()
	if err != nil {
		return nil, err
	}
	s, err := gb.df.Col(column)
	if err != nil {
		return nil, err
	}
	aggCol, err := gb.aggregateColumn(gp, s, agg)
	if err != nil {
		return nil, err
	}
	// aggCol is in output (sorted-group) order; rank maps group id ->
	// position, and the broadcast is one typed gather.
	rank := make([]int, gp.plan.Count)
	for r, g := range gp.order {
		rank[g] = r
	}
	positions := make([]int, len(gp.plan.GroupIDs))
	for i, g := range gp.plan.GroupIDs {
		if g < 0 {
			positions[i] = -1 // NA key -> NA output
			continue
		}
		positions[i] = rank[g]
	}
	out, err := aggCol.Take(positions)
	if err != nil {
		return nil, err
	}
	return series.Assemble(column, out, gb.df.Index().Clone()), nil
}

// GroupCond is a group-level filter condition built with GroupSize()
// or GroupCount(column).
type GroupCond struct {
	column string // "" = group size
	op     string
	value  float64
}

// GroupMetric builds group filter conditions:
// GroupSize().Gt(2), GroupCount("salary").Ge(3).
type GroupMetric struct{ column string }

// GroupSize filters on the number of rows per group.
func GroupSize() GroupMetric { return GroupMetric{} }

// GroupCount filters on the number of non-NA values of a column per
// group.
func GroupCount(column string) GroupMetric { return GroupMetric{column: column} }

func (m GroupMetric) cond(op string, v float64) GroupCond {
	return GroupCond{column: m.column, op: op, value: v}
}

// Gt, Ge, Lt, Le, Eq and Ne compare the group metric against a value.
func (m GroupMetric) Gt(v float64) GroupCond { return m.cond(">", v) }
func (m GroupMetric) Ge(v float64) GroupCond { return m.cond(">=", v) }
func (m GroupMetric) Lt(v float64) GroupCond { return m.cond("<", v) }
func (m GroupMetric) Le(v float64) GroupCond { return m.cond("<=", v) }
func (m GroupMetric) Eq(v float64) GroupCond { return m.cond("==", v) }
func (m GroupMetric) Ne(v float64) GroupCond { return m.cond("!=", v) }

func (c GroupCond) holds(metric float64) bool {
	switch c.op {
	case ">":
		return metric > c.value
	case ">=":
		return metric >= c.value
	case "<":
		return metric < c.value
	case "<=":
		return metric <= c.value
	case "==":
		return metric == c.value
	case "!=":
		return metric != c.value
	}
	return false
}

// Filter keeps whole groups that satisfy the condition (v0.10), pandas'
// df.groupby(key).filter(...): row order and index are preserved, rows
// whose key is NA (dropped groups) are removed, and the gather is
// typed.
func (gb *GroupBy) Filter(cond GroupCond) (*DataFrame, error) {
	if cond.op == "" {
		return nil, fmt.Errorf("%w: empty group filter condition (use GroupSize()/GroupCount(col) builders)", errs.ErrInvalidOperation)
	}
	gp, err := gb.buildPlan()
	if err != nil {
		return nil, err
	}
	var metrics []int
	if cond.column == "" {
		metrics = gby.Sizes(gp.plan.GroupIDs, gp.plan.Count)
	} else {
		s, err := gb.df.Col(cond.column)
		if err != nil {
			return nil, err
		}
		metrics = gby.CountNonNA(maskOf(s.Storage()), gp.plan.GroupIDs, gp.plan.Count)
	}
	keep := make([]bool, gp.plan.Count)
	for g, m := range metrics {
		keep[g] = cond.holds(float64(m))
	}
	var positions []int
	for i, g := range gp.plan.GroupIDs {
		if g >= 0 && keep[g] {
			positions = append(positions, i)
		}
	}
	return gb.df.Take(positions)
}

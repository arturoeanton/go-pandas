// Package groupby implements the typed GroupBy engine (v0.5): group ids
// are built once from typed key buffers (no per-row fmt/boxing) and
// aggregations run as segment reductions over those ids — no
// sub-DataFrame is ever materialized per group.
package groupby

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/internal/column"
)

// Plan is the result of grouping rows by one or more key columns.
type Plan struct {
	// GroupIDs[i] is the group of row i in FIRST-SEEN order;
	// -1 marks rows dropped because a key was NA and dropNA is set.
	GroupIDs []int
	// Count is the number of groups.
	Count int
	// FirstRow[g] is the first row belonging to group g — the
	// representative used to gather typed key label columns.
	FirstRow []int
}

// Rows expands the plan into per-group row lists (used by Apply and by
// object-column fallbacks; the typed reducers never need it).
func (p *Plan) Rows() [][]int {
	sizes := make([]int, p.Count)
	for _, g := range p.GroupIDs {
		if g >= 0 {
			sizes[g]++
		}
	}
	rows := make([][]int, p.Count)
	for g := range rows {
		rows[g] = make([]int, 0, sizes[g])
	}
	for i, g := range p.GroupIDs {
		if g >= 0 {
			rows[g] = append(rows[g], i)
		}
	}
	return rows
}

// Build groups rows by the given key columns. Each key column gets its
// own typed id pass; multiple keys combine pairwise through comparable
// [2]int composite keys (one map entry per distinct combination, no
// per-row allocation).
func Build(keys []column.Column, dropNA bool) *Plan {
	if len(keys) == 0 {
		return &Plan{}
	}
	plan := singleKey(keys[0], dropNA)
	for _, key := range keys[1:] {
		next := singleKey(key, dropNA)
		plan = combine(plan, next)
	}
	return plan
}

// singleKey assigns group ids for one key column using a typed map.
func singleKey(c column.Column, dropNA bool) *Plan {
	n := c.Len()
	ids := make([]int, n)
	var first []int
	register := func(row int) int {
		first = append(first, row)
		return len(first) - 1
	}
	naGroup := -1
	assignNA := func(i int) {
		if dropNA {
			ids[i] = -1
			return
		}
		if naGroup == -1 {
			naGroup = register(i)
		}
		ids[i] = naGroup
	}

	switch {
	case tryCategorical(c, ids, register, assignNA):
	case tryStrings(c, ids, register, assignNA):
	case tryBools(c, ids, register, assignNA):
	case tryTimes(c, ids, register, assignNA):
	case tryNumeric(c, ids, register, assignNA):
	default:
		objectKey(c, ids, register, assignNA)
	}
	return &Plan{GroupIDs: ids, Count: len(first), FirstRow: first}
}

// tryCategorical is the v0.7 code fast path: category codes are already
// dense ids, so a slot array replaces the hash map — one array index per
// row instead of a map lookup on the label.
func tryCategorical(c column.Column, ids []int, register func(int) int, assignNA func(int)) bool {
	cc, ok := column.AsCategorical(c)
	if !ok {
		return false
	}
	codes, mask := cc.RawCodes()
	slots := make([]int, cc.CategoryCount())
	for i := range slots {
		slots[i] = -1
	}
	for i, code := range codes {
		if mask[i] {
			assignNA(i)
			continue
		}
		if slots[code] == -1 {
			slots[code] = register(i)
		}
		ids[i] = slots[code]
	}
	return true
}

func tryStrings(c column.Column, ids []int, register func(int) int, assignNA func(int)) bool {
	vals, mask, ok := column.Strings(c)
	if !ok {
		return false
	}
	seen := make(map[string]int)
	for i, v := range vals {
		if mask[i] {
			assignNA(i)
			continue
		}
		g, found := seen[v]
		if !found {
			g = register(i)
			seen[v] = g
		}
		ids[i] = g
	}
	return true
}

func tryBools(c column.Column, ids []int, register func(int) int, assignNA func(int)) bool {
	vals, mask, ok := column.Bools(c)
	if !ok {
		return false
	}
	slot := [2]int{-1, -1}
	for i, v := range vals {
		if mask[i] {
			assignNA(i)
			continue
		}
		k := 0
		if v {
			k = 1
		}
		if slot[k] == -1 {
			slot[k] = register(i)
		}
		ids[i] = slot[k]
	}
	return true
}

func tryTimes(c column.Column, ids []int, register func(int) int, assignNA func(int)) bool {
	vals, mask, ok := column.Times(c)
	if !ok {
		return false
	}
	seen := make(map[time.Time]int)
	for i, v := range vals {
		if mask[i] {
			assignNA(i)
			continue
		}
		g, found := seen[v]
		if !found {
			g = register(i)
			seen[v] = g
		}
		ids[i] = g
	}
	return true
}

// tryNumeric unifies int/int64/float32/float64/bool-as-number keys
// through the Float64s buffer accessor: numeric widths group together
// (1, int64(1) and 1.0 share a group), matching the previous behavior.
func tryNumeric(c column.Column, ids []int, register func(int) int, assignNA func(int)) bool {
	if column.IsObjectBacked(c) {
		return false
	}
	vals, mask, ok := c.Float64s()
	if !ok {
		return false
	}
	seen := make(map[float64]int)
	for i, v := range vals {
		if mask[i] {
			assignNA(i)
			continue
		}
		g, found := seen[v]
		if !found {
			g = register(i)
			seen[v] = g
		}
		ids[i] = g
	}
	return true
}

// objectKey is the boxed fallback for object-backed key columns,
// preserving the historical `%v` grouping semantics.
func objectKey(c column.Column, ids []int, register func(int) int, assignNA func(int)) {
	seen := make(map[string]int)
	for i := 0; i < c.Len(); i++ {
		if c.IsNA(i) {
			assignNA(i)
			continue
		}
		k := fmt.Sprintf("%v", c.Value(i))
		g, found := seen[k]
		if !found {
			g = register(i)
			seen[k] = g
		}
		ids[i] = g
	}
}

// combine folds two per-key plans into a composite plan. The [2]int map
// key is comparable and allocation-free per lookup.
func combine(a, b *Plan) *Plan {
	n := len(a.GroupIDs)
	ids := make([]int, n)
	var first []int
	seen := make(map[[2]int]int)
	for i := 0; i < n; i++ {
		ga, gb := a.GroupIDs[i], b.GroupIDs[i]
		if ga == -1 || gb == -1 {
			ids[i] = -1
			continue
		}
		key := [2]int{ga, gb}
		g, found := seen[key]
		if !found {
			first = append(first, i)
			g = len(first) - 1
			seen[key] = g
		}
		ids[i] = g
	}
	return &Plan{GroupIDs: ids, Count: len(first), FirstRow: first}
}

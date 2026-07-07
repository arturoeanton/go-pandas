// Package join implements the typed hash-join engine (v0.6): left and
// right key columns are mapped into one shared id space with typed maps
// (no fmt in typed paths), the right side is indexed CSR-style and the
// probe emits exact-size row-pair vectors that materialize through typed
// gathers.
package join

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// PairIDs maps the left and right key tuples into one shared id space.
// ids are dense (0..count-1); -1 marks rows whose key contains NA — NA
// keys never match (a documented difference from pandas, which joins
// NaN keys to each other). Multi-key tuples compose pairwise through
// comparable [2]int map keys, allocation-free per row.
func PairIDs(left, right []column.Column) (lids, rids []int, count int) {
	lids, rids, count = pairIDsSingle(left[0], right[0])
	for k := 1; k < len(left); k++ {
		nl, nr, nc := pairIDsSingle(left[k], right[k])
		lids, rids, count = combine(lids, rids, nl, nr, count, nc)
	}
	return lids, rids, count
}

// register tracks the shared id space across both sides.
type register struct{ n int }

func (r *register) next() int {
	id := r.n
	r.n++
	return id
}

// pairIDsSingle picks the typed builder for one key pair. Both sides
// must expose the same kind; anything else (and object backings) routes
// through the %v fallback, which reproduces the historical cross-kind
// matching.
func pairIDsSingle(l, r column.Column) ([]int, []int, int) {
	reg := &register{}
	if ls, lm, ok := column.Strings(l); ok {
		if rs, rm, ok := column.Strings(r); ok {
			seen := make(map[string]int)
			lids := idsOver(len(ls), lm, func(i int) int { return lookup(seen, ls[i], reg) })
			rids := idsOver(len(rs), rm, func(i int) int { return lookup(seen, rs[i], reg) })
			return lids, rids, reg.n
		}
	}
	if lt, lm, ok := column.Times(l); ok {
		if rt, rm, ok := column.Times(r); ok {
			seen := make(map[time.Time]int)
			lids := idsOver(len(lt), lm, func(i int) int { return lookup(seen, lt[i], reg) })
			rids := idsOver(len(rt), rm, func(i int) int { return lookup(seen, rt[i], reg) })
			return lids, rids, reg.n
		}
	}
	if bothNumeric(l, r) {
		lf, lm, _ := l.Float64s()
		rf, rm, _ := r.Float64s()
		seen := make(map[float64]int)
		lids := idsOver(len(lf), lm, func(i int) int { return lookup(seen, lf[i], reg) })
		rids := idsOver(len(rf), rm, func(i int) int { return lookup(seen, rf[i], reg) })
		return lids, rids, reg.n
	}
	// object / mixed-kind fallback: historical %v matching (numerics
	// normalized through float64 so int 1 matches 1.0 across frames).
	seen := make(map[string]int)
	key := func(c column.Column, i int) string {
		v := c.Value(i)
		if f, ok := dtype.AsFloat(v); ok {
			if _, isBool := v.(bool); !isBool {
				return fmt.Sprintf("n\x00%v", f)
			}
		}
		return fmt.Sprintf("%T\x00%v", v, v)
	}
	lids := idsOver(l.Len(), nil, func(i int) int {
		if l.IsNA(i) {
			return -1
		}
		return lookup(seen, key(l, i), reg)
	})
	rids := idsOver(r.Len(), nil, func(i int) int {
		if r.IsNA(i) {
			return -1
		}
		return lookup(seen, key(r, i), reg)
	})
	return lids, rids, reg.n
}

func bothNumeric(l, r column.Column) bool {
	numeric := func(c column.Column) bool {
		if column.IsObjectBacked(c) {
			return false
		}
		return dtype.IsNumeric(c.DType()) || dtype.IsBool(c.DType())
	}
	return numeric(l) && numeric(r)
}

func lookup[K comparable](seen map[K]int, k K, reg *register) int {
	id, ok := seen[k]
	if !ok {
		id = reg.next()
		seen[k] = id
	}
	return id
}

// idsOver builds the id vector for one side; mask (when non-nil) marks
// NA rows as -1 before consulting the id function.
func idsOver(n int, mask []bool, id func(i int) int) []int {
	out := make([]int, n)
	for i := 0; i < n; i++ {
		if mask != nil && mask[i] {
			out[i] = -1
			continue
		}
		out[i] = id(i)
	}
	return out
}

// combine folds two key layers into composite ids over both sides.
func combine(la, ra, lb, rb []int, _, _ int) ([]int, []int, int) {
	seen := make(map[[2]int]int)
	reg := &register{}
	fold := func(a, b []int) []int {
		out := make([]int, len(a))
		for i := range a {
			if a[i] == -1 || b[i] == -1 {
				out[i] = -1
				continue
			}
			out[i] = lookup(seen, [2]int{a[i], b[i]}, reg)
		}
		return out
	}
	l := fold(la, lb)
	r := fold(ra, rb)
	return l, r, reg.n
}

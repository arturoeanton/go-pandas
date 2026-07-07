package join

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// How enumerates the join types of the typed engine.
type How int

const (
	Inner How = iota
	Left
	Right
	Outer
)

// MatchKind labels each output pair for the indicator column.
type MatchKind uint8

const (
	MatchBoth MatchKind = iota
	MatchLeftOnly
	MatchRightOnly
)

// Plan holds the row-pair vectors that materialize the join output:
// row i of the result combines left row LeftRows[i] with right row
// RightRows[i]; -1 marks the missing side.
type Plan struct {
	LeftRows  []int
	RightRows []int
	Match     []MatchKind
}

// csr indexes one side's rows by key id in compressed form: rows with
// id g live at rows[offsets[g]:offsets[g+1]]. Two allocations total.
type csr struct {
	offsets []int
	rows    []int
}

func buildCSR(ids []int, count int) csr {
	offsets := make([]int, count+1)
	for _, g := range ids {
		if g >= 0 {
			offsets[g+1]++
		}
	}
	for g := 0; g < count; g++ {
		offsets[g+1] += offsets[g]
	}
	rows := make([]int, offsets[count])
	cursor := append([]int(nil), offsets[:count]...)
	for i, g := range ids {
		if g >= 0 {
			rows[cursor[g]] = i
			cursor[g]++
		}
	}
	return csr{offsets: offsets, rows: rows}
}

func (c csr) group(g int) []int {
	if g < 0 {
		return nil
	}
	return c.rows[c.offsets[g]:c.offsets[g+1]]
}

// Build computes the pair vectors for a join type. Sizes are pre-counted
// so every vector allocates exactly once; pair order is deterministic:
// probe-side order, matches in build-side row order, then (outer)
// unmatched build-side rows in their own order.
func Build(how How, lids, rids []int, count int) *Plan {
	switch how {
	case Right:
		// A right join is a left join probed from the right.
		leftIdx := buildCSR(lids, count)
		return probe(rids, leftIdx, true, true)
	case Outer:
		rightIdx := buildCSR(rids, count)
		plan := probe(lids, rightIdx, true, false)
		appendUnmatchedRight(plan, rids, rightIdx, lids)
		return plan
	case Left:
		rightIdx := buildCSR(rids, count)
		return probe(lids, rightIdx, true, false)
	default: // Inner
		rightIdx := buildCSR(rids, count)
		return probe(lids, rightIdx, false, false)
	}
}

// probe walks the probe side in order, pairing with build-side matches.
// keepUnmatched emits probe rows without matches (left/right/outer
// joins); swapped reports that the probe side is the RIGHT frame.
func probe(probeIDs []int, buildIdx csr, keepUnmatched, swapped bool) *Plan {
	// pass 1: exact output size
	total := 0
	for _, id := range probeIDs {
		m := len(buildIdx.group(id))
		if m == 0 {
			if keepUnmatched {
				total++
			}
			continue
		}
		total += m
	}
	plan := &Plan{
		LeftRows:  make([]int, 0, total),
		RightRows: make([]int, 0, total),
		Match:     make([]MatchKind, 0, total),
	}
	for i, id := range probeIDs {
		matches := buildIdx.group(id)
		if len(matches) == 0 {
			if !keepUnmatched {
				continue
			}
			if swapped {
				plan.LeftRows = append(plan.LeftRows, -1)
				plan.RightRows = append(plan.RightRows, i)
				plan.Match = append(plan.Match, MatchRightOnly)
			} else {
				plan.LeftRows = append(plan.LeftRows, i)
				plan.RightRows = append(plan.RightRows, -1)
				plan.Match = append(plan.Match, MatchLeftOnly)
			}
			continue
		}
		for _, m := range matches {
			if swapped {
				plan.LeftRows = append(plan.LeftRows, m)
				plan.RightRows = append(plan.RightRows, i)
			} else {
				plan.LeftRows = append(plan.LeftRows, i)
				plan.RightRows = append(plan.RightRows, m)
			}
			plan.Match = append(plan.Match, MatchBoth)
		}
	}
	return plan
}

// appendUnmatchedRight adds right rows whose key never matched a left
// row (outer join tail), in right-row order.
func appendUnmatchedRight(plan *Plan, rids []int, rightIdx csr, lids []int) {
	matched := make([]bool, len(rids))
	for _, id := range lids {
		for _, r := range rightIdx.group(id) {
			matched[r] = true
		}
	}
	for i := range rids {
		if !matched[i] { // includes NA-key right rows, which never match
			plan.LeftRows = append(plan.LeftRows, -1)
			plan.RightRows = append(plan.RightRows, i)
			plan.Match = append(plan.Match, MatchRightOnly)
		}
	}
}

// Cross builds the cartesian product plan.
func Cross(nLeft, nRight int) *Plan {
	total := nLeft * nRight
	plan := &Plan{
		LeftRows:  make([]int, 0, total),
		RightRows: make([]int, 0, total),
		Match:     make([]MatchKind, 0, total),
	}
	for i := 0; i < nLeft; i++ {
		for j := 0; j < nRight; j++ {
			plan.LeftRows = append(plan.LeftRows, i)
			plan.RightRows = append(plan.RightRows, j)
			plan.Match = append(plan.Match, MatchBoth)
		}
	}
	return plan
}

// Validate enforces merge cardinality constraints using id counts (no
// boxing). NA keys are excluded, matching the historical behavior.
func Validate(rule string, lids, rids []int, count int) error {
	if rule == "" || rule == "many_to_many" {
		return nil
	}
	unique := func(ids []int) bool {
		counts := make([]int, count)
		for _, id := range ids {
			if id < 0 {
				continue
			}
			counts[id]++
			if counts[id] > 1 {
				return false
			}
		}
		return true
	}
	switch rule {
	case "one_to_one":
		if !unique(lids) || !unique(rids) {
			return fmt.Errorf("%w: merge keys are not one_to_one", errs.ErrInvalidJoin)
		}
	case "one_to_many":
		if !unique(lids) {
			return fmt.Errorf("%w: left merge keys are not unique for one_to_many", errs.ErrInvalidJoin)
		}
	case "many_to_one":
		if !unique(rids) {
			return fmt.Errorf("%w: right merge keys are not unique for many_to_one", errs.ErrInvalidJoin)
		}
	default:
		return fmt.Errorf("%w: validate=%q", errs.ErrInvalidJoin, rule)
	}
	return nil
}

package index

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// resolveLevel maps a level selector (int position or string name) to
// its position.
func (ix *MultiIndex) resolveLevel(level any) (int, error) {
	switch v := level.(type) {
	case int:
		if v < 0 {
			v += ix.NLevels() // pandas-style negative positions
		}
		if v < 0 || v >= ix.NLevels() {
			return -1, fmt.Errorf("%w: level %d out of range for %d levels", errs.ErrInvalidIndex, level, ix.NLevels())
		}
		return v, nil
	case string:
		for l, name := range ix.names {
			if name == v {
				return l, nil
			}
		}
		return -1, fmt.Errorf("%w: level %q not found in %v", errs.ErrInvalidIndex, v, ix.names)
	}
	return -1, fmt.Errorf("%w: level selector must be an int position or string name, got %T", errs.ErrInvalidIndex, level)
}

// selectLevels assembles a new index keeping the given level positions
// in order: a MultiIndex for 2+, the flat labels for exactly one.
func (ix *MultiIndex) selectLevels(keep []int) (Index, error) {
	if len(keep) == 0 {
		return nil, fmt.Errorf("%w: cannot drop every level", errs.ErrInvalidIndex)
	}
	if len(keep) == 1 {
		l := keep[0]
		labels := make([]any, ix.length)
		for i := 0; i < ix.length; i++ {
			if c := ix.codes[l][i]; c >= 0 {
				labels[i] = ix.levels[l][c]
			}
		}
		return FromLabels(labels, ix.names[l]), nil
	}
	levels := make([][]any, len(keep))
	codes := make([][]int32, len(keep))
	names := make([]string, len(keep))
	for out, l := range keep {
		levels[out] = append([]any(nil), ix.levels[l]...)
		codes[out] = append([]int32(nil), ix.codes[l]...)
		names[out] = ix.names[l]
	}
	return NewMultiIndexFromCodes(levels, codes, names)
}

// DropLevel removes one level by name or position (v1.0-rc): the
// result is a MultiIndex when 2+ levels remain, otherwise a flat index
// of the remaining level's labels — pandas' droplevel. Duplicate
// resulting labels are allowed, like pandas.
func (ix *MultiIndex) DropLevel(level any) (Index, error) {
	drop, err := ix.resolveLevel(level)
	if err != nil {
		return nil, err
	}
	if ix.NLevels() < 2 {
		return nil, fmt.Errorf("%w: cannot drop the only level", errs.ErrInvalidIndex)
	}
	keep := make([]int, 0, ix.NLevels()-1)
	for l := 0; l < ix.NLevels(); l++ {
		if l != drop {
			keep = append(keep, l)
		}
	}
	return ix.selectLevels(keep)
}

// SwapLevel exchanges two levels by name or position (v1.0-rc),
// pandas' swaplevel. Without arguments it swaps the last two levels.
func (ix *MultiIndex) SwapLevel(levels ...any) (*MultiIndex, error) {
	a, b := any(ix.NLevels()-2), any(ix.NLevels()-1)
	switch len(levels) {
	case 0:
	case 2:
		a, b = levels[0], levels[1]
	default:
		return nil, fmt.Errorf("%w: SwapLevel takes zero or two level selectors", errs.ErrInvalidOperation)
	}
	la, err := ix.resolveLevel(a)
	if err != nil {
		return nil, err
	}
	lb, err := ix.resolveLevel(b)
	if err != nil {
		return nil, err
	}
	order := make([]int, ix.NLevels())
	for l := range order {
		order[l] = l
	}
	order[la], order[lb] = order[lb], order[la]
	out, err := ix.selectLevels(order)
	if err != nil {
		return nil, err
	}
	return out.(*MultiIndex), nil
}

// ReorderLevels rearranges every level by name or position (v1.0-rc),
// pandas' reorder_levels. The order must mention each level exactly
// once.
func (ix *MultiIndex) ReorderLevels(order ...any) (*MultiIndex, error) {
	if len(order) != ix.NLevels() {
		return nil, fmt.Errorf("%w: ReorderLevels needs all %d levels, got %d", errs.ErrInvalidOperation, ix.NLevels(), len(order))
	}
	keep := make([]int, len(order))
	seen := make(map[int]bool, len(order))
	for i, sel := range order {
		l, err := ix.resolveLevel(sel)
		if err != nil {
			return nil, err
		}
		if seen[l] {
			return nil, fmt.Errorf("%w: level %v repeated in ReorderLevels", errs.ErrInvalidOperation, sel)
		}
		seen[l] = true
		keep[i] = l
	}
	out, err := ix.selectLevels(keep)
	if err != nil {
		return nil, err
	}
	mi, ok := out.(*MultiIndex)
	if !ok {
		return nil, fmt.Errorf("%w: ReorderLevels needs 2+ levels", errs.ErrInvalidIndex)
	}
	return mi, nil
}

// PositionsLevel returns every row whose component at the given level
// matches the label (engine use: DataFrame.XS).
func (ix *MultiIndex) PositionsLevel(level int, label any) []int {
	code := ix.codeOf(level, label)
	if code < 0 {
		return nil
	}
	var out []int
	for i, c := range ix.codes[level] {
		if c == code {
			out = append(out, i)
		}
	}
	return out
}

// LevelPositions resolves a level selector (name or position) and
// returns the matching row positions plus the resolved level.
func (ix *MultiIndex) LevelPositions(level any, label any) ([]int, int, error) {
	l, err := ix.resolveLevel(level)
	if err != nil {
		return nil, -1, err
	}
	return ix.PositionsLevel(l, label), l, nil
}

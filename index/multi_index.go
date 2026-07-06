package index

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-pandas/errs"
)

// MultiIndex is a hierarchical index. v0.1 supports construction from
// arrays, positional access and display; label-based operations return
// ErrNotImplemented.
type MultiIndex struct {
	levels [][]any
	codes  [][]int
	names  []string
}

// NewMultiIndexFromArrays builds a MultiIndex from parallel label arrays,
// one per level.
func NewMultiIndexFromArrays(arrays [][]any, names []string) (*MultiIndex, error) {
	if len(arrays) == 0 {
		return nil, fmt.Errorf("%w: MultiIndex needs at least one array", errs.ErrInvalidIndex)
	}
	n := len(arrays[0])
	for _, arr := range arrays {
		if len(arr) != n {
			return nil, fmt.Errorf("%w: all MultiIndex arrays must have the same length", errs.ErrLengthMismatch)
		}
	}
	if names == nil {
		names = make([]string, len(arrays))
	}
	if len(names) != len(arrays) {
		return nil, fmt.Errorf("%w: names must match number of levels", errs.ErrLengthMismatch)
	}
	mi := &MultiIndex{names: append([]string(nil), names...)}
	for _, arr := range arrays {
		var level []any
		seen := map[any]int{}
		codes := make([]int, n)
		for i, v := range arr {
			code, ok := seen[v]
			if !ok {
				code = len(level)
				seen[v] = code
				level = append(level, v)
			}
			codes[i] = code
		}
		mi.levels = append(mi.levels, level)
		mi.codes = append(mi.codes, codes)
	}
	return mi, nil
}

// NLevels returns the number of levels of the MultiIndex.
func (ix *MultiIndex) NLevels() int { return len(ix.levels) }

func (ix *MultiIndex) Name() string { return strings.Join(ix.names, ", ") }

func (ix *MultiIndex) Len() int {
	if len(ix.codes) == 0 {
		return 0
	}
	return len(ix.codes[0])
}

// At returns the label tuple at a position as a []any.
func (ix *MultiIndex) At(pos int) any {
	tuple := make([]any, len(ix.levels))
	for lv := range ix.levels {
		tuple[lv] = ix.levels[lv][ix.codes[lv][pos]]
	}
	return tuple
}

func (ix *MultiIndex) Values() []any {
	out := make([]any, ix.Len())
	for i := range out {
		out[i] = ix.At(i)
	}
	return out
}

func (ix *MultiIndex) Pos(label any) (int, bool) { return -1, false }

func (ix *MultiIndex) Positions(label any) []int { return nil }

func (ix *MultiIndex) Slice(start, stop any) ([]int, error) {
	return nil, errs.NotImplemented("MultiIndex.Slice")
}

func (ix *MultiIndex) Equals(other Index) bool {
	o, ok := other.(*MultiIndex)
	if !ok || ix.Len() != o.Len() || len(ix.levels) != len(o.levels) {
		return false
	}
	for i := 0; i < ix.Len(); i++ {
		a := ix.At(i).([]any)
		b := o.At(i).([]any)
		for j := range a {
			if a[j] != b[j] {
				return false
			}
		}
	}
	return true
}

func (ix *MultiIndex) Clone() Index {
	c := &MultiIndex{names: append([]string(nil), ix.names...)}
	for _, lv := range ix.levels {
		c.levels = append(c.levels, append([]any(nil), lv...))
	}
	for _, cd := range ix.codes {
		c.codes = append(c.codes, append([]int(nil), cd...))
	}
	return c
}

func (ix *MultiIndex) String() string {
	var b strings.Builder
	b.WriteString("MultiIndex([")
	for i := 0; i < ix.Len(); i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%v", ix.At(i)))
	}
	b.WriteString(fmt.Sprintf("], names=%v)", ix.names))
	return b.String()
}

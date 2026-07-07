package index

import (
	"fmt"
	"sync"
)

// StringIndex is a label index backed by strings.
type StringIndex struct {
	values []string
	name   string
	// lookup caches label -> positions for O(1) Pos; built lazily so
	// gather-heavy paths (filtering) never pay for it (v0.4.1).
	lookupOnce sync.Once
	lookup     map[string][]int
}

// NewStringIndex builds a StringIndex, optionally named.
func NewStringIndex(values []string, name ...string) Index {
	n := ""
	if len(name) > 0 {
		n = name[0]
	}
	return &StringIndex{values: append([]string(nil), values...), name: n}
}

func (ix *StringIndex) buildLookup() {
	ix.lookupOnce.Do(func() {
		ix.lookup = make(map[string][]int, len(ix.values))
		for i, v := range ix.values {
			ix.lookup[v] = append(ix.lookup[v], i)
		}
	})
}

func (ix *StringIndex) Name() string   { return ix.name }
func (ix *StringIndex) Len() int       { return len(ix.values) }
func (ix *StringIndex) At(pos int) any { return ix.values[pos] }

func (ix *StringIndex) Values() []any {
	out := make([]any, len(ix.values))
	for i, v := range ix.values {
		out[i] = v
	}
	return out
}

func (ix *StringIndex) Pos(label any) (int, bool) {
	s, ok := label.(string)
	if !ok {
		return -1, false
	}
	ix.buildLookup()
	if ps := ix.lookup[s]; len(ps) > 0 {
		return ps[0], true
	}
	return -1, false
}

func (ix *StringIndex) Positions(label any) []int {
	s, ok := label.(string)
	if !ok {
		return nil
	}
	ix.buildLookup()
	return append([]int(nil), ix.lookup[s]...)
}

func (ix *StringIndex) Slice(start, stop any) ([]int, error) {
	return labelSlice(ix, start, stop)
}

func (ix *StringIndex) Equals(other Index) bool { return valuesEqual(ix, other) }

func (ix *StringIndex) Clone() Index {
	return NewStringIndex(ix.values, ix.name)
}

func (ix *StringIndex) String() string {
	return fmt.Sprintf("StringIndex(%v, name=%q)", ix.values, ix.name)
}

package index

import (
	"fmt"
	"sync"

	"github.com/arturoeanton/go-pandas/errs"
)

// Int64Index is an integer label index backed by []int64 — the typed
// result of gathering a RangeIndex by arbitrary positions (v0.4.1).
// Labels box as int in At()/Values() so they compare and render exactly
// like RangeIndex labels.
type Int64Index struct {
	values []int64
	name   string

	lookupOnce sync.Once
	lookup     map[int64][]int
}

// NewInt64Index builds an integer label index, optionally named.
func NewInt64Index(values []int64, name ...string) *Int64Index {
	n := ""
	if len(name) > 0 {
		n = name[0]
	}
	return &Int64Index{values: values, name: n}
}

func (ix *Int64Index) buildLookup() {
	ix.lookupOnce.Do(func() {
		ix.lookup = make(map[int64][]int, len(ix.values))
		for i, v := range ix.values {
			ix.lookup[v] = append(ix.lookup[v], i)
		}
	})
}

func (ix *Int64Index) Name() string   { return ix.name }
func (ix *Int64Index) Len() int       { return len(ix.values) }
func (ix *Int64Index) At(pos int) any { return int(ix.values[pos]) }

func (ix *Int64Index) Values() []any {
	out := make([]any, len(ix.values))
	for i, v := range ix.values {
		out[i] = int(v)
	}
	return out
}

func labelToInt64(label any) (int64, bool) {
	switch v := label.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	}
	return 0, false
}

func (ix *Int64Index) Pos(label any) (int, bool) {
	v, ok := labelToInt64(label)
	if !ok {
		return -1, false
	}
	ix.buildLookup()
	if ps := ix.lookup[v]; len(ps) > 0 {
		return ps[0], true
	}
	return -1, false
}

func (ix *Int64Index) Positions(label any) []int {
	v, ok := labelToInt64(label)
	if !ok {
		return nil
	}
	ix.buildLookup()
	return append([]int(nil), ix.lookup[v]...)
}

// Slice selects the inclusive label range [start, stop] by first
// occurrence, like the other label indexes.
func (ix *Int64Index) Slice(start, stop any) ([]int, error) {
	from := 0
	to := ix.Len() - 1
	if start != nil {
		p, ok := ix.Pos(start)
		if !ok {
			return nil, fmt.Errorf("%w: label %v not in Int64Index", errs.ErrInvalidIndex, start)
		}
		from = p
	}
	if stop != nil {
		p, ok := ix.Pos(stop)
		if !ok {
			return nil, fmt.Errorf("%w: label %v not in Int64Index", errs.ErrInvalidIndex, stop)
		}
		to = p
	}
	if from > to {
		return []int{}, nil
	}
	out := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out, nil
}

func (ix *Int64Index) Equals(other Index) bool { return valuesEqual(ix, other) }

func (ix *Int64Index) Clone() Index {
	return NewInt64Index(append([]int64(nil), ix.values...), ix.name)
}

func (ix *Int64Index) String() string {
	return fmt.Sprintf("Int64Index(%d values, name=%q)", len(ix.values), ix.name)
}

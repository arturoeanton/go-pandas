package index

import (
	"fmt"
	"time"
)

// DatetimeIndex is a label index backed by time.Time values. Partial in
// v0.1: lookup, slicing and display; no frequency/resampling semantics.
type DatetimeIndex struct {
	values []time.Time
	name   string
}

// NewDatetimeIndex builds a DatetimeIndex, optionally named.
func NewDatetimeIndex(values []time.Time, name ...string) Index {
	n := ""
	if len(name) > 0 {
		n = name[0]
	}
	return &DatetimeIndex{values: append([]time.Time(nil), values...), name: n}
}

func (ix *DatetimeIndex) Name() string   { return ix.name }
func (ix *DatetimeIndex) Len() int       { return len(ix.values) }
func (ix *DatetimeIndex) At(pos int) any { return ix.values[pos] }

func (ix *DatetimeIndex) Values() []any {
	out := make([]any, len(ix.values))
	for i, v := range ix.values {
		out[i] = v
	}
	return out
}

func (ix *DatetimeIndex) Pos(label any) (int, bool) {
	t, ok := label.(time.Time)
	if !ok {
		return -1, false
	}
	for i, v := range ix.values {
		if v.Equal(t) {
			return i, true
		}
	}
	return -1, false
}

func (ix *DatetimeIndex) Positions(label any) []int {
	t, ok := label.(time.Time)
	if !ok {
		return nil
	}
	var out []int
	for i, v := range ix.values {
		if v.Equal(t) {
			out = append(out, i)
		}
	}
	return out
}

// Slice selects positions whose timestamp lies between start and stop,
// inclusive on both ends (pandas .loc datetime slicing).
func (ix *DatetimeIndex) Slice(start, stop any) ([]int, error) {
	var from, to *time.Time
	if start != nil {
		t, ok := start.(time.Time)
		if !ok {
			return nil, fmt.Errorf("datetime slice start must be time.Time, got %T", start)
		}
		from = &t
	}
	if stop != nil {
		t, ok := stop.(time.Time)
		if !ok {
			return nil, fmt.Errorf("datetime slice stop must be time.Time, got %T", stop)
		}
		to = &t
	}
	var out []int
	for i, v := range ix.values {
		if from != nil && v.Before(*from) {
			continue
		}
		if to != nil && v.After(*to) {
			continue
		}
		out = append(out, i)
	}
	return out, nil
}

func (ix *DatetimeIndex) Equals(other Index) bool {
	o, ok := other.(*DatetimeIndex)
	if !ok || ix.Len() != o.Len() {
		return false
	}
	for i, v := range ix.values {
		if !v.Equal(o.values[i]) {
			return false
		}
	}
	return true
}

func (ix *DatetimeIndex) Clone() Index {
	return NewDatetimeIndex(ix.values, ix.name)
}

func (ix *DatetimeIndex) String() string {
	return fmt.Sprintf("DatetimeIndex(%d values, name=%q)", len(ix.values), ix.name)
}

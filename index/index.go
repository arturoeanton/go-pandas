// Package index implements pandas-style axis labels: RangeIndex,
// StringIndex, DatetimeIndex and a partial MultiIndex, plus alignment and
// set operations.
package index

import (
	"fmt"
	"time"
)

// Index is the interface implemented by every axis label container.
type Index interface {
	Name() string
	Len() int
	At(pos int) any
	Values() []any
	// Pos returns the first position of a label.
	Pos(label any) (int, bool)
	// Positions returns every position of a label (duplicate labels are
	// allowed, as in pandas).
	Positions(label any) []int
	// Slice returns the positions selected by a label slice. Like
	// pandas .loc slicing, both endpoints are inclusive.
	Slice(start, stop any) ([]int, error)
	Equals(other Index) bool
	Clone() Index
	String() string
}

// Take builds a new index from a list of positions. A negative position
// yields a missing label (used by outer joins).
func Take(idx Index, positions []int) Index {
	values := make([]any, len(positions))
	for i, p := range positions {
		if p < 0 {
			values[i] = nil
			continue
		}
		values[i] = idx.At(p)
	}
	return fromValues(values, idx.Name())
}

// fromValues rebuilds the most specific index type for a list of labels.
func fromValues(values []any, name string) Index {
	allString, allTime := true, true
	for _, v := range values {
		if _, ok := v.(string); !ok {
			allString = false
		}
		if _, ok := v.(time.Time); !ok {
			allTime = false
		}
	}
	switch {
	case allString && len(values) > 0:
		strs := make([]string, len(values))
		for i, v := range values {
			strs[i] = v.(string)
		}
		return NewStringIndex(strs, name)
	case allTime && len(values) > 0:
		ts := make([]time.Time, len(values))
		for i, v := range values {
			ts[i] = v.(time.Time)
		}
		return NewDatetimeIndex(ts, name)
	default:
		return &anyIndex{values: values, name: name}
	}
}

// anyIndex is a generic label index used when labels are heterogeneous
// (e.g. after Take on an outer join with missing labels).
type anyIndex struct {
	values []any
	name   string
}

func (ix *anyIndex) Name() string   { return ix.name }
func (ix *anyIndex) Len() int       { return len(ix.values) }
func (ix *anyIndex) At(pos int) any { return ix.values[pos] }
func (ix *anyIndex) Values() []any  { return append([]any(nil), ix.values...) }

func (ix *anyIndex) Pos(label any) (int, bool) {
	for i, v := range ix.values {
		if v == label {
			return i, true
		}
	}
	return -1, false
}

func (ix *anyIndex) Positions(label any) []int {
	var out []int
	for i, v := range ix.values {
		if v == label {
			out = append(out, i)
		}
	}
	return out
}

func (ix *anyIndex) Slice(start, stop any) ([]int, error) {
	return labelSlice(ix, start, stop)
}

func (ix *anyIndex) Equals(other Index) bool { return valuesEqual(ix, other) }

func (ix *anyIndex) Clone() Index {
	return &anyIndex{values: append([]any(nil), ix.values...), name: ix.name}
}

func (ix *anyIndex) String() string {
	return fmt.Sprintf("Index(%v, name=%q)", ix.values, ix.name)
}

// labelSlice implements inclusive label slicing shared by label indexes.
func labelSlice(ix Index, start, stop any) ([]int, error) {
	from := 0
	to := ix.Len() - 1
	if start != nil {
		p, ok := ix.Pos(start)
		if !ok {
			return nil, fmt.Errorf("label %v not found in index", start)
		}
		from = p
	}
	if stop != nil {
		p, ok := ix.Pos(stop)
		if !ok {
			return nil, fmt.Errorf("label %v not found in index", stop)
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

func valuesEqual(a, b Index) bool {
	if b == nil || a.Len() != b.Len() {
		return false
	}
	for i := 0; i < a.Len(); i++ {
		if a.At(i) != b.At(i) {
			return false
		}
	}
	return true
}

package index

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// RangeIndex is the default positional index (0, 1, 2, ...), like the
// pandas RangeIndex.
type RangeIndex struct {
	Start int
	Stop  int
	Step  int
	name  string
}

// NewRangeIndex builds a RangeIndex over [0, n).
func NewRangeIndex(n int) Index {
	return &RangeIndex{Start: 0, Stop: n, Step: 1}
}

// RangeIndexFrom builds a RangeIndex over [start, stop) with the given
// step. A zero step defaults to 1.
func RangeIndexFrom(start, stop, step int) Index {
	if step == 0 {
		step = 1
	}
	return &RangeIndex{Start: start, Stop: stop, Step: step}
}

func (ix *RangeIndex) Name() string { return ix.name }

func (ix *RangeIndex) Len() int {
	if ix.Step > 0 {
		if ix.Stop <= ix.Start {
			return 0
		}
		return (ix.Stop - ix.Start + ix.Step - 1) / ix.Step
	}
	if ix.Stop >= ix.Start {
		return 0
	}
	return (ix.Start - ix.Stop - ix.Step - 1) / -ix.Step
}

func (ix *RangeIndex) At(pos int) any { return ix.Start + pos*ix.Step }

func (ix *RangeIndex) Values() []any {
	out := make([]any, ix.Len())
	for i := range out {
		out[i] = ix.At(i)
	}
	return out
}

func (ix *RangeIndex) Pos(label any) (int, bool) {
	i, ok := toInt(label)
	if !ok {
		return -1, false
	}
	d := i - ix.Start
	if ix.Step == 0 || d%ix.Step != 0 {
		return -1, false
	}
	p := d / ix.Step
	if p < 0 || p >= ix.Len() {
		return -1, false
	}
	return p, true
}

func (ix *RangeIndex) Positions(label any) []int {
	if p, ok := ix.Pos(label); ok {
		return []int{p}
	}
	return nil
}

func (ix *RangeIndex) Slice(start, stop any) ([]int, error) {
	from := 0
	to := ix.Len() - 1
	if start != nil {
		p, ok := ix.Pos(start)
		if !ok {
			return nil, fmt.Errorf("%w: label %v not in RangeIndex", errs.ErrInvalidIndex, start)
		}
		from = p
	}
	if stop != nil {
		p, ok := ix.Pos(stop)
		if !ok {
			return nil, fmt.Errorf("%w: label %v not in RangeIndex", errs.ErrInvalidIndex, stop)
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

func (ix *RangeIndex) Equals(other Index) bool {
	if o, ok := other.(*RangeIndex); ok {
		return ix.Start == o.Start && ix.Len() == o.Len() && (ix.Len() == 0 || ix.Step == o.Step)
	}
	return valuesEqual(ix, other)
}

func (ix *RangeIndex) Clone() Index {
	c := *ix
	return &c
}

func (ix *RangeIndex) String() string {
	return fmt.Sprintf("RangeIndex(start=%d, stop=%d, step=%d)", ix.Start, ix.Stop, ix.Step)
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case int32:
		return int(x), true
	}
	return 0, false
}

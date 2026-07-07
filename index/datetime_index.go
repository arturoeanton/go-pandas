package index

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
)

// DatetimeIndex is a label index backed by time.Time values. Since v0.9
// it carries an NA mask (NaT labels), typed Take/SlicePos, monotonicity
// and range helpers, and label lookup by time.Time or by parseable
// string.
type DatetimeIndex struct {
	values []time.Time
	mask   []bool // nil = no missing labels
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

// NewDatetimeIndexWithMask builds a DatetimeIndex with missing labels
// (mask[i] = true marks NaT). A nil or all-false mask means none.
func NewDatetimeIndexWithMask(values []time.Time, mask []bool, name string) Index {
	ix := &DatetimeIndex{values: append([]time.Time(nil), values...), name: name}
	for _, m := range mask {
		if m {
			ix.mask = append([]bool(nil), mask...)
			break
		}
	}
	return ix
}

func (ix *DatetimeIndex) Name() string { return ix.name }
func (ix *DatetimeIndex) Len() int     { return len(ix.values) }

func (ix *DatetimeIndex) isNA(pos int) bool { return ix.mask != nil && ix.mask[pos] }

func (ix *DatetimeIndex) At(pos int) any {
	if ix.isNA(pos) {
		return nil
	}
	return ix.values[pos]
}

func (ix *DatetimeIndex) Values() []any {
	out := make([]any, len(ix.values))
	for i, v := range ix.values {
		if ix.isNA(i) {
			continue
		}
		out[i] = v
	}
	return out
}

// RawTimes exposes the backing buffers for engine use (resample). The
// mask is nil when the index has no missing labels; both are read-only.
func (ix *DatetimeIndex) RawTimes() ([]time.Time, []bool) { return ix.values, ix.mask }

// Start returns the earliest non-NA timestamp (zero time when empty).
func (ix *DatetimeIndex) Start() time.Time {
	var start time.Time
	seen := false
	for i, v := range ix.values {
		if ix.isNA(i) {
			continue
		}
		if !seen || v.Before(start) {
			start, seen = v, true
		}
	}
	return start
}

// End returns the latest non-NA timestamp (zero time when empty).
func (ix *DatetimeIndex) End() time.Time {
	var end time.Time
	seen := false
	for i, v := range ix.values {
		if ix.isNA(i) {
			continue
		}
		if !seen || v.After(end) {
			end, seen = v, true
		}
	}
	return end
}

// IsMonotonicIncreasing reports whether labels never decrease. Like
// pandas, any missing label makes the index non-monotonic.
func (ix *DatetimeIndex) IsMonotonicIncreasing() bool {
	if ix.mask != nil {
		for _, m := range ix.mask {
			if m {
				return false
			}
		}
	}
	for i := 1; i < len(ix.values); i++ {
		if ix.values[i].Before(ix.values[i-1]) {
			return false
		}
	}
	return true
}

// lookupLabel widens accepted label forms: time.Time, or a string
// parseable by the deterministic ToDatetime inference list (v0.9).
func lookupLabel(label any) (time.Time, bool) {
	switch v := label.(type) {
	case time.Time:
		return v, true
	case string:
		for _, layout := range dtype.InferTimeLayouts {
			if t, err := time.Parse(layout, v); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func (ix *DatetimeIndex) Pos(label any) (int, bool) {
	t, ok := lookupLabel(label)
	if !ok {
		return -1, false
	}
	for i, v := range ix.values {
		if !ix.isNA(i) && v.Equal(t) {
			return i, true
		}
	}
	return -1, false
}

func (ix *DatetimeIndex) Positions(label any) []int {
	t, ok := lookupLabel(label)
	if !ok {
		return nil
	}
	var out []int
	for i, v := range ix.values {
		if !ix.isNA(i) && v.Equal(t) {
			out = append(out, i)
		}
	}
	return out
}

// Slice selects positions whose timestamp lies between start and stop,
// inclusive on both ends (pandas .loc datetime slicing). Endpoints may
// be time.Time or parseable strings; NA labels never match.
func (ix *DatetimeIndex) Slice(start, stop any) ([]int, error) {
	var from, to *time.Time
	if start != nil {
		t, ok := lookupLabel(start)
		if !ok {
			return nil, fmt.Errorf("datetime slice start must be time.Time or parseable string, got %v (%T)", start, start)
		}
		from = &t
	}
	if stop != nil {
		t, ok := lookupLabel(stop)
		if !ok {
			return nil, fmt.Errorf("datetime slice stop must be time.Time or parseable string, got %v (%T)", stop, stop)
		}
		to = &t
	}
	var out []int
	for i, v := range ix.values {
		if ix.isNA(i) {
			continue
		}
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

// Take gathers labels by position; negative positions become NA labels.
func (ix *DatetimeIndex) Take(positions []int) Index {
	values := make([]time.Time, len(positions))
	var mask []bool
	setNA := func(i int) {
		if mask == nil {
			mask = make([]bool, len(positions))
		}
		mask[i] = true
	}
	for i, p := range positions {
		if p < 0 || ix.isNA(p) {
			setNA(i)
			continue
		}
		values[i] = ix.values[p]
	}
	return &DatetimeIndex{values: values, mask: mask, name: ix.name}
}

// SlicePos returns the positional slice [start, stop) as a new index.
func (ix *DatetimeIndex) SlicePos(start, stop int) Index {
	out := &DatetimeIndex{
		values: append([]time.Time(nil), ix.values[start:stop]...),
		name:   ix.name,
	}
	if ix.mask != nil {
		out.mask = append([]bool(nil), ix.mask[start:stop]...)
	}
	return out
}

func (ix *DatetimeIndex) Equals(other Index) bool {
	o, ok := other.(*DatetimeIndex)
	if !ok || ix.Len() != o.Len() {
		return false
	}
	for i, v := range ix.values {
		if ix.isNA(i) != o.isNA(i) {
			return false
		}
		if !ix.isNA(i) && !v.Equal(o.values[i]) {
			return false
		}
	}
	return true
}

func (ix *DatetimeIndex) Clone() Index {
	out := &DatetimeIndex{
		values: append([]time.Time(nil), ix.values...),
		name:   ix.name,
	}
	if ix.mask != nil {
		out.mask = append([]bool(nil), ix.mask...)
	}
	return out
}

func (ix *DatetimeIndex) String() string {
	return fmt.Sprintf("DatetimeIndex(%d values, name=%q)", len(ix.values), ix.name)
}

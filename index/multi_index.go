package index

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// Tuple is one hierarchical index label: one component per level. NA
// components are nil.
type Tuple []any

// String renders the tuple pandas-style: (AR, Buenos Aires); NA
// components render as NA.
func (t Tuple) String() string {
	parts := make([]string, len(t))
	for i, v := range t {
		if v == nil {
			parts[i] = "NA"
			continue
		}
		parts[i] = fmt.Sprint(v)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// MultiIndex is a real hierarchical index (v0.8): per-level unique label
// lists plus int32 code arrays, exactly the levels/codes model pandas
// uses. codes[level][row] == -1 marks an NA tuple component.
//
// Invariants: len(codes) == len(levels) == len(names); every codes[l]
// has length Len(); level values are unique and never mutated in place
// (Take/SlicePos/Clone share level slices). Codes may reference a level
// subset after Take — levels are not compacted (documented).
type MultiIndex struct {
	names  []string
	levels [][]any
	codes  [][]int32
	length int

	// levelLookup is shared by every index derived with the same level
	// lists; posLookup is per code layout (rebuilt by Take/SlicePos).
	levelLookup *miLevelLookup
	posLookup   *miPosLookup
}

// miLevelLookup lazily indexes each level's labels -> code. Numeric
// labels normalize through float64 so int 1, int64 1 and 1.0 resolve to
// the same level entry (the project-wide numeric key rule).
type miLevelLookup struct {
	once sync.Once
	maps []map[any]int32
}

// miPosLookup lazily maps the encoded full code tuple -> row positions.
type miPosLookup struct {
	once sync.Once
	m    map[string][]int
}

// levelKey normalizes a label for level-lookup maps.
func levelKey(v any) any {
	if _, isBool := v.(bool); isBool {
		return v
	}
	if f, ok := dtype.AsFloat(v); ok {
		return f
	}
	return v
}

// hashableIndexLabel reports whether a label can be a map key.
func hashableIndexLabel(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Comparable()
}

// factorizeLevel builds one level: sorted unique labels when the labels
// form one orderable family (reusing the categorical factorizer, which
// matches pandas' sorted MultiIndex levels), falling back to
// first-appearance order for mixed-family labels (documented
// implementation-defined order).
func factorizeLevel(arr []any) (levels []any, codes []int32) {
	if cat, err := column.Factorize(arr, nil, false); err == nil {
		raw, _ := cat.RawCodes()
		return cat.Categories(), append([]int32(nil), raw...)
	}
	var out []any
	seen := make(map[any]int32)
	codes = make([]int32, len(arr))
	for i, v := range arr {
		if dtype.IsNA(v) {
			codes[i] = -1
			continue
		}
		k := levelKey(v)
		code, ok := seen[k]
		if !ok {
			code = int32(len(out))
			seen[k] = code
			out = append(out, v)
		}
		codes[i] = code
	}
	return out, codes
}

// NewMultiIndexFromArrays builds a MultiIndex from parallel label
// arrays, one per level. NA labels become code -1. Level lists are the
// sorted unique labels per level (pandas parity) when orderable.
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
		return nil, fmt.Errorf("%w: %d names for %d levels", errs.ErrLengthMismatch, len(names), len(arrays))
	}
	mi := &MultiIndex{
		names:       append([]string(nil), names...),
		length:      n,
		levelLookup: &miLevelLookup{},
		posLookup:   &miPosLookup{},
	}
	for _, arr := range arrays {
		for _, v := range arr {
			if !dtype.IsNA(v) && !hashableIndexLabel(v) {
				return nil, fmt.Errorf("%w: cannot use %T as an index label", errs.ErrTypeMismatch, v)
			}
		}
		levels, codes := factorizeLevel(arr)
		mi.levels = append(mi.levels, levels)
		mi.codes = append(mi.codes, codes)
	}
	return mi, nil
}

// NewMultiIndexFromTuples builds a MultiIndex from row tuples. Every
// tuple must have the same number of components.
func NewMultiIndexFromTuples(tuples [][]any, names []string) (*MultiIndex, error) {
	if len(tuples) == 0 {
		return nil, fmt.Errorf("%w: MultiIndex needs at least one tuple", errs.ErrInvalidIndex)
	}
	width := len(tuples[0])
	if width == 0 {
		return nil, fmt.Errorf("%w: empty tuple", errs.ErrInvalidIndex)
	}
	arrays := make([][]any, width)
	for l := range arrays {
		arrays[l] = make([]any, len(tuples))
	}
	for i, t := range tuples {
		if len(t) != width {
			return nil, fmt.Errorf("%w: tuple %d has %d components, want %d", errs.ErrLengthMismatch, i, len(t), width)
		}
		for l, v := range t {
			arrays[l][i] = v
		}
	}
	return NewMultiIndexFromArrays(arrays, names)
}

// derive builds a same-levels index over new codes (Take/SlicePos):
// level lists and the level lookup are shared, the position lookup is
// rebuilt lazily for the new layout.
func (ix *MultiIndex) derive(codes [][]int32, length int) *MultiIndex {
	return &MultiIndex{
		names:       append([]string(nil), ix.names...),
		levels:      ix.levels,
		codes:       codes,
		length:      length,
		levelLookup: ix.levelLookup,
		posLookup:   &miPosLookup{},
	}
}

// NLevels returns the number of levels.
func (ix *MultiIndex) NLevels() int { return len(ix.levels) }

// Names returns the level names (copy).
func (ix *MultiIndex) Names() []string { return append([]string(nil), ix.names...) }

// Levels returns the per-level unique label lists (deep copy).
func (ix *MultiIndex) Levels() [][]any {
	out := make([][]any, len(ix.levels))
	for l, lv := range ix.levels {
		out[l] = append([]any(nil), lv...)
	}
	return out
}

// Codes returns the per-level code arrays (deep copy; -1 = NA).
func (ix *MultiIndex) Codes() [][]int32 {
	out := make([][]int32, len(ix.codes))
	for l, c := range ix.codes {
		out[l] = append([]int32(nil), c...)
	}
	return out
}

// Name joins the level names (pandas prints them as a tuple).
func (ix *MultiIndex) Name() string { return strings.Join(ix.names, ", ") }

func (ix *MultiIndex) Len() int { return ix.length }

// IsNA reports whether the tuple component at (pos, level) is missing.
func (ix *MultiIndex) IsNA(pos, level int) bool { return ix.codes[level][pos] == -1 }

// Tuple returns the label tuple at a position.
func (ix *MultiIndex) Tuple(pos int) Tuple {
	t := make(Tuple, len(ix.levels))
	for l := range ix.levels {
		code := ix.codes[l][pos]
		if code < 0 {
			continue // nil component
		}
		t[l] = ix.levels[l][code]
	}
	return t
}

// Tuples returns every label tuple.
func (ix *MultiIndex) Tuples() []Tuple {
	out := make([]Tuple, ix.length)
	for i := range out {
		out[i] = ix.Tuple(i)
	}
	return out
}

// At returns the label at a position as a Tuple.
func (ix *MultiIndex) At(pos int) any { return ix.Tuple(pos) }

func (ix *MultiIndex) Values() []any {
	out := make([]any, ix.length)
	for i := range out {
		out[i] = ix.Tuple(i)
	}
	return out
}

// levelMaps lazily builds the per-level label -> code maps (shared by
// derived indexes; race-safe under the Once).
func (ix *MultiIndex) levelMaps() []map[any]int32 {
	ix.levelLookup.once.Do(func() {
		maps := make([]map[any]int32, len(ix.levels))
		for l, lv := range ix.levels {
			m := make(map[any]int32, len(lv))
			for code, v := range lv {
				m[levelKey(v)] = int32(code)
			}
			maps[l] = m
		}
		ix.levelLookup.maps = maps
	})
	return ix.levelLookup.maps
}

// codeOf resolves one tuple component to its level code (-1 = absent).
func (ix *MultiIndex) codeOf(level int, label any) int32 {
	if !hashableIndexLabel(label) {
		return -1
	}
	if code, ok := ix.levelMaps()[level][levelKey(label)]; ok {
		return code
	}
	return -1
}

// encodeCodes builds the position-lookup key for a full code tuple.
func encodeCodes(codes []int32) string {
	var b strings.Builder
	for l, c := range codes {
		if l > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(int64(c), 10))
	}
	return b.String()
}

// positionsMap lazily builds the full-tuple -> positions map.
func (ix *MultiIndex) positionsMap() map[string][]int {
	ix.posLookup.once.Do(func() {
		m := make(map[string][]int, ix.length)
		row := make([]int32, len(ix.codes))
		for i := 0; i < ix.length; i++ {
			for l := range ix.codes {
				row[l] = ix.codes[l][i]
			}
			k := encodeCodes(row)
			m[k] = append(m[k], i)
		}
		ix.posLookup.m = m
	})
	return ix.posLookup.m
}

// resolveTuple maps a label tuple to level codes; ok is false when a
// non-NA component is not present in its level (no rows can match).
func (ix *MultiIndex) resolveTuple(t []any) ([]int32, bool) {
	codes := make([]int32, len(t))
	for l, v := range t {
		if dtype.IsNA(v) {
			codes[l] = -1
			continue
		}
		c := ix.codeOf(l, v)
		if c < 0 {
			return nil, false
		}
		codes[l] = c
	}
	return codes, true
}

// PositionsTuple returns every row whose full label tuple matches. NA
// components (nil) match NA tuple components.
func (ix *MultiIndex) PositionsTuple(t []any) []int {
	if len(t) != len(ix.levels) {
		return nil
	}
	codes, ok := ix.resolveTuple(t)
	if !ok {
		return nil
	}
	return ix.positionsMap()[encodeCodes(codes)]
}

// PositionsPrefix returns every row whose leading tuple components match
// the prefix. This scans the code arrays (documented v0.8 behavior; the
// full-tuple path uses the lookup map).
func (ix *MultiIndex) PositionsPrefix(prefix []any) []int {
	if len(prefix) == 0 || len(prefix) > len(ix.levels) {
		return nil
	}
	codes, ok := ix.resolveTuple(prefix)
	if !ok {
		return nil
	}
	var out []int
	for i := 0; i < ix.length; i++ {
		match := true
		for l, c := range codes {
			if ix.codes[l][i] != c {
				match = false
				break
			}
		}
		if match {
			out = append(out, i)
		}
	}
	return out
}

// asTuple widens the label forms accepted by Pos/Positions.
func asTuple(label any) ([]any, bool) {
	switch t := label.(type) {
	case Tuple:
		return t, true
	case []any:
		return t, true
	}
	return nil, false
}

// Pos returns the first position of a full label tuple. A bare value is
// accepted for one-level indexes.
func (ix *MultiIndex) Pos(label any) (int, bool) {
	positions := ix.Positions(label)
	if len(positions) == 0 {
		return -1, false
	}
	return positions[0], true
}

// Positions returns every position of a full label tuple.
func (ix *MultiIndex) Positions(label any) []int {
	if t, ok := asTuple(label); ok {
		return ix.PositionsTuple(t)
	}
	if len(ix.levels) == 1 {
		return ix.PositionsTuple([]any{label})
	}
	return nil
}

// Slice by label range needs an ordered MultiIndex; not implemented in
// v0.8 (documented).
func (ix *MultiIndex) Slice(start, stop any) ([]int, error) {
	return nil, errs.NotImplemented("MultiIndex.Slice by label range")
}

// Take gathers rows by position: names and level lists are preserved
// (levels are NOT compacted — codes may reference a level subset), a
// negative position produces an all-NA tuple.
func (ix *MultiIndex) Take(positions []int) Index {
	codes := make([][]int32, len(ix.codes))
	for l, src := range ix.codes {
		dst := make([]int32, len(positions))
		for i, p := range positions {
			if p < 0 {
				dst[i] = -1
				continue
			}
			dst[i] = src[p]
		}
		codes[l] = dst
	}
	return ix.derive(codes, len(positions))
}

// SlicePos returns the positional slice [start, stop) as a new index.
func (ix *MultiIndex) SlicePos(start, stop int) Index {
	codes := make([][]int32, len(ix.codes))
	for l, src := range ix.codes {
		codes[l] = append([]int32(nil), src[start:stop]...)
	}
	return ix.derive(codes, stop-start)
}

// Equals compares tuple labels (names are ignored, like pandas equals).
func (ix *MultiIndex) Equals(other Index) bool {
	o, ok := other.(*MultiIndex)
	if !ok || ix.length != o.length || len(ix.levels) != len(o.levels) {
		return false
	}
	for i := 0; i < ix.length; i++ {
		a, b := ix.Tuple(i), o.Tuple(i)
		for l := range a {
			if a[l] != b[l] {
				return false
			}
		}
	}
	return true
}

func (ix *MultiIndex) Clone() Index {
	codes := make([][]int32, len(ix.codes))
	for l, c := range ix.codes {
		codes[l] = append([]int32(nil), c...)
	}
	return ix.derive(codes, ix.length)
}

func (ix *MultiIndex) String() string {
	const maxShow = 10
	var b strings.Builder
	b.WriteString("MultiIndex([")
	shown := ix.length
	if shown > maxShow {
		shown = maxShow
	}
	for i := 0; i < shown; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(ix.Tuple(i).String())
	}
	if ix.length > maxShow {
		fmt.Fprintf(&b, ", ... (%d total)", ix.length)
	}
	fmt.Fprintf(&b, "], names=%v)", ix.names)
	return b.String()
}

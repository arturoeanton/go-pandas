package column

import (
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
)

// ConcatPart is one vertical segment of a concatenated column. A nil Col
// is an all-NA gap of Len rows (a column missing from one frame in an
// outer concat).
type ConcatPart struct {
	Col Column
	Len int
}

// ConcatParts stacks column segments vertically with typed storage
// (v0.6.1): same-dtype parts append into one typed buffer; compatible
// numeric mixes promote once (via the shared dtype promotion rules) into
// one typed buffer; anything else falls back to object — for that column
// only. Masks copy; NA gaps mask their whole segment.
func ConcatParts(parts []ConcatPart) Column {
	total := 0
	for _, p := range parts {
		total += p.Len
	}
	// Categorical parts union their categories and stack codes (v0.7).
	if out, ok := tryConcatCategorical(parts, total); ok {
		return out
	}
	// Same concrete dtype fast paths.
	if out, ok := tryConcatSame[bool](parts, total); ok {
		return out
	}
	if out, ok := tryConcatSame[int](parts, total); ok {
		return out
	}
	if out, ok := tryConcatSame[int64](parts, total); ok {
		return out
	}
	if out, ok := tryConcatSame[float32](parts, total); ok {
		return out
	}
	if out, ok := tryConcatSame[float64](parts, total); ok {
		return out
	}
	if out, ok := tryConcatSame[string](parts, total); ok {
		return out
	}
	if out, ok := tryConcatSame[time.Time](parts, total); ok {
		return out
	}
	// Mixed numeric dtypes promote once.
	if out, ok := tryConcatNumeric(parts, total); ok {
		return out
	}
	// Object fallback: boxed append (historical behavior).
	values := make([]any, 0, total)
	for _, p := range parts {
		if p.Col == nil {
			for i := 0; i < p.Len; i++ {
				values = append(values, nil)
			}
			continue
		}
		values = append(values, p.Col.Values()...)
	}
	return Infer(values)
}

// tryConcatCategorical stacks categorical parts in code space: the
// result keeps the categorical dtype with the union of the category
// lists (first-seen order, like pandas' union_categoricals — plain
// pd.concat would downgrade differing categories to object, a documented
// difference). Ordered survives only when every part is ordered with an
// identical category list.
func tryConcatCategorical(parts []ConcatPart, total int) (Column, bool) {
	cats := make([]*CategoricalColumn, len(parts))
	present := false
	for i, p := range parts {
		if p.Col == nil {
			continue
		}
		cc, ok := AsCategorical(p.Col)
		if !ok {
			return nil, false
		}
		cats[i] = cc
		present = true
	}
	if !present {
		return nil, false
	}
	lookup := make(map[any]int32)
	var categories []any
	ordered := true
	for _, cc := range cats {
		if cc == nil {
			continue
		}
		if !cc.Ordered() {
			ordered = false
		}
		for _, cat := range cc.Categories() {
			if _, ok := lookup[cat]; !ok {
				lookup[cat] = int32(len(categories))
				categories = append(categories, cat)
			}
		}
	}
	if ordered {
		for _, cc := range cats {
			if cc == nil {
				continue
			}
			pc := cc.Categories()
			if len(pc) != len(categories) {
				ordered = false
				break
			}
			for i := range pc {
				if pc[i] != categories[i] {
					ordered = false
					break
				}
			}
		}
	}
	codes := make([]int32, 0, total)
	mask := make([]bool, 0, total)
	for pi, p := range parts {
		if p.Col == nil {
			for i := 0; i < p.Len; i++ {
				codes = append(codes, -1)
				mask = append(mask, true)
			}
			continue
		}
		cc := cats[pi]
		remap := make([]int32, cc.CategoryCount())
		for i, cat := range cc.Categories() {
			remap[i] = lookup[cat]
		}
		pcodes, pmask := cc.RawCodes()
		for i, code := range pcodes {
			if pmask[i] {
				codes = append(codes, -1)
				mask = append(mask, true)
				continue
			}
			codes = append(codes, remap[code])
			mask = append(mask, false)
		}
	}
	return NewCategorical(codes, categories, ordered, mask), true
}

// tryConcatSame appends parts when every present part is the same
// concrete typed column.
func tryConcatSame[T any](parts []ConcatPart, total int) (Column, bool) {
	var proto *typedColumn[T]
	for _, p := range parts {
		if p.Col == nil {
			continue
		}
		tc, ok := p.Col.(*typedColumn[T])
		if !ok {
			return nil, false
		}
		if proto == nil {
			proto = tc
		}
	}
	if proto == nil {
		return nil, false // all gaps: object all-NA fallback
	}
	data := make([]T, 0, total)
	mask := make([]bool, 0, total)
	for _, p := range parts {
		if p.Col == nil {
			var zero T
			for i := 0; i < p.Len; i++ {
				data = append(data, zero)
				mask = append(mask, true)
			}
			continue
		}
		tc := p.Col.(*typedColumn[T])
		data = append(data, tc.data...)
		mask = append(mask, tc.mask...)
	}
	return proto.with(data, mask), true
}

// tryConcatNumeric promotes mixed numeric parts into one typed buffer.
func tryConcatNumeric(parts []ConcatPart, total int) (Column, bool) {
	promoted := dtype.Invalid
	for _, p := range parts {
		if p.Col == nil {
			continue
		}
		if IsObjectBacked(p.Col) {
			return nil, false
		}
		dt := p.Col.DType()
		if !dtype.IsNumeric(dt) && dt != dtype.Bool {
			return nil, false
		}
		promoted = dtype.Promote(promoted, dt)
	}
	if promoted == dtype.Invalid {
		return nil, false
	}
	if promoted == dtype.Bool {
		return nil, false // bool-only mixes are handled by the same-dtype path
	}
	// Gather each part through the float64 buffer, storing into the
	// promoted representation.
	mask := make([]bool, 0, total)
	floats := make([]float64, 0, total)
	for _, p := range parts {
		if p.Col == nil {
			for i := 0; i < p.Len; i++ {
				floats = append(floats, 0)
				mask = append(mask, true)
			}
			continue
		}
		vals, m, ok := p.Col.Float64s()
		if !ok {
			return nil, false
		}
		floats = append(floats, vals...)
		mask = append(mask, m...)
	}
	switch {
	case dtype.IsInteger(promoted):
		data := make([]int64, total)
		for i, v := range floats {
			data[i] = int64(v)
		}
		if promoted == dtype.Int {
			ints := make([]int, total)
			for i, v := range data {
				ints[i] = int(v)
			}
			return NewInt(ints, mask), true
		}
		return NewInt64(data, mask), true
	case promoted == dtype.Float32:
		data := make([]float32, total)
		for i, v := range floats {
			data[i] = float32(v)
		}
		return NewFloat32(data, mask), true
	default:
		return NewFloat64(floats, mask), true
	}
}

package column

import "time"

// GatherCoalesce builds a column that takes a's value at aRows[i] when
// present and b's value at bRows[i] otherwise (-1 = missing side) — the
// shape of a merged key column. Same-typed pairs gather without boxing;
// ok is false when the pair needs the boxed fallback.
func GatherCoalesce(a, b Column, aRows, bRows []int) (Column, bool) {
	switch ac := a.(type) {
	case *typedColumn[bool]:
		if bc, ok := b.(*typedColumn[bool]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *typedColumn[int]:
		if bc, ok := b.(*typedColumn[int]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *typedColumn[int64]:
		if bc, ok := b.(*typedColumn[int64]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *typedColumn[float32]:
		if bc, ok := b.(*typedColumn[float32]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *typedColumn[float64]:
		if bc, ok := b.(*typedColumn[float64]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *typedColumn[string]:
		if bc, ok := b.(*typedColumn[string]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *typedColumn[time.Time]:
		if bc, ok := b.(*typedColumn[time.Time]); ok {
			return coalesce(ac, bc, aRows, bRows), true
		}
	case *CategoricalColumn:
		if bc, ok := b.(*CategoricalColumn); ok {
			return coalesceCategorical(ac, bc, aRows, bRows), true
		}
	}
	return nil, false
}

// coalesceCategorical merges two categorical key columns in code space:
// the output categories are a's list extended by b-only categories, so
// outer-merge key columns stay categorical (v0.7). Ordered survives only
// when both sides are ordered with identical categories.
func coalesceCategorical(a, b *CategoricalColumn, aRows, bRows []int) Column {
	byLabel := make(map[any]int32, a.CategoryCount())
	categories := a.Categories()
	for id, cat := range categories {
		byLabel[cat] = int32(id)
	}
	identical := b.CategoryCount() == len(categories)
	remap := make([]int32, b.CategoryCount())
	for i, cat := range b.Categories() {
		if id, ok := byLabel[cat]; ok {
			remap[i] = id
			if identical && int(id) != i {
				identical = false
			}
			continue
		}
		remap[i] = int32(len(categories))
		categories = append(categories, cat)
		identical = false
	}
	ordered := a.Ordered() && b.Ordered() && identical

	acodes, amask := a.RawCodes()
	bcodes, bmask := b.RawCodes()
	n := len(aRows)
	codes := make([]int32, n)
	mask := make([]bool, n)
	for i := 0; i < n; i++ {
		switch {
		case aRows[i] >= 0 && !amask[aRows[i]]:
			codes[i] = acodes[aRows[i]]
		case bRows[i] >= 0 && !bmask[bRows[i]]:
			codes[i] = remap[bcodes[bRows[i]]]
		default:
			codes[i] = -1
			mask[i] = true
		}
	}
	return NewCategorical(codes, categories, ordered, mask)
}

func coalesce[T any](a, b *typedColumn[T], aRows, bRows []int) Column {
	n := len(aRows)
	data := make([]T, n)
	mask := make([]bool, n)
	for i := 0; i < n; i++ {
		switch {
		case aRows[i] >= 0 && !a.mask[aRows[i]]:
			data[i] = a.data[aRows[i]]
		case bRows[i] >= 0 && !b.mask[bRows[i]]:
			data[i] = b.data[bRows[i]]
		default:
			mask[i] = true
		}
	}
	return a.with(data, mask)
}

// GatherCoalesceBoxed is the fallback for mixed-typed key pairs.
func GatherCoalesceBoxed(a, b Column, aRows, bRows []int) Column {
	values := make([]any, len(aRows))
	for i := range aRows {
		switch {
		case aRows[i] >= 0 && !a.IsNA(aRows[i]):
			values[i] = a.Value(aRows[i])
		case bRows[i] >= 0 && !b.IsNA(bRows[i]):
			values[i] = b.Value(bRows[i])
		}
	}
	return Infer(values)
}

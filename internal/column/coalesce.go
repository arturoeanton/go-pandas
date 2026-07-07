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
	}
	return nil, false
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

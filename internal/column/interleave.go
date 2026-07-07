package column

import "time"

// StackInterleave builds one column holding the row-major interleave of
// same-typed columns (row0col0, row0col1, ..., row1col0, ...) — the
// value layout of DataFrame.Stack (v0.10.1). ok is false when the
// columns do not all share one typed backing; callers fall back to the
// boxed path.
func StackInterleave(cols []Column) (Column, bool) {
	if len(cols) == 0 {
		return nil, false
	}
	if out, ok := stackInterleave[bool](cols); ok {
		return out, true
	}
	if out, ok := stackInterleave[int](cols); ok {
		return out, true
	}
	if out, ok := stackInterleave[int64](cols); ok {
		return out, true
	}
	if out, ok := stackInterleave[float32](cols); ok {
		return out, true
	}
	if out, ok := stackInterleave[float64](cols); ok {
		return out, true
	}
	if out, ok := stackInterleave[string](cols); ok {
		return out, true
	}
	if out, ok := stackInterleave[time.Time](cols); ok {
		return out, true
	}
	return nil, false
}

// UnstackGather scatters a column's rows into per-output-column typed
// cell grids (v1.0-rc): dst[colOf[i]][rowOf[i]] = src[i]; unassigned
// cells stay NA. ok is false for non-typed backings (object,
// categorical) — callers keep the boxed fallback.
func UnstackGather(src Column, colOf, rowOf []int, nCols, nRows int) ([]Column, bool) {
	if out, ok := unstackGather[bool](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	if out, ok := unstackGather[int](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	if out, ok := unstackGather[int64](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	if out, ok := unstackGather[float32](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	if out, ok := unstackGather[float64](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	if out, ok := unstackGather[string](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	if out, ok := unstackGather[time.Time](src, colOf, rowOf, nCols, nRows); ok {
		return out, true
	}
	return nil, false
}

func unstackGather[T any](src Column, colOf, rowOf []int, nCols, nRows int) ([]Column, bool) {
	tc, ok := src.(*typedColumn[T])
	if !ok {
		return nil, false
	}
	data := make([][]T, nCols)
	mask := make([][]bool, nCols)
	for j := range data {
		data[j] = make([]T, nRows)
		mask[j] = make([]bool, nRows)
		for r := range mask[j] {
			mask[j][r] = true // unfilled cells are NA
		}
	}
	for i := range colOf {
		j, r := colOf[i], rowOf[i]
		data[j][r] = tc.data[i]
		mask[j][r] = tc.mask[i]
	}
	out := make([]Column, nCols)
	for j := range out {
		out[j] = tc.with(data[j], mask[j])
	}
	return out, true
}

func stackInterleave[T any](cols []Column) (Column, bool) {
	typed := make([]*typedColumn[T], len(cols))
	for j, c := range cols {
		tc, ok := c.(*typedColumn[T])
		if !ok {
			return nil, false
		}
		typed[j] = tc
	}
	n, k := typed[0].Len(), len(typed)
	data := make([]T, n*k)
	mask := make([]bool, n*k)
	for j, tc := range typed {
		for i := 0; i < n; i++ {
			data[i*k+j] = tc.data[i]
			mask[i*k+j] = tc.mask[i]
		}
	}
	return typed[0].with(data, mask), true
}

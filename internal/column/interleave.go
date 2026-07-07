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

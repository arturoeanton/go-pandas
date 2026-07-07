package dataframe

import (
	"fmt"
	"sort"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	"github.com/arturoeanton/go-pandas/series"
)

// Stack pivots the columns into an inner index level (v0.10), pandas'
// df.stack(): the result is a Series whose MultiIndex is the original
// index levels plus a column-name level, laid out row-major
// ((row0, colA), (row0, colB), (row1, colA), ...). NA cells are KEPT,
// matching pandas' future stack behavior (classic stack dropped them —
// documented difference). Values re-infer typed storage: homogeneous
// columns stay typed, mixed dtypes fall back to object.
//
// Note: the v0.1 placeholder returned (*DataFrame, error) and always
// ErrNotImplemented; since v0.10 Stack returns the pandas-like Series.
func (df *DataFrame) Stack() (*series.Series, error) {
	if len(df.columns) == 0 {
		return nil, fmt.Errorf("%w: Stack needs at least one column", errs.ErrInvalidOperation)
	}
	n, k := df.Len(), len(df.columns)
	names := df.Columns()

	// Index levels: the existing labels (per level for a MultiIndex)
	// plus the column-name level.
	var baseArrays [][]any
	var levelNames []string
	switch ix := df.index.(type) {
	case *index.MultiIndex:
		baseArrays = make([][]any, ix.NLevels())
		for l := range baseArrays {
			baseArrays[l] = make([]any, 0, n*k)
		}
		levelNames = append(ix.Names(), "")
	default:
		baseArrays = [][]any{make([]any, 0, n*k)}
		levelNames = []string{df.index.Name(), ""}
	}
	colLevel := make([]any, 0, n*k)
	values := make([]any, 0, n*k)

	colValues := make([][]any, k)
	for j, c := range df.columns {
		colValues[j] = c.Values()
	}
	for i := 0; i < n; i++ {
		var rowLabels []any
		if mi, ok := df.index.(*index.MultiIndex); ok {
			rowLabels = mi.Tuple(i)
		} else {
			rowLabels = []any{df.index.At(i)}
		}
		for j := 0; j < k; j++ {
			for l := range baseArrays {
				baseArrays[l] = append(baseArrays[l], rowLabels[l])
			}
			colLevel = append(colLevel, names[j])
			values = append(values, colValues[j][i])
		}
	}
	arrays := append(baseArrays, colLevel)
	mi, err := index.NewMultiIndexFromArrays(arrays, levelNames)
	if err != nil {
		return nil, err
	}
	return series.Assemble("", column.Infer(values), mi), nil
}

// UnstackSeries pivots the LAST MultiIndex level of a series into
// columns (v0.10), pandas' s.unstack(): rows are the leading level(s)
// in level order (sorted labels), columns are the observed last-level
// labels in level order, missing combinations are NA, and duplicate
// (row, column) entries are an error — aggregate with PivotTable
// instead.
func UnstackSeries(s *series.Series) (*DataFrame, error) {
	mi, ok := s.Index().(*index.MultiIndex)
	if !ok || mi.NLevels() < 2 {
		return nil, fmt.Errorf("%w: Unstack needs a MultiIndex with at least two levels", errs.ErrInvalidIndex)
	}
	codes := mi.Codes()
	levels := mi.Levels()
	last := mi.NLevels() - 1

	// Observed column labels, in level order (levels are sorted).
	colSeen := make(map[int32]bool)
	for _, c := range codes[last] {
		if c >= 0 {
			colSeen[c] = true
		}
	}
	var colCodes []int32
	for c := range levels[last] {
		if colSeen[int32(c)] {
			colCodes = append(colCodes, int32(c))
		}
	}
	colPos := make(map[int32]int, len(colCodes))
	for j, c := range colCodes {
		colPos[c] = j
	}

	// Row keys: leading level code tuples, first grouped then sorted by
	// code order (level order == sorted labels, pandas parity).
	type rowKey string
	encode := func(i int) rowKey {
		key := make([]byte, 0, 8*last)
		for l := 0; l < last; l++ {
			key = append(key, fmt.Sprintf("%d,", codes[l][i])...)
		}
		return rowKey(key)
	}
	rowID := make(map[rowKey]int)
	var rowFirst []int
	rowOf := make([]int, s.Len())
	for i := 0; i < s.Len(); i++ {
		k := encode(i)
		id, ok := rowID[k]
		if !ok {
			id = len(rowFirst)
			rowID[k] = id
			rowFirst = append(rowFirst, i)
		}
		rowOf[i] = id
	}
	order := make([]int, len(rowFirst))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ra, rb := rowFirst[order[a]], rowFirst[order[b]]
		for l := 0; l < last; l++ {
			ca, cb := codes[l][ra], codes[l][rb]
			if ca != cb {
				return ca < cb
			}
		}
		return false
	})
	rowRank := make([]int, len(order))
	for rank, id := range order {
		rowRank[id] = rank
	}

	// Fill the cell grid; duplicates error like pandas.
	nRows, nCols := len(order), len(colCodes)
	cells := make([][]any, nCols)
	filled := make([][]bool, nCols)
	for j := range cells {
		cells[j] = make([]any, nRows)
		filled[j] = make([]bool, nRows)
	}
	sv := s.Values()
	for i := 0; i < s.Len(); i++ {
		c := codes[last][i]
		if c < 0 {
			return nil, fmt.Errorf("%w: cannot unstack an NA label in the column level", errs.ErrInvalidIndex)
		}
		j := colPos[c]
		r := rowRank[rowOf[i]]
		if filled[j][r] {
			return nil, fmt.Errorf("%w: index contains duplicate entries, cannot unstack (aggregate with PivotTable)", errs.ErrInvalidOperation)
		}
		filled[j][r] = true
		cells[j][r] = sv[i]
	}

	// Row index: one leading level -> plain index, several -> MultiIndex.
	var idx index.Index
	if last == 1 {
		labels := make([]any, nRows)
		for rank, id := range order {
			c := codes[0][rowFirst[id]]
			if c >= 0 {
				labels[rank] = levels[0][c]
			}
		}
		idx = index.FromLabels(labels, mi.Names()[0])
	} else {
		arrays := make([][]any, last)
		for l := 0; l < last; l++ {
			arrays[l] = make([]any, nRows)
			for rank, id := range order {
				c := codes[l][rowFirst[id]]
				if c >= 0 {
					arrays[l][rank] = levels[l][c]
				}
			}
		}
		var err error
		if idx, err = index.NewMultiIndexFromArrays(arrays, mi.Names()[:last]); err != nil {
			return nil, err
		}
	}

	cols := make([]*series.Series, nCols)
	for j, c := range colCodes {
		name := fmt.Sprint(levels[last][c])
		cols[j] = series.Assemble(name, column.Infer(cells[j]), idx)
	}
	return newFrame(cols, idx)
}

// Unstack pivots the last MultiIndex level into columns (v0.10). A
// single-column frame behaves like UnstackSeries on that column; with
// several columns the output flattens names to "column_label"
// (go-pandas has no MultiIndex columns — documented difference).
func (df *DataFrame) Unstack() (*DataFrame, error) {
	if _, ok := df.index.(*index.MultiIndex); !ok {
		return nil, fmt.Errorf("%w: Unstack needs a MultiIndex", errs.ErrInvalidIndex)
	}
	if len(df.columns) == 1 {
		return UnstackSeries(df.columns[0])
	}
	var out *DataFrame
	for _, c := range df.columns {
		part, err := UnstackSeries(c)
		if err != nil {
			return nil, err
		}
		renamed := make([]*series.Series, len(part.columns))
		for j, pc := range part.columns {
			renamed[j] = pc.Rename(c.Name() + "_" + pc.Name())
		}
		partFrame, err := newFrame(renamed, part.index)
		if err != nil {
			return nil, err
		}
		if out == nil {
			out = partFrame
			continue
		}
		merged, err := concatColumns([]*DataFrame{out, partFrame})
		if err != nil {
			return nil, err
		}
		out = merged
	}
	return out, nil
}

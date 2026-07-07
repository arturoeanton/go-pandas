package dataframe

import (
	"fmt"
	"sort"
	"strconv"

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

	// Row levels: reuse MultiIndex levels/codes, or factorize the plain
	// labels ONCE (v0.10.1 — previously every cell boxed its row label
	// into a per-level array that re-factorized n*k values).
	var rowLevels [][]any
	var rowCodes [][]int32
	var levelNames []string
	switch ix := df.index.(type) {
	case *index.MultiIndex:
		rowLevels = ix.Levels()
		rowCodes = ix.Codes()
		levelNames = append(ix.Names(), "")
	default:
		lv, codes := index.FactorizeLabels(df.index.Values())
		rowLevels = [][]any{lv}
		rowCodes = [][]int32{codes}
		levelNames = []string{df.index.Name(), ""}
	}
	// Column level keeps the original column order (pandas' stack
	// column level; frame column names are unique by construction).
	colLevel := make([]any, k)
	for j, name := range names {
		colLevel[j] = name
	}

	total := n * k
	outCodes := make([][]int32, len(rowCodes)+1)
	for l := range rowCodes {
		expanded := make([]int32, total)
		for i := 0; i < n; i++ {
			c := rowCodes[l][i]
			base := i * k
			for j := 0; j < k; j++ {
				expanded[base+j] = c
			}
		}
		outCodes[l] = expanded
	}
	colCodes := make([]int32, total)
	for i := 0; i < n; i++ {
		base := i * k
		for j := 0; j < k; j++ {
			colCodes[base+j] = int32(j)
		}
	}
	outCodes[len(rowCodes)] = colCodes
	mi, err := index.NewMultiIndexFromCodes(append(rowLevels, colLevel), outCodes, levelNames)
	if err != nil {
		return nil, err
	}

	// Values: same-typed columns interleave into one typed buffer
	// (v0.10.1); mixed dtypes keep the boxed Infer fallback.
	storages := make([]column.Column, k)
	for j, c := range df.columns {
		storages[j] = c.Storage()
	}
	if col, ok := column.StackInterleave(storages); ok {
		return series.Assemble("", col, mi), nil
	}
	values := make([]any, 0, total)
	colValues := make([][]any, k)
	for j, c := range df.columns {
		colValues[j] = c.Values()
	}
	for i := 0; i < n; i++ {
		for j := 0; j < k; j++ {
			values = append(values, colValues[j][i])
		}
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
	// code order (level order == sorted labels, pandas parity). One
	// leading level (the 2-level common case) keys by the code itself;
	// deeper indexes build compact byte keys (v1.0-rc — previously
	// fmt.Sprintf per row).
	var rowFirst []int
	rowOf := make([]int, s.Len())
	if last == 1 {
		rowID := make(map[int32]int)
		lead := codes[0]
		for i := 0; i < s.Len(); i++ {
			id, ok := rowID[lead[i]]
			if !ok {
				id = len(rowFirst)
				rowID[lead[i]] = id
				rowFirst = append(rowFirst, i)
			}
			rowOf[i] = id
		}
	} else {
		rowID := make(map[string]int)
		buf := make([]byte, 0, 12*last)
		for i := 0; i < s.Len(); i++ {
			buf = buf[:0]
			for l := 0; l < last; l++ {
				buf = strconv.AppendInt(buf, int64(codes[l][i]), 10)
				buf = append(buf, ',')
			}
			id, ok := rowID[string(buf)]
			if !ok {
				id = len(rowFirst)
				rowID[string(buf)] = id
				rowFirst = append(rowFirst, i)
			}
			rowOf[i] = id
		}
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

	// Resolve every source row's output cell; duplicates error like
	// pandas.
	nRows, nCols := len(order), len(colCodes)
	cIdx := make([]int, s.Len())
	rIdx := make([]int, s.Len())
	filled := make([][]bool, nCols)
	for j := range filled {
		filled[j] = make([]bool, nRows)
	}
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
		cIdx[i], rIdx[i] = j, r
	}
	// Typed scatter for typed backings (v1.0-rc); boxed fallback for
	// object/categorical sources (categorical unstacks to its labels'
	// inferred dtype — documented).
	typedCols, typedOK := column.UnstackGather(s.Storage(), cIdx, rIdx, nCols, nRows)
	var cells [][]any
	if !typedOK {
		cells = make([][]any, nCols)
		for j := range cells {
			cells[j] = make([]any, nRows)
		}
		sv := s.Values()
		for i := 0; i < s.Len(); i++ {
			cells[cIdx[i]][rIdx[i]] = sv[i]
		}
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
		if typedOK {
			cols[j] = series.Assemble(name, typedCols[j], idx)
			continue
		}
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

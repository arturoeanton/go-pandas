// Package dataframe implements the pandas-style DataFrame: a 2-D table of
// labeled, typed columns sharing a row index.
package dataframe

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/display"
	"github.com/arturoeanton/go-pandas/ndarray"
	"github.com/arturoeanton/go-pandas/series"
)

// DataFrame is a columnar table: an ordered list of Series sharing one row
// index.
type DataFrame struct {
	columns []*series.Series
	byName  map[string]int
	index   index.Index
}

// Shape returns (rows, cols).
func (df *DataFrame) Shape() (rows int, cols int) {
	return df.Len(), len(df.columns)
}

// Len returns the number of rows.
func (df *DataFrame) Len() int {
	if df.index != nil {
		return df.index.Len()
	}
	if len(df.columns) > 0 {
		return df.columns[0].Len()
	}
	return 0
}

// Empty reports whether the frame has no rows or no columns.
func (df *DataFrame) Empty() bool {
	return df.Len() == 0 || len(df.columns) == 0
}

// Columns returns the column names in order.
func (df *DataFrame) Columns() []string {
	out := make([]string, len(df.columns))
	for i, c := range df.columns {
		out[i] = c.Name()
	}
	return out
}

// Index returns the row labels.
func (df *DataFrame) Index() index.Index { return df.index }

// DTypes maps each column name to its dtype.
func (df *DataFrame) DTypes() map[string]dtype.DType {
	out := make(map[string]dtype.DType, len(df.columns))
	for _, c := range df.columns {
		out[c.Name()] = c.DType()
	}
	return out
}

// Values returns the cell values row-major (missing entries are nil).
func (df *DataFrame) Values() [][]any { return df.ToRows() }

// ToRows returns the rows as slices ordered like Columns().
func (df *DataFrame) ToRows() [][]any {
	rows := df.Len()
	out := make([][]any, rows)
	colValues := make([][]any, len(df.columns))
	for j, c := range df.columns {
		colValues[j] = c.Values()
	}
	for i := 0; i < rows; i++ {
		row := make([]any, len(df.columns))
		for j := range df.columns {
			row[j] = colValues[j][i]
		}
		out[i] = row
	}
	return out
}

// ToRecords returns the rows as column-name -> value maps.
func (df *DataFrame) ToRecords() []map[string]any {
	names := df.Columns()
	rows := df.ToRows()
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		rec := make(map[string]any, len(names))
		for j, name := range names {
			rec[name] = row[j]
		}
		out[i] = rec
	}
	return out
}

// ToNDArray converts numeric columns (all, or the named subset) into a 2-D
// array; missing values become NaN.
func (df *DataFrame) ToNDArray(columns ...string) (*ndarray.NDArray, error) {
	sub := df
	if len(columns) > 0 {
		var err error
		sub, err = df.Select(columns...)
		if err != nil {
			return nil, err
		}
	}
	rows, cols := sub.Shape()
	data := make([]float64, 0, rows*cols)
	colFloats := make([][]float64, cols)
	for j, c := range sub.columns {
		fs, err := c.ToFloat64()
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", c.Name(), err)
		}
		colFloats[j] = fs
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			data = append(data, colFloats[j][i])
		}
	}
	return ndarray.FromSlice(data, rows, cols)
}

// Copy returns a deep copy.
func (df *DataFrame) Copy() *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.Copy()
	}
	out, _ := newFrame(cols, df.index)
	return out
}

// String renders the frame with the current display options.
func (df *DataFrame) String() string {
	opts := display.Get()
	return df.Repr(opts.MaxRows, opts.MaxCols)
}

// Repr renders the frame limited to maxRows/maxCols, pandas-style: index
// leftmost, aligned columns, "..." on truncation, <NA> for missing.
func (df *DataFrame) Repr(maxRows, maxCols int) string {
	rows, cols := df.Shape()
	if cols == 0 {
		return fmt.Sprintf("Empty DataFrame\nColumns: []\nIndex: %d entries", rows)
	}
	shownRows := rows
	rowsTruncated := false
	if maxRows > 0 && rows > maxRows {
		shownRows = maxRows
		rowsTruncated = true
	}
	shownCols := cols
	colsTruncated := false
	if maxCols > 0 && cols > maxCols {
		shownCols = maxCols
		colsTruncated = true
	}
	// Build the cell grid: header + rows; column 0 is the index.
	grid := make([][]string, shownRows+1)
	grid[0] = make([]string, shownCols+1)
	grid[0][0] = ""
	for j := 0; j < shownCols; j++ {
		grid[0][j+1] = df.columns[j].Name()
	}
	for i := 0; i < shownRows; i++ {
		grid[i+1] = make([]string, shownCols+1)
		grid[i+1][0] = fmt.Sprint(df.index.At(i))
		for j := 0; j < shownCols; j++ {
			v, _ := df.columns[j].At(i)
			grid[i+1][j+1] = series.FormatValue(v, dtype.IsNA(v))
		}
	}
	widths := make([]int, shownCols+1)
	for _, row := range grid {
		for j, cell := range row {
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}
	var b strings.Builder
	for r, row := range grid {
		for j, cell := range row {
			if j > 0 {
				b.WriteString("  ")
			}
			b.WriteString(fmt.Sprintf("%-*s", widths[j], cell))
		}
		b.WriteString("\n")
		_ = r
	}
	if rowsTruncated {
		b.WriteString("...\n")
	}
	if colsTruncated {
		b.WriteString(fmt.Sprintf("[%d columns total]\n", cols))
	}
	b.WriteString(fmt.Sprintf("\n[%d rows x %d columns]", rows, cols))
	return b.String()
}

// Info returns a concise summary: shape, index type, per-column dtype and
// non-null counts.
func (df *DataFrame) Info() string {
	rows, cols := df.Shape()
	var b strings.Builder
	b.WriteString("<go-pandas.DataFrame>\n")
	b.WriteString(fmt.Sprintf("%s\n", df.index.String()))
	b.WriteString(fmt.Sprintf("Data columns (total %d columns):\n", cols))
	for i, c := range df.columns {
		b.WriteString(fmt.Sprintf(" %d  %-15s %d non-null  %s\n", i, c.Name(), c.Count(), c.DType()))
	}
	b.WriteString(fmt.Sprintf("rows: %d", rows))
	return b.String()
}

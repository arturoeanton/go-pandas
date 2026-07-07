package dataframe

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/ndarray"
	"github.com/arturoeanton/go-pandas/series"
)

// DataFrameOption customizes construction.
type DataFrameOption func(*builderOptions)

type builderOptions struct {
	index       index.Index
	columnOrder []string
}

// WithDataFrameIndex attaches an explicit row index.
func WithDataFrameIndex(idx index.Index) DataFrameOption {
	return func(o *builderOptions) { o.index = idx }
}

// WithColumnOrder fixes the column order for map-based constructors (Go
// maps are unordered; without this option columns are sorted by name).
func WithColumnOrder(columns ...string) DataFrameOption {
	return func(o *builderOptions) { o.columnOrder = columns }
}

// newFrame assembles a DataFrame from columns, validating lengths and
// aligning every column to the frame index.
func newFrame(cols []*series.Series, idx index.Index) (*DataFrame, error) {
	n := -1
	for _, c := range cols {
		if n == -1 {
			n = c.Len()
		} else if c.Len() != n {
			return nil, fmt.Errorf("%w: column %q has %d rows, expected %d", errs.ErrLengthMismatch, c.Name(), c.Len(), n)
		}
	}
	if n == -1 {
		n = 0
	}
	if idx == nil {
		if len(cols) > 0 {
			idx = cols[0].Index().Clone()
		} else {
			idx = index.NewRangeIndex(0)
		}
	}
	if idx.Len() != n {
		return nil, fmt.Errorf("%w: index has %d entries for %d rows", errs.ErrLengthMismatch, idx.Len(), n)
	}
	df := &DataFrame{index: idx, byName: make(map[string]int, len(cols))}
	for _, c := range cols {
		if _, dup := df.byName[c.Name()]; dup {
			return nil, fmt.Errorf("%w: duplicate column %q", errs.ErrInvalidOperation, c.Name())
		}
		df.byName[c.Name()] = len(df.columns)
		// Columns already carrying this exact index (typed gather paths)
		// attach without the WithIndexed deep copy (v0.4.1).
		if c.Index() == idx {
			df.columns = append(df.columns, c)
		} else {
			df.columns = append(df.columns, c.WithIndexed(idx))
		}
	}
	return df, nil
}

// NewDataFrame builds a frame from Series columns. All columns must share
// the same length; the first column's index becomes the frame index.
func NewDataFrame(cols ...*series.Series) (*DataFrame, error) {
	return newFrame(cols, nil)
}

// DataFrameFromRecords builds a frame from row maps. Column order is
// alphabetical unless WithColumnOrder is given. Missing keys become NA.
func DataFrameFromRecords(records []map[string]any, opts ...DataFrameOption) (*DataFrame, error) {
	o := applyOptions(opts)
	names := o.columnOrder
	if names == nil {
		seen := map[string]bool{}
		for _, rec := range records {
			for k := range rec {
				seen[k] = true
			}
		}
		for k := range seen {
			names = append(names, k)
		}
		sort.Strings(names)
	}
	cols := make([]*series.Series, len(names))
	for j, name := range names {
		values := make([]any, len(records))
		for i, rec := range records {
			values[i] = rec[name]
		}
		cols[j] = series.NewSeries(name, values)
	}
	return newFrame(cols, o.index)
}

// DataFrameFromRows builds a frame from ordered column names and rows.
func DataFrameFromRows(columns []string, rows [][]any, opts ...DataFrameOption) (*DataFrame, error) {
	o := applyOptions(opts)
	cols := make([]*series.Series, len(columns))
	for j, name := range columns {
		values := make([]any, len(rows))
		for i, row := range rows {
			if j < len(row) {
				values[i] = row[j]
			}
		}
		cols[j] = series.NewSeries(name, values)
	}
	return newFrame(cols, o.index)
}

// DataFrameFromMap builds a frame from column name -> values. Column order
// is alphabetical unless WithColumnOrder is given.
func DataFrameFromMap(data map[string][]any, opts ...DataFrameOption) (*DataFrame, error) {
	o := applyOptions(opts)
	names := o.columnOrder
	if names == nil {
		for k := range data {
			names = append(names, k)
		}
		sort.Strings(names)
	}
	cols := make([]*series.Series, len(names))
	for j, name := range names {
		values, ok := data[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errs.ErrColumnNotFound, name)
		}
		cols[j] = series.NewSeries(name, values)
	}
	return newFrame(cols, o.index)
}

// DataFrameFromStructs builds a frame from a slice of structs; exported
// fields become columns (in declaration order). A `pd:"name"` tag renames
// a column; `pd:"-"` skips a field.
func DataFrameFromStructs(v any, opts ...DataFrameOption) (*DataFrame, error) {
	o := applyOptions(opts)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("%w: DataFrameFromStructs expects a slice of structs, got %T", errs.ErrTypeMismatch, v)
	}
	elemType := rv.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: DataFrameFromStructs expects struct elements, got %s", errs.ErrTypeMismatch, elemType)
	}
	type fieldInfo struct {
		idx  int
		name string
	}
	var fields []fieldInfo
	for i := 0; i < elemType.NumField(); i++ {
		f := elemType.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Name
		if tag, ok := f.Tag.Lookup("pd"); ok {
			if tag == "-" {
				continue
			}
			name = tag
		}
		fields = append(fields, fieldInfo{idx: i, name: name})
	}
	n := rv.Len()
	cols := make([]*series.Series, len(fields))
	for j, f := range fields {
		values := make([]any, n)
		for i := 0; i < n; i++ {
			elem := rv.Index(i)
			if elem.Kind() == reflect.Ptr {
				if elem.IsNil() {
					continue
				}
				elem = elem.Elem()
			}
			values[i] = elem.Field(f.idx).Interface()
		}
		cols[j] = series.NewSeries(f.name, values)
	}
	return newFrame(cols, o.index)
}

// DataFrameFromNDArray converts a 2-D array to a frame with named columns.
func DataFrameFromNDArray(a *ndarray.NDArray, columns []string) (*DataFrame, error) {
	if a.NDim() != 2 {
		return nil, fmt.Errorf("%w: DataFrameFromNDArray expects a 2-D array", errs.ErrShapeMismatch)
	}
	shape := a.Shape()
	if len(columns) != shape[1] {
		return nil, fmt.Errorf("%w: %d column names for %d columns", errs.ErrLengthMismatch, len(columns), shape[1])
	}
	cols := make([]*series.Series, shape[1])
	for j := range columns {
		values := make([]any, shape[0])
		for i := 0; i < shape[0]; i++ {
			v, err := a.At(i, j)
			if err != nil {
				return nil, err
			}
			values[i] = v
		}
		cols[j] = series.NewSeries(columns[j], values)
	}
	return newFrame(cols, nil)
}

func applyOptions(opts []DataFrameOption) builderOptions {
	var o builderOptions
	for _, f := range opts {
		f(&o)
	}
	return o
}

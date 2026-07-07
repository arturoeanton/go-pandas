package dataframe

import (
	"fmt"
	"math/rand"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// Row returns the row at a position as a column-name -> value map.
func (df *DataFrame) Row(pos int) (map[string]any, error) {
	if pos < 0 || pos >= df.Len() {
		return nil, fmt.Errorf("%w: row %d for frame of length %d", errs.ErrIndexOutOfBounds, pos, df.Len())
	}
	rec := make(map[string]any, len(df.columns))
	for _, c := range df.columns {
		v, err := c.At(pos)
		if err != nil {
			return nil, err
		}
		rec[c.Name()] = v
	}
	return rec, nil
}

// IRow is an alias of Row (positional access).
func (df *DataFrame) IRow(pos int) (map[string]any, error) { return df.Row(pos) }

// Head returns the first n rows.
func (df *DataFrame) Head(n int) *DataFrame {
	if n > df.Len() {
		n = df.Len()
	}
	if n < 0 {
		n = 0
	}
	out, _ := df.Slice(0, n)
	return out
}

// Tail returns the last n rows.
func (df *DataFrame) Tail(n int) *DataFrame {
	if n > df.Len() {
		n = df.Len()
	}
	if n < 0 {
		n = 0
	}
	out, _ := df.Slice(df.Len()-n, df.Len())
	return out
}

// Slice returns rows [start, stop).
func (df *DataFrame) Slice(start, stop int) (*DataFrame, error) {
	if start < 0 || stop < start || stop > df.Len() {
		return nil, fmt.Errorf("%w: slice [%d:%d] for frame of length %d", errs.ErrIndexOutOfBounds, start, stop, df.Len())
	}
	pos := make([]int, 0, stop-start)
	for i := start; i < stop; i++ {
		pos = append(pos, i)
	}
	return df.Take(pos)
}

// Take selects rows by position. The gather is fully typed (v0.4.1):
// each column buffer is gathered directly and the row index is taken
// once and shared across the result columns.
func (df *DataFrame) Take(pos []int) (*DataFrame, error) {
	idx := index.Take(df.index, pos)
	cols := make([]*series.Series, len(df.columns))
	for j, c := range df.columns {
		taken, err := c.Storage().Take(pos)
		if err != nil {
			return nil, err
		}
		cols[j] = series.Assemble(c.Name(), taken, idx)
	}
	return newFrame(cols, idx)
}

// SampleOptions configures Sample.
type SampleOptions struct {
	Seed    int64
	hasSeed bool
}

// SampleOption mutates SampleOptions.
type SampleOption func(*SampleOptions)

// WithSampleSeed makes sampling deterministic.
func WithSampleSeed(seed int64) SampleOption {
	return func(o *SampleOptions) { o.Seed = seed; o.hasSeed = true }
}

// Sample returns n rows drawn without replacement.
func (df *DataFrame) Sample(n int, opts ...SampleOption) (*DataFrame, error) {
	var o SampleOptions
	for _, f := range opts {
		f(&o)
	}
	if n < 0 || n > df.Len() {
		return nil, fmt.Errorf("%w: cannot sample %d rows from %d", errs.ErrInvalidOperation, n, df.Len())
	}
	r := rand.New(rand.NewSource(o.Seed))
	if !o.hasSeed {
		r = rand.New(rand.NewSource(rand.Int63()))
	}
	perm := r.Perm(df.Len())[:n]
	return df.Take(perm)
}

// ResetIndex returns the frame with a fresh RangeIndex. Like pandas
// reset_index, a non-default index is inserted as leading columns: one
// column per MultiIndex level (named after the level, or level_N), or a
// single column for a plain index (named after the index, or "index").
func (df *DataFrame) ResetIndex() *DataFrame {
	var cols []*series.Series
	switch ix := df.index.(type) {
	case *index.RangeIndex:
		// default index: nothing to insert
	case *index.MultiIndex:
		names := ix.Names()
		for l := 0; l < ix.NLevels(); l++ {
			name := names[l]
			if name == "" {
				name = fmt.Sprintf("level_%d", l)
			}
			values := make([]any, ix.Len())
			for i := range values {
				if ix.IsNA(i, l) {
					continue
				}
				values[i] = ix.Tuple(i)[l]
			}
			// Infer restores the typed backing per level (string/int/
			// time/...), so level dtypes round-trip.
			cols = append(cols, series.NewSeries(name, values))
		}
	default:
		name := df.index.Name()
		if name == "" {
			name = "index"
		}
		cols = append(cols, series.NewSeries(name, df.index.Values()))
	}
	for _, c := range df.columns {
		cols = append(cols, c.ResetIndex())
	}
	out, _ := newFrame(cols, index.NewRangeIndex(df.Len()))
	return out
}

// SetIndex uses column values as the new row index, removing the columns
// from the frame like df.set_index(cols). One column keeps the
// historical simple-index behavior; two or more build a MultiIndex
// (v0.8) whose level dtypes follow the column values (categorical
// columns contribute their labels).
func (df *DataFrame) SetIndex(columns ...string) (*DataFrame, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("%w: SetIndex needs a column", errs.ErrInvalidOperation)
	}
	if len(columns) > 1 {
		return df.setIndexMulti(columns)
	}
	return df.setIndexSingle(columns[0])
}

func (df *DataFrame) setIndexMulti(columns []string) (*DataFrame, error) {
	arrays := make([][]any, len(columns))
	for k, name := range columns {
		c, err := df.Col(name)
		if err != nil {
			return nil, err
		}
		arrays[k] = c.Values() // categorical columns yield labels
	}
	idx, err := index.NewMultiIndexFromArrays(arrays, columns)
	if err != nil {
		return nil, err
	}
	rest, err := df.Drop(columns...)
	if err != nil {
		return nil, err
	}
	cols := make([]*series.Series, len(rest.columns))
	for i, c := range rest.columns {
		cols[i] = c.WithIndexed(idx)
	}
	return newFrame(cols, idx)
}

func (df *DataFrame) setIndexSingle(column string) (*DataFrame, error) {
	c, err := df.Col(column)
	if err != nil {
		return nil, err
	}
	values := c.Values()
	allStrings := true
	strs := make([]string, len(values))
	for i, v := range values {
		s, ok := v.(string)
		if !ok {
			allStrings = false
			break
		}
		strs[i] = s
	}
	var idx index.Index
	if allStrings && len(values) > 0 {
		idx = index.NewStringIndex(strs, column)
	} else {
		labels := make([]string, len(values))
		for i, v := range values {
			labels[i] = fmt.Sprint(v)
		}
		idx = index.NewStringIndex(labels, column)
	}
	rest, err := df.Drop(column)
	if err != nil {
		return nil, err
	}
	cols := make([]*series.Series, len(rest.columns))
	for i, c := range rest.columns {
		cols[i] = c.WithIndexed(idx)
	}
	return newFrame(cols, idx)
}

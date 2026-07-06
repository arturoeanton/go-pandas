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

// Take selects rows by position.
func (df *DataFrame) Take(pos []int) (*DataFrame, error) {
	cols := make([]*series.Series, len(df.columns))
	for j, c := range df.columns {
		taken, err := c.Take(pos)
		if err != nil {
			return nil, err
		}
		cols[j] = taken
	}
	return newFrame(cols, index.Take(df.index, pos))
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

// ResetIndex returns the frame with a fresh RangeIndex.
func (df *DataFrame) ResetIndex() *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.ResetIndex()
	}
	out, _ := newFrame(cols, index.NewRangeIndex(df.Len()))
	return out
}

// SetIndex uses a column's values as the new row index; the column is
// removed from the frame, like df.set_index("col").
func (df *DataFrame) SetIndex(column string) (*DataFrame, error) {
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

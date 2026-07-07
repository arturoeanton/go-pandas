// Package series implements the pandas-style Series: a labeled 1-D array
// backed by typed column storage (v0.3) with a missing-value mask.
package series

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
	"github.com/arturoeanton/go-pandas/internal/display"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// Series is a labeled one-dimensional array. Values live in a typed
// column (int/float/bool/string/time backing) whenever the data is
// homogeneous; mixed data falls back to object storage.
type Series struct {
	name  string
	col   column.Column
	index index.Index
}

// Name returns the series name.
func (s *Series) Name() string { return s.name }

// Rename returns a copy with a new name.
func (s *Series) Rename(name string) *Series {
	c := s.Copy()
	c.name = name
	return c
}

// Len returns the number of elements.
func (s *Series) Len() int { return s.col.Len() }

// DType returns the element dtype.
func (s *Series) DType() dtype.DType { return s.col.DType() }

// StorageDType returns the dtype of the physical storage: identical to
// DType for typed-backed series, Object for []any-backed series.
func (s *Series) StorageDType() dtype.DType { return column.StorageDType(s.col) }

// IsObjectBacked reports whether the series stores boxed []any values
// instead of a typed column.
func (s *Series) IsObjectBacked() bool { return column.IsObjectBacked(s.col) }

// Index returns the axis labels.
func (s *Series) Index() index.Index { return s.index }

// Values returns the values with missing entries as nil.
func (s *Series) Values() []any { return s.col.Values() }

// ToList is an alias of Values, mirroring Series.tolist().
func (s *Series) ToList() []any { return s.Values() }

// ToFloat64 converts to a float64 slice; missing values become NaN.
func (s *Series) ToFloat64() ([]float64, error) {
	if fs, mask, ok := s.col.Float64s(); ok {
		out := make([]float64, len(fs))
		for i := range fs {
			if mask[i] {
				out[i] = math.NaN()
			} else {
				out[i] = fs[i]
			}
		}
		return out, nil
	}
	out := make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.col.IsNA(i) {
			out[i] = math.NaN()
			continue
		}
		f, ok := dtype.AsFloat(s.col.Value(i))
		if !ok {
			return nil, fmt.Errorf("%w: cannot convert %T at position %d to float64", errs.ErrTypeMismatch, s.col.Value(i), i)
		}
		out[i] = f
	}
	return out, nil
}

// ToNDArray converts a numeric series to a 1-D NDArray (missing -> NaN).
func (s *Series) ToNDArray() (*ndarray.NDArray, error) {
	fs, err := s.ToFloat64()
	if err != nil {
		return nil, err
	}
	return ndarray.Array(fs), nil
}

// Copy returns a deep copy of the series.
func (s *Series) Copy() *Series {
	return &Series{
		name:  s.name,
		col:   s.col.Copy(),
		index: s.index.Clone(),
	}
}

// HasNA reports whether the series contains missing values.
func (s *Series) HasNA() bool {
	for i := 0; i < s.col.Len(); i++ {
		if s.col.IsNA(i) {
			return true
		}
	}
	return false
}

// isNAAt reports whether position i holds a missing value.
func (s *Series) isNAAt(i int) bool { return s.col.IsNA(i) }

// valueAt returns the value at position i, nil when missing.
func (s *Series) valueAt(i int) any {
	if s.col.IsNA(i) {
		return nil
	}
	return s.col.Value(i)
}

// fromColumn assembles a series around an existing column.
func fromColumn(name string, col column.Column, idx index.Index) *Series {
	if idx == nil || idx.Len() != col.Len() {
		idx = index.NewRangeIndex(col.Len())
	}
	return &Series{name: name, col: col, index: idx}
}

// FormatValue renders a single cell the way pandas would (<NA>, NaT...).
func FormatValue(v any, isNA bool) string {
	if isNA {
		switch v.(type) {
		case dtype.NaTMarker:
			return "NaT"
		case float64, float32:
			return "NaN"
		default:
			return "<NA>"
		}
	}
	switch x := v.(type) {
	case float64:
		return formatFloat(x)
	case float32:
		return formatFloat(float64(x))
	case time.Time:
		return x.Format("2006-01-02 15:04:05")
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

func formatFloat(f float64) string {
	prec := display.Get().Precision
	s := strconv.FormatFloat(f, 'g', -1, 64)
	// Fall back to fixed precision only for long representations.
	if len(s) > prec+8 {
		s = strconv.FormatFloat(f, 'g', prec, 64)
	}
	return s
}

// String renders the series like pandas:
//
//	0    Ana
//	1    Luis
//	Name: name, dtype: string
func (s *Series) String() string {
	opts := display.Get()
	var b strings.Builder
	n := s.Len()
	shown := n
	truncated := false
	if n > opts.MaxRows {
		shown = opts.MaxRows
		truncated = true
	}
	labels := make([]string, shown)
	cells := make([]string, shown)
	width := 0
	for i := 0; i < shown; i++ {
		labels[i] = fmt.Sprint(s.index.At(i))
		if len(labels[i]) > width {
			width = len(labels[i])
		}
		cells[i] = FormatValue(s.col.Value(i), s.col.IsNA(i))
	}
	for i := 0; i < shown; i++ {
		b.WriteString(fmt.Sprintf("%-*s    %s\n", width, labels[i], cells[i]))
	}
	if truncated {
		b.WriteString("...\n")
	}
	b.WriteString(fmt.Sprintf("Name: %s, dtype: %s", s.name, s.DType()))
	return b.String()
}

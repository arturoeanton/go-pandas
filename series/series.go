// Package series implements the pandas-style Series: a labeled 1-D array
// with a dtype, a missing-value mask and an index.
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
	"github.com/arturoeanton/go-pandas/internal/display"
	"github.com/arturoeanton/go-pandas/ndarray"
)

// Series is a labeled one-dimensional array. mask[i] == true means the
// value at position i is missing.
type Series struct {
	name  string
	dtype dtype.DType
	data  []any
	mask  []bool
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
func (s *Series) Len() int { return len(s.data) }

// DType returns the element dtype.
func (s *Series) DType() dtype.DType { return s.dtype }

// Index returns the axis labels.
func (s *Series) Index() index.Index { return s.index }

// Values returns the values with missing entries as nil.
func (s *Series) Values() []any {
	out := make([]any, len(s.data))
	for i, v := range s.data {
		if s.mask[i] {
			out[i] = nil
		} else {
			out[i] = v
		}
	}
	return out
}

// ToList is an alias of Values, mirroring Series.tolist().
func (s *Series) ToList() []any { return s.Values() }

// ToFloat64 converts to a float64 slice; missing values become NaN.
func (s *Series) ToFloat64() ([]float64, error) {
	out := make([]float64, len(s.data))
	for i, v := range s.data {
		if s.mask[i] {
			out[i] = math.NaN()
			continue
		}
		f, ok := dtype.AsFloat(v)
		if !ok {
			return nil, fmt.Errorf("%w: cannot convert %T at position %d to float64", errs.ErrTypeMismatch, v, i)
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
		dtype: s.dtype,
		data:  append([]any(nil), s.data...),
		mask:  append([]bool(nil), s.mask...),
		index: s.index.Clone(),
	}
}

// HasNA reports whether the series contains missing values.
func (s *Series) HasNA() bool {
	for _, m := range s.mask {
		if m {
			return true
		}
	}
	return false
}

// isNAAt reports whether position i holds a missing value.
func (s *Series) isNAAt(i int) bool { return s.mask[i] }

// valueAt returns the value at position i, nil when missing.
func (s *Series) valueAt(i int) any {
	if s.mask[i] {
		return nil
	}
	return s.data[i]
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
		cells[i] = FormatValue(s.data[i], s.mask[i])
	}
	for i := 0; i < shown; i++ {
		b.WriteString(fmt.Sprintf("%-*s    %s\n", width, labels[i], cells[i]))
	}
	if truncated {
		b.WriteString("...\n")
	}
	b.WriteString(fmt.Sprintf("Name: %s, dtype: %s", s.name, s.dtype))
	return b.String()
}

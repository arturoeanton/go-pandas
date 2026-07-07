package series

import (
	"fmt"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// DatetimeOptions configures ToDatetime (v0.9).
type DatetimeOptions struct {
	// Format is a pandas/strftime-style format ("%Y-%m-%d"). Empty
	// means the deterministic inference list.
	Format string
	// Errors is "raise" (default: the first bad value errors) or
	// "coerce" (bad values become NA). pandas' "ignore" is not
	// supported (documented).
	Errors string
	// Unit interprets numeric values as unix timestamps: "s", "ms",
	// "us" or "ns". Without it numeric values are an error/NA.
	Unit string
	// UTC converts parsed values to UTC. There is no timezone dtype —
	// this only calls time.Time.UTC() on the parsed values.
	UTC bool
}

// DatetimeOption mutates DatetimeOptions.
type DatetimeOption func(*DatetimeOptions)

// WithDatetimeFormat sets an explicit pandas-style format.
func WithDatetimeFormat(format string) DatetimeOption {
	return func(o *DatetimeOptions) { o.Format = format }
}

// WithDatetimeErrors sets the error mode: "raise" (default) or "coerce".
func WithDatetimeErrors(mode string) DatetimeOption {
	return func(o *DatetimeOptions) { o.Errors = mode }
}

// WithDatetimeUnit interprets numeric values as unix timestamps in the
// given unit ("s", "ms", "us", "ns").
func WithDatetimeUnit(unit string) DatetimeOption {
	return func(o *DatetimeOptions) { o.Unit = unit }
}

// WithDatetimeUTC converts parsed values to UTC (no timezone dtype).
func WithDatetimeUTC(v bool) DatetimeOption {
	return func(o *DatetimeOptions) { o.UTC = v }
}

// ToDatetime converts a series to typed datetime storage, pandas'
// pd.to_datetime:
//
//   - explicit format via WithDatetimeFormat ("%Y-%m-%d" style);
//   - otherwise the deterministic inference list
//     (dtype.InferTimeLayouts) — day-first for the ambiguous
//     slash form, documented;
//   - nil/NA stays NA; time.Time passes through; empty and invalid
//     strings error under "raise" and become NA under "coerce";
//   - numeric values need WithDatetimeUnit.
func ToDatetime(s *Series, opts ...DatetimeOption) (*Series, error) {
	o := DatetimeOptions{Errors: "raise"}
	for _, f := range opts {
		f(&o)
	}
	switch o.Errors {
	case "raise", "coerce":
	case "ignore":
		return nil, fmt.Errorf("%w: ToDatetime errors=\"ignore\" is not supported (use raise or coerce)", errs.ErrInvalidOperation)
	default:
		return nil, fmt.Errorf("%w: unknown ToDatetime error mode %q", errs.ErrInvalidOperation, o.Errors)
	}
	var layout string
	if o.Format != "" {
		var err error
		if layout, err = dtype.TranslateTimeFormat(o.Format); err != nil {
			return nil, err
		}
	}
	coerce := o.Errors == "coerce"

	n := s.Len()
	values := make([]time.Time, n)
	mask := make([]bool, n)
	fail := func(i int, v any) error {
		if coerce {
			mask[i] = true
			return nil
		}
		return fmt.Errorf("%w: cannot parse %v (position %d) as datetime", errs.ErrTypeMismatch, v, i)
	}
	for i := 0; i < n; i++ {
		if s.col.IsNA(i) {
			mask[i] = true
			continue
		}
		v := s.col.Value(i)
		var t time.Time
		switch x := v.(type) {
		case time.Time:
			t = x
		case string:
			if x == "" {
				if err := fail(i, `""`); err != nil {
					return nil, err
				}
				continue
			}
			parsed, err := parseDatetimeString(x, layout)
			if err != nil {
				if err := fail(i, fmt.Sprintf("%q", x)); err != nil {
					return nil, err
				}
				continue
			}
			t = parsed
		default:
			if f, ok := dtype.AsFloat(v); ok {
				if _, isBool := v.(bool); !isBool && o.Unit != "" {
					parsed, err := unixTime(f, o.Unit)
					if err != nil {
						return nil, err
					}
					t = parsed
					break
				}
			}
			if err := fail(i, v); err != nil {
				return nil, err
			}
			continue
		}
		if o.UTC {
			t = t.UTC()
		}
		values[i] = t
	}
	return fromColumn(s.name, column.NewTime(values, mask), s.index.Clone()), nil
}

// ToDatetime is the method form of the package-level ToDatetime.
func (s *Series) ToDatetime(opts ...DatetimeOption) (*Series, error) {
	return ToDatetime(s, opts...)
}

// parseDatetimeString parses with the explicit layout, or walks the
// deterministic inference list.
func parseDatetimeString(s, layout string) (time.Time, error) {
	if layout != "" {
		return time.Parse(layout, s)
	}
	for _, l := range dtype.InferTimeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: no inference layout matches %q", errs.ErrTypeMismatch, s)
}

// unixTime converts a numeric timestamp in the given unit.
func unixTime(v float64, unit string) (time.Time, error) {
	switch unit {
	case "s":
		return time.Unix(int64(v), int64((v-float64(int64(v)))*1e9)).UTC(), nil
	case "ms":
		return time.UnixMilli(int64(v)).UTC(), nil
	case "us":
		return time.UnixMicro(int64(v)).UTC(), nil
	case "ns":
		return time.Unix(0, int64(v)).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: unknown datetime unit %q (use s, ms, us or ns)", errs.ErrInvalidOperation, unit)
}

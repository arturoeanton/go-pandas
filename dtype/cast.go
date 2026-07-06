package dtype

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/arturoeanton/go-pandas/errs"
)

// AsFloat converts any supported numeric (or bool) value to float64.
func AsFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// AsInt converts any supported integer-like value to int64.
func AsInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		return int64(x), true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		return int64(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case float32:
		if float32(int64(x)) == x {
			return int64(x), true
		}
	case float64:
		if float64(int64(x)) == x {
			return int64(x), true
		}
	}
	return 0, false
}

var timeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02",
	"01/02/2006",
}

// ParseTime parses a string into time.Time trying a set of common layouts.
func ParseTime(s string) (time.Time, error) {
	for _, layout := range timeFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q as datetime", errs.ErrTypeMismatch, s)
}

// CastValue converts a single value to the target dtype. Missing values
// pass through unchanged (they stay missing in whatever container holds
// them).
func CastValue(v any, target DType) (any, error) {
	if IsNA(v) {
		return v, nil
	}
	switch target {
	case Bool:
		switch x := v.(type) {
		case bool:
			return x, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(x)) {
			case "true", "t", "1", "yes":
				return true, nil
			case "false", "f", "0", "no":
				return false, nil
			}
			return nil, fmt.Errorf("%w: cannot cast %q to bool", errs.ErrTypeMismatch, x)
		default:
			if f, ok := AsFloat(v); ok {
				return f != 0, nil
			}
		}
	case Int, Int8, Int16, Int32, Int64, UInt, UInt8, UInt16, UInt32, UInt64:
		switch x := v.(type) {
		case string:
			i, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: cannot cast %q to %s", errs.ErrTypeMismatch, x, target)
			}
			return castInt(i, target), nil
		default:
			if f, ok := AsFloat(v); ok {
				return castInt(int64(math.Trunc(f)), target), nil
			}
		}
	case Float32, Float64:
		switch x := v.(type) {
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
			if err != nil {
				return nil, fmt.Errorf("%w: cannot cast %q to %s", errs.ErrTypeMismatch, x, target)
			}
			if target == Float32 {
				return float32(f), nil
			}
			return f, nil
		default:
			if f, ok := AsFloat(v); ok {
				if target == Float32 {
					return float32(f), nil
				}
				return f, nil
			}
		}
	case String:
		switch x := v.(type) {
		case string:
			return x, nil
		default:
			return fmt.Sprint(x), nil
		}
	case Time:
		switch x := v.(type) {
		case time.Time:
			return x, nil
		case string:
			return ParseTime(x)
		}
	case Object:
		return v, nil
	}
	return nil, fmt.Errorf("%w: cannot cast %T to %s", errs.ErrTypeMismatch, v, target)
}

func castInt(i int64, target DType) any {
	switch target {
	case Int:
		return int(i)
	case Int8:
		return int8(i)
	case Int16:
		return int16(i)
	case Int32:
		return int32(i)
	case Int64:
		return i
	case UInt:
		return uint(i)
	case UInt8:
		return uint8(i)
	case UInt16:
		return uint16(i)
	case UInt32:
		return uint32(i)
	case UInt64:
		return uint64(i)
	}
	return i
}

// CastSlice converts every value of the slice to the target dtype.
func CastSlice(values []any, target DType) ([]any, error) {
	out := make([]any, len(values))
	for i, v := range values {
		c, err := CastValue(v, target)
		if err != nil {
			return nil, fmt.Errorf("at position %d: %w", i, err)
		}
		out[i] = c
	}
	return out, nil
}

// CanCast reports whether a cast from one dtype to another is generally
// allowed (it may still fail per-value, e.g. String -> Int on "abc").
func CanCast(from, to DType) bool {
	if from == to || to == Object || to == String {
		return true
	}
	switch {
	case IsNumeric(from) || IsBool(from):
		return IsNumeric(to) || IsBool(to)
	case IsString(from):
		return IsNumeric(to) || IsBool(to) || IsDatetime(to)
	case IsDatetime(from):
		return to == Time
	case from == Object:
		return true
	}
	return false
}

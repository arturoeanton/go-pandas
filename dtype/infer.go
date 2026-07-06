package dtype

import "time"

// kind buckets used during inference.
type inferKind int

const (
	kindNone inferKind = iota
	kindBool
	kindInt
	kindInt64
	kindUint
	kindFloat
	kindString
	kindTime
	kindOther
)

func kindOf(v any) inferKind {
	switch v.(type) {
	case bool:
		return kindBool
	case int, int8, int16, int32:
		return kindInt
	case int64:
		return kindInt64
	case uint, uint8, uint16, uint32, uint64:
		return kindUint
	case float32, float64:
		return kindFloat
	case string:
		return kindString
	case time.Time:
		return kindTime
	}
	return kindOther
}

// InferDType infers the dtype of a slice of values following the pandas
// rules: missing values are skipped, mixed int/float becomes Float64,
// incompatible mixes fall back to Object and an all-NA slice is Object.
func InferDType(values []any) DType {
	var sawBool, sawInt, sawInt64, sawUint, sawFloat, sawString, sawTime, sawOther bool
	sawAny := false
	for _, v := range values {
		if IsNA(v) {
			continue
		}
		sawAny = true
		switch kindOf(v) {
		case kindBool:
			sawBool = true
		case kindInt:
			sawInt = true
		case kindInt64:
			sawInt64 = true
		case kindUint:
			sawUint = true
		case kindFloat:
			sawFloat = true
		case kindString:
			sawString = true
		case kindTime:
			sawTime = true
		default:
			sawOther = true
		}
	}
	if !sawAny {
		return Object
	}
	if sawOther {
		return Object
	}
	numeric := sawInt || sawInt64 || sawUint || sawFloat
	switch {
	case sawString && !numeric && !sawBool && !sawTime:
		return String
	case sawTime && !numeric && !sawBool && !sawString:
		return Time
	case sawBool && !numeric && !sawString && !sawTime:
		return Bool
	case numeric && !sawString && !sawBool && !sawTime:
		if sawFloat {
			return Float64
		}
		if sawInt64 || sawUint {
			return Int64
		}
		return Int
	}
	return Object
}

// InferDTypeStrict is like InferDType but refuses any mixing: every non-NA
// value must belong to the same kind, otherwise the result is Object.
// Mixed int/float, which InferDType promotes to Float64, is Object here.
func InferDTypeStrict(values []any) DType {
	first := kindNone
	sawAny := false
	for _, v := range values {
		if IsNA(v) {
			continue
		}
		k := kindOf(v)
		if !sawAny {
			first = k
			sawAny = true
			continue
		}
		if k != first {
			return Object
		}
	}
	if !sawAny {
		return Object
	}
	switch first {
	case kindBool:
		return Bool
	case kindInt:
		return Int
	case kindInt64:
		return Int64
	case kindUint:
		return UInt64
	case kindFloat:
		return Float64
	case kindString:
		return String
	case kindTime:
		return Time
	}
	return Object
}

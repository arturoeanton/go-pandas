package dtype

import (
	"fmt"
	"strings"

	"github.com/arturoeanton/go-pandas/errs"
)

// Number is a pseudo-dtype usable in SelectDTypes-style filters: it
// matches every numeric dtype, like pandas' include=["number"].
const Number DType = -1

// Kind buckets dtypes the way NumPy kinds do ('b', 'i', 'u', 'f', ...).
type DTypeKind int

const (
	KindInvalid DTypeKind = iota
	KindBool
	KindSignedInt
	KindUnsignedInt
	KindFloat
	KindComplex
	KindString
	KindBytes
	KindDatetime
	KindTimedelta
	KindCategory
	KindObject
)

// Kind returns the kind bucket of a dtype.
func (t DType) Kind() DTypeKind {
	switch {
	case t == Bool:
		return KindBool
	case isSignedInt(t):
		return KindSignedInt
	case isUnsignedInt(t):
		return KindUnsignedInt
	case IsFloat(t):
		return KindFloat
	case t == Complex64 || t == Complex128:
		return KindComplex
	case t == String:
		return KindString
	case t == Bytes:
		return KindBytes
	case t == Time:
		return KindDatetime
	case t == Timedelta:
		return KindTimedelta
	case t == Category:
		return KindCategory
	case t == Object:
		return KindObject
	}
	return KindInvalid
}

// parseAliases maps pandas/NumPy dtype spellings to go-pandas dtypes.
var parseAliases = map[string]DType{
	"bool":            Bool,
	"boolean":         Bool,
	"int":             Int,
	"int8":            Int8,
	"int16":           Int16,
	"int32":           Int32,
	"int64":           Int64,
	"uint":            UInt,
	"uint8":           UInt8,
	"uint16":          UInt16,
	"uint32":          UInt32,
	"uint64":          UInt64,
	"float":           Float64,
	"float32":         Float32,
	"float64":         Float64,
	"double":          Float64,
	"complex64":       Complex64,
	"complex128":      Complex128,
	"str":             String,
	"string":          String,
	"bytes":           Bytes,
	"datetime":        Time,
	"datetime64":      Time,
	"datetime64[ns]":  Time,
	"datetime64[us]":  Time,
	"datetime64[ms]":  Time,
	"datetime64[s]":   Time,
	"timedelta":       Timedelta,
	"timedelta64":     Timedelta,
	"timedelta64[ns]": Timedelta,
	"category":        Category,
	"object":          Object,
	"o":               Object,
	"number":          Number,
}

// ParseDType parses a pandas/NumPy-style dtype name:
//
//	ParseDType("int64")
//	ParseDType("float64")
//	ParseDType("datetime64[ns]")
//	ParseDType("category")
func ParseDType(name string) (DType, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	// pandas nullable spellings ("Int64", "Float64", "boolean") map to the
	// same dtype: every go-pandas series carries a missing mask already.
	if t, ok := parseAliases[key]; ok {
		return t, nil
	}
	return Invalid, fmt.Errorf("%w: unknown dtype %q", errs.ErrInvalidDType, name)
}

// Matches reports whether a concrete dtype satisfies a selector dtype
// (which may be the Number pseudo-dtype).
func Matches(selector, concrete DType) bool {
	if selector == Number {
		return IsNumeric(concrete) || IsBool(concrete)
	}
	return selector == concrete
}

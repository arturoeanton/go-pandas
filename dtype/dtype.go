// Package dtype implements the go-pandas data type system, inspired by
// pandas/NumPy dtypes: type constants, inference, casting, promotion and
// the missing value model (NA / NaT).
package dtype

// DType identifies the logical element type of a Series or NDArray column.
type DType int

const (
	Invalid DType = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	UInt
	UInt8
	UInt16
	UInt32
	UInt64
	Float32
	Float64
	Complex64
	Complex128
	String
	Bytes
	Time
	Timedelta
	Category
	Object
)

var dtypeNames = map[DType]string{
	Invalid:    "invalid",
	Bool:       "bool",
	Int:        "int",
	Int8:       "int8",
	Int16:      "int16",
	Int32:      "int32",
	Int64:      "int64",
	UInt:       "uint",
	UInt8:      "uint8",
	UInt16:     "uint16",
	UInt32:     "uint32",
	UInt64:     "uint64",
	Float32:    "float32",
	Float64:    "float64",
	Complex64:  "complex64",
	Complex128: "complex128",
	String:     "string",
	Bytes:      "bytes",
	Time:       "datetime64",
	Timedelta:  "timedelta64",
	Category:   "category",
	Object:     "object",
}

func (t DType) String() string {
	if s, ok := dtypeNames[t]; ok {
		return s
	}
	return "unknown"
}

// NullableDType marks a dtype as explicitly nullable (pandas extension
// dtypes such as Int64 with pd.NA). In v0.1 every Series carries a missing
// mask, so this is a semantic wrapper reserved for future typed storage.
type NullableDType struct {
	Base DType
}

func (n NullableDType) String() string { return n.Base.String() + "?" }

// IsNumeric reports whether t is a numeric dtype (integers, unsigned
// integers, floats or complex).
func IsNumeric(t DType) bool {
	return IsInteger(t) || IsFloat(t) || t == Complex64 || t == Complex128
}

func IsInteger(t DType) bool {
	switch t {
	case Int, Int8, Int16, Int32, Int64, UInt, UInt8, UInt16, UInt32, UInt64:
		return true
	}
	return false
}

func IsFloat(t DType) bool    { return t == Float32 || t == Float64 }
func IsString(t DType) bool   { return t == String }
func IsBool(t DType) bool     { return t == Bool }
func IsDatetime(t DType) bool { return t == Time }

func isSignedInt(t DType) bool {
	switch t {
	case Int, Int8, Int16, Int32, Int64:
		return true
	}
	return false
}

func isUnsignedInt(t DType) bool {
	switch t {
	case UInt, UInt8, UInt16, UInt32, UInt64:
		return true
	}
	return false
}

package dtype

// intRank orders integer widths for promotion. Plain Int/UInt are treated
// as 64-bit (Go's int is 64-bit on all supported platforms).
func intRank(t DType) int {
	switch t {
	case Int8, UInt8:
		return 8
	case Int16, UInt16:
		return 16
	case Int32, UInt32:
		return 32
	case Int, UInt, Int64, UInt64:
		return 64
	}
	return 0
}

// Promote returns the common dtype able to represent values of both a and
// b, following NumPy-style promotion simplified for v0.1.
func Promote(a, b DType) DType {
	if a == b {
		return a
	}
	if a == Invalid {
		return b
	}
	if b == Invalid {
		return a
	}
	if a == Object || b == Object {
		return Object
	}
	// Bool promotes to the other numeric type.
	if a == Bool && IsNumeric(b) {
		return b
	}
	if b == Bool && IsNumeric(a) {
		return a
	}
	if IsNumeric(a) && IsNumeric(b) {
		if a == Complex128 || b == Complex128 || a == Complex64 || b == Complex64 {
			return Complex128
		}
		if IsFloat(a) || IsFloat(b) {
			if a == Float32 && b == Float32 {
				return Float32
			}
			// float32 + int wider than 16 bits, or any float64 -> float64
			return Float64
		}
		// both integers
		switch {
		case isSignedInt(a) && isSignedInt(b):
			if intRank(a) >= intRank(b) {
				return widerSigned(a, b)
			}
			return widerSigned(b, a)
		case isUnsignedInt(a) && isUnsignedInt(b):
			if intRank(a) >= intRank(b) {
				return a
			}
			return b
		default:
			// mixed signed/unsigned -> signed 64-bit
			return Int64
		}
	}
	if a == String || b == String {
		return Object
	}
	if a == Time || b == Time {
		return Object
	}
	return Object
}

func widerSigned(wider, narrower DType) DType {
	// Int and Int64 have the same rank; prefer Int64 when they mix.
	if (wider == Int && narrower == Int64) || (wider == Int64 && narrower == Int) {
		return Int64
	}
	return wider
}

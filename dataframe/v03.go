package dataframe

import "github.com/arturoeanton/go-pandas/dtype"

// StorageDTypes maps every column to its physical storage dtype: equal
// to DTypes() for typed-backed columns, Object for []any-backed ones.
// Use it to verify that data landed in typed storage (v0.3).
func (df *DataFrame) StorageDTypes() map[string]dtype.DType {
	out := make(map[string]dtype.DType, len(df.columns))
	for _, c := range df.columns {
		out[c.Name()] = c.StorageDType()
	}
	return out
}

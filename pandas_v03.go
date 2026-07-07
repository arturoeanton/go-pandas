package pandas

import (
	"github.com/arturoeanton/go-pandas/ndarray"
)

// v0.3 typed-storage surface.

// DebugPlan reports which execution path an expression takes on a frame:
// "columnar" (v0.4 typed kernels) or "row-fallback". For debugging and
// tests:
//
//	fmt.Println(pd.DebugPlan(df, pd.Col("age").Gt(30)))
func DebugPlan(df *DataFrame, e Expr) string {
	return df.Plan(e).String()
}

// ArrayString builds a 1-D array backed by []string. String arrays
// support comparisons, Sort, Unique and Astype; arithmetic returns
// errors.
func ArrayString(data []string) *NDArray { return ndarray.ArrayString(data) }

// FromSliceTyped builds an array with an explicit shape from any
// supported element slice, keeping the element type in storage:
//
//	m, _ := pd.FromSliceTyped([]int{1, 2, 3, 4}, 2, 2) // int backing
//
// (pd.FromSlice keeps its float64 signature for compatibility.)
func FromSliceTyped[T ndarray.Element](data []T, shape ...int) (*NDArray, error) {
	return ndarray.FromSlice(data, shape...)
}

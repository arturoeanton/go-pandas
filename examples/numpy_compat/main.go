// Example numpy_compat: NumPy-style arrays with broadcasting, reductions
// and linear algebra.
package main

import (
	"fmt"

	pd "github.com/arturoeanton/go-pandas"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// a = np.arange(6).reshape(2, 3)
	a, err := pd.Arange(6).Reshape(2, 3)
	check(err)
	fmt.Println("a =", a)

	// b = np.array([10, 20, 30])
	b := pd.Array([]float64{10, 20, 30})
	fmt.Println("b =", b)

	// broadcasting: (2,3) + (3,)
	c, err := a.Add(b)
	check(err)
	fmt.Println("a + b =", c)

	// scalar math
	fmt.Println("b * 2 =", b.MulScalar(2))
	fmt.Println("sqrt([1 4 9]) =", pd.Array([]float64{1, 4, 9}).Sqrt())

	// reductions
	fmt.Println("a.sum() =", a.SumAll())
	fmt.Println("a.mean() =", a.MeanAll())
	sum0, err := a.Sum(0)
	check(err)
	fmt.Println("a.sum(axis=0) =", sum0)

	// linear algebra
	m, err := pd.FromSlice([]float64{1, 2, 3, 4}, 2, 2)
	check(err)
	mm, err := pd.MatMul(m, m)
	check(err)
	fmt.Println("m @ m =", mm)
	dot, err := pd.Dot(pd.Array([]float64{1, 2, 3}), pd.Array([]float64{4, 5, 6}))
	check(err)
	fmt.Println("dot([1 2 3], [4 5 6]) =", dot)

	// slicing returns views
	view, err := a.Slice(pd.All(), pd.Slice(1, 3))
	check(err)
	fmt.Println("a[:, 1:3] =", view)

	// incompatible broadcast is an explicit error
	if _, err := pd.Array([]float64{1, 2, 3}).Add(pd.Zeros(4)); err != nil {
		fmt.Println("(3,) + (4,) ->", err)
	}
}

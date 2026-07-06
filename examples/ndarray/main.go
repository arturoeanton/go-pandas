// Example ndarray: shapes, views, slicing and mutation semantics.
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
	a, err := pd.Arange(12).Reshape(3, 4)
	check(err)
	fmt.Println("a =", a)
	fmt.Println("shape:", a.Shape(), "strides:", a.Strides(), "size:", a.Size())

	// slicing returns a view: mutating it mutates the base array
	view, err := a.Slice(pd.Slice(0, 2), pd.Slice(1, 3))
	check(err)
	fmt.Println("view a[0:2, 1:3] =", view)
	check(view.Set(99, 0, 0))
	fmt.Println("after view.Set(99, 0, 0), a =", a)

	// Copy is independent
	c := a.Copy()
	check(c.Set(-1, 0, 0))
	fmt.Println("copies are independent:", a.MustAt(0, 0), "vs", c.MustAt(0, 0))

	// transpose is a view too
	tr, err := a.T()
	check(err)
	fmt.Println("a.T shape:", tr.Shape())

	// elementwise pipeline
	normalized := a.SubScalar(a.MeanAll()).DivScalar(a.StdAll())
	fmt.Println("normalized mean ~ 0:", normalized.MeanAll())

	// random arrays
	r := pd.Randn(2, 2)
	fmt.Println("randn(2,2) =", r)
}

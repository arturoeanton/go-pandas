// Example merge: pandas-style joins between frames.
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
	left, err := pd.DataFrameFromRecords([]map[string]any{
		{"id": 1, "name": "Ana"},
		{"id": 2, "name": "Luis"},
		{"id": 3, "name": "Marta"},
	}, pd.WithColumnOrder("id", "name"))
	check(err)

	right, err := pd.DataFrameFromRecords([]map[string]any{
		{"id": 1, "salary": 1000.0},
		{"id": 2, "salary": 2000.0},
		{"id": 4, "salary": 4000.0},
	}, pd.WithColumnOrder("id", "salary"))
	check(err)

	inner, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "inner"})
	check(err)
	fmt.Println("inner join:")
	fmt.Println(inner)
	fmt.Println()

	leftJoin, err := left.Merge(right, pd.MergeOptions{On: []string{"id"}, How: "left"})
	check(err)
	fmt.Println("left join (unmatched -> <NA>):")
	fmt.Println(leftJoin)
	fmt.Println()

	outer, err := pd.Merge(left, right, pd.MergeOptions{On: []string{"id"}, How: "outer", Indicator: true})
	check(err)
	fmt.Println("outer join with indicator:")
	fmt.Println(outer)
	fmt.Println()

	frames := []*pd.DataFrame{left.Head(2), left.Tail(1)}
	stacked, err := pd.Concat(frames, pd.ConcatIgnoreIndex(true))
	check(err)
	fmt.Println("concat:")
	fmt.Println(stacked)
}

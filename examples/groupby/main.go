// Example groupby: pandas-style grouped aggregations.
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
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
		{"country": "BR", "name": "Bia", "age": 28, "salary": 1200.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
	check(err)

	// df.groupby("country")["salary"].mean()
	mean, err := df.GroupBy("country").Mean("salary")
	check(err)
	fmt.Println("mean salary by country:")
	fmt.Println(mean)
	fmt.Println()

	// df.groupby("country").size()
	size, err := df.GroupBy("country").Size()
	check(err)
	fmt.Println("group sizes:")
	fmt.Println(size)
	fmt.Println()

	// df.groupby("country").agg({"salary": "mean", "age": "max"})
	grouped, err := df.GroupBy("country").Agg(map[string]string{
		"salary": "mean",
		"age":    "max",
	})
	check(err)
	fmt.Println("agg salary:mean, age:max:")
	fmt.Println(grouped)
	fmt.Println()

	// multiple aggregations per column
	multi, err := df.GroupBy("country").AggList(map[string][]string{
		"salary": {"min", "max"},
	})
	check(err)
	fmt.Println("agg salary:[min max]:")
	fmt.Println(multi)
}

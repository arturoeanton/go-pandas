// Example basic: the canonical pandas-like workflow — build a frame,
// filter, select, sort and print.
package main

import (
	"fmt"

	pd "github.com/arturoeanton/go-pandas"
)

func main() {
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
	if err != nil {
		panic(err)
	}

	fmt.Println("full frame:")
	fmt.Println(df)
	fmt.Println()

	result, err := df.Where(pd.Col("age").Gt(30))
	if err != nil {
		panic(err)
	}

	result, err = result.Select("country", "name", "salary")
	if err != nil {
		panic(err)
	}

	result, err = result.SortValues("salary", false)
	if err != nil {
		panic(err)
	}

	fmt.Println("age > 30, sorted by salary desc:")
	fmt.Println(result)
}

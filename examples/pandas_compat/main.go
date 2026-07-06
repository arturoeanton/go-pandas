// Example pandas_compat: side-by-side translations of common pandas
// idioms into go-pandas.
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
		{"country": "BR", "name": "Bia", "age": nil, "salary": 900.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
	check(err)

	// df.head(2)
	fmt.Println("# df.head(2)")
	fmt.Println(df.Head(2))
	fmt.Println()

	// df[["name", "age"]]
	sel, err := df.Select("name", "age")
	check(err)
	fmt.Println("# df[['name', 'age']]")
	fmt.Println(sel)
	fmt.Println()

	// df[df["age"] > 30]
	adults, err := df.Where(pd.Col("age").Gt(30))
	check(err)
	fmt.Println("# df[df['age'] > 30]")
	fmt.Println(adults)
	fmt.Println()

	// df["total"] = df["salary"] * 12
	yearly, err := df.AssignExpr("yearly", pd.Col("salary").Mul(12))
	check(err)
	fmt.Println("# df.assign(yearly=df['salary'] * 12)")
	fmt.Println(yearly)
	fmt.Println()

	// df.query("age >= 30 and country == 'AR'")
	q, err := df.Query(`age >= 30 and country == "AR"`)
	check(err)
	fmt.Println("# df.query('age >= 30 and country == \"AR\"')")
	fmt.Println(q)
	fmt.Println()

	// df.isna() / df.fillna(...) / df.dropna()
	fmt.Println("# df.dropna()")
	fmt.Println(df.DropNA())
	fmt.Println()
	filled, err := df.FillNA(map[string]any{"age": 0})
	check(err)
	fmt.Println("# df.fillna({'age': 0})")
	fmt.Println(filled)
	fmt.Println()

	// df.describe()
	fmt.Println("# df.describe()")
	fmt.Println(df.Describe())
	fmt.Println()

	// Series string accessor: df["name"].str.upper()
	names, err := df.Col("name")
	check(err)
	fmt.Println("# df['name'].str.upper()")
	fmt.Println(names.Str().Upper())
}

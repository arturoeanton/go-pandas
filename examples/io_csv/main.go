// Example io_csv: write a frame to CSV, read it back with type inference
// and NA handling.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	pd "github.com/arturoeanton/go-pandas"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	dir, err := os.MkdirTemp("", "go-pandas-example")
	check(err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "people.csv")

	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
		{"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
		{"country": "BR", "name": "Joao", "age": nil, "salary": 1500.0},
	}, pd.WithColumnOrder("country", "name", "age", "salary"))
	check(err)

	// df.to_csv(path)
	check(df.ToCSV(path))
	raw, err := os.ReadFile(path)
	check(err)
	fmt.Println("CSV on disk:")
	fmt.Println(string(raw))

	// pd.read_csv(path) — types are inferred, empty cells become NA
	back, err := pd.ReadCSV(path)
	check(err)
	fmt.Println("read back:")
	fmt.Println(back)
	fmt.Println()
	fmt.Println("dtypes:", back.DTypes())

	// JSON round trip too
	jsonPath := filepath.Join(dir, "people.json")
	check(df.ToJSON(jsonPath))
	fromJSON, err := pd.ReadJSON(jsonPath)
	check(err)
	rows, cols := fromJSON.Shape()
	fmt.Printf("JSON round trip: %d rows x %d columns\n", rows, cols)
}

// Example rolling: moving window statistics over Series and DataFrames.
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
	prices := pd.FloatSeries("price", []float64{10, 11, 12, 11, 13, 15, 14, 16})

	// s.rolling(3).mean()
	ma, err := prices.Rolling(3).Mean()
	check(err)
	fmt.Println("rolling(3).mean():")
	fmt.Println(ma)
	fmt.Println()

	// min_periods=1 fills the warm-up windows
	ma1, err := prices.Rolling(3, pd.RollingMinPeriods(1)).Mean()
	check(err)
	fmt.Println("rolling(3, min_periods=1).mean():")
	fmt.Println(ma1)
	fmt.Println()

	// expanding().mean()
	exp, err := prices.Expanding().Mean()
	check(err)
	fmt.Println("expanding().mean():")
	fmt.Println(exp)
	fmt.Println()

	// DataFrame rolling applies to every numeric column
	df, err := pd.DataFrameFromMap(map[string][]any{
		"open":  {1.0, 2.0, 3.0, 4.0},
		"close": {2.0, 3.0, 4.0, 5.0},
	})
	check(err)
	rolled, err := df.Rolling(2).Sum()
	check(err)
	fmt.Println("df.rolling(2).sum():")
	fmt.Println(rolled)
}

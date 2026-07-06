package dataframe

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/series"
)

// Filter keeps the rows where the boolean mask series is true (the pandas
// df[mask]). Missing mask entries drop the row.
func (df *DataFrame) Filter(mask *series.Series) (*DataFrame, error) {
	if mask.Len() != df.Len() {
		return nil, fmt.Errorf("%w: mask of length %d for frame of length %d", errs.ErrLengthMismatch, mask.Len(), df.Len())
	}
	bools := mask.AsMask()
	var pos []int
	for i, keep := range bools {
		if keep {
			pos = append(pos, i)
		}
	}
	return df.Take(pos)
}

// Where keeps the rows matching a predicate (the pandas df[df.x > 1]):
//
//	df.Where(pd.Col("age").Gt(30))
func (df *DataFrame) Where(pred expr.Predicate) (*DataFrame, error) {
	records := df.ToRecords()
	var pos []int
	for i, rec := range records {
		ok, err := pred.EvalBool(rec)
		if err != nil {
			return nil, fmt.Errorf("evaluating %s at row %d: %w", pred, i, err)
		}
		if ok {
			pos = append(pos, i)
		}
	}
	return df.Take(pos)
}

// Query filters rows with a small pandas-like query language:
//
//	df.Query(`age >= 30 and salary < 2000`)
//	df.Query(`country in ["AR", "BR"]`)
func (df *DataFrame) Query(q string) (*DataFrame, error) {
	pred, err := expr.ParseQuery(q)
	if err != nil {
		return nil, err
	}
	return df.Where(pred)
}

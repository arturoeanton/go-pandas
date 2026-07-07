package pandas

import (
	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/series"
)

// v0.10 reshape/groupby re-exports.
type (
	// GroupCond is a group-level filter condition for GroupBy.Filter.
	GroupCond = dataframe.GroupCond
	// GroupMetric builds group filter conditions.
	GroupMetric = dataframe.GroupMetric
)

// GroupSize filters groups on their row count:
// df.GroupBy("k").Filter(pd.GroupSize().Gt(2)).
func GroupSize() GroupMetric { return dataframe.GroupSize() }

// GroupCount filters groups on a column's non-NA count:
// df.GroupBy("k").Filter(pd.GroupCount("salary").Ge(3)).
func GroupCount(column string) GroupMetric { return dataframe.GroupCount(column) }

// UnstackSeries pivots the last MultiIndex level of a series into
// DataFrame columns, pandas' s.unstack().
func UnstackSeries(s *series.Series) (*DataFrame, error) {
	return dataframe.UnstackSeries(s)
}

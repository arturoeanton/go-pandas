package pandas

import (
	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/index"
)

// v0.8 MultiIndex re-exports.
type (
	// Tuple is one hierarchical index label (one component per level;
	// nil = NA): pd.Tuple{"AR", "Buenos Aires"}.
	Tuple = index.Tuple
)

// MultiIndexFromArrays builds a hierarchical index from parallel Series,
// one per level. Level lists are the sorted unique labels (pandas
// parity); NA values become code -1.
func MultiIndexFromArrays(names []string, arrays ...*Series) (*MultiIndex, error) {
	boxed := make([][]any, len(arrays))
	for i, s := range arrays {
		boxed[i] = s.Values()
	}
	return index.NewMultiIndexFromArrays(boxed, names)
}

// MultiIndexFromTuples builds a hierarchical index from row tuples.
func MultiIndexFromTuples(names []string, tuples []Tuple) (*MultiIndex, error) {
	boxed := make([][]any, len(tuples))
	for i, t := range tuples {
		boxed[i] = t
	}
	return index.NewMultiIndexFromTuples(boxed, names)
}

// GroupAsIndex makes multi-key groupby results carry the group keys as
// a MultiIndex (single keys as a regular index) instead of key columns,
// like pandas groupby(as_index=True). go-pandas defaults to as_index
// =false (keys stay columns) — the historical behavior.
func GroupAsIndex(v bool) GroupByOption { return dataframe.GroupAsIndex(v) }

# go-pandas compatibility report

The numbers below are computed **directly from the matrix rows** in
`pandas_matrix.md` / `numpy_matrix.md`: `done` and `partial` rows count as
implemented; `planned` and `not_supported` do not. Reproduce them with:

```bash
go run ./cmd/compat-report
```

Behavioral verification: 200+ golden test cases generated from real
pandas 2.3.3 / NumPy 2.0.2 (`compat/goldens/`), all passing. A matrix row
can group several closely-related pandas/NumPy APIs, so these counts are
rows tracked, not individual Python functions.

## pandas compatibility (98 rows tracked, 91 implemented, 93%)

| Area | Rows tracked | Implemented | Coverage |
|---|---:|---:|---:|
| Constructors and core attributes | 9 | 9 | 100% |
| Selection and indexing | 13 | 12 | 92% |
| Mutation and transforms | 15 | 15 | 100% |
| Missing values | 5 | 5 | 100% |
| Series | 15 | 15 | 100% |
| String and datetime accessors | 11 | 10 | 90% |
| GroupBy | 7 | 6 | 85% |
| Merge / join / concat | 5 | 5 | 100% |
| Reshape and window | 9 | 6 | 66% |
| IO | 9 | 8 | 88% |

Not implemented in the pandas area: MultiIndex operations, Categorical,
timezone handling (`tz_localize`/`tz_convert`), resample, stack/unstack,
eval, ewm, groupby transform/filter, Parquet/Excel/SQL IO.

Note that "implemented" includes `partial` rows — 17 of the 91 pandas
rows are partial (e.g. Query grammar subset, single-column SetIndex,
PivotTable with one aggregation). See the matrix notes for each.

## NumPy compatibility (53 rows tracked, 47 implemented, 89%)

| Area | Rows tracked | Implemented | Coverage |
|---|---:|---:|---:|
| Constructors | 7 | 7 | 100% |
| Shape, views and joining | 9 | 9 | 100% |
| Indexing | 8 | 7 | 87% |
| Math | 9 | 9 | 100% |
| Reductions | 6 | 5 | 83% |
| Sorting and set operations | 4 | 3 | 75% |
| Linear algebra | 5 | 3 | 60% |
| Random | 5 | 4 | 80% |

Not implemented in the NumPy area: det/inv/solve/eig/SVD (planned via the
gonum adapter), keepdims/axis tuples, fancy integer indexing, negative
slice steps, searchsorted/isin, random distributions beyond
rand/randn/randint.

**Storage (v0.3):** NDArrays and Series columns store real typed
backings (bool/int/int64/float32/float64/string, plus time for Series);
`[]any` object storage remains only for mixed values. Verified by 19
dtype golden cases against real pandas/NumPy plus typed-storage
acceptance tests. See [known_differences.md](known_differences.md).

## How to update

Edit the matrices, then run `go run ./cmd/compat-report` and refresh the
tables above. Do not edit the numbers here by hand without re-running the
tool.

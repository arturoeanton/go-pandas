# go-pandas compatibility report

The numbers below are computed **directly from the matrix rows** in
`pandas_matrix.md` / `numpy_matrix.md`: `done` and `partial` rows count as
implemented; `planned` and `not_supported` do not. Reproduce them with:

```bash
go run ./cmd/compat-report
```

Behavioral verification: 295 golden test cases generated from real
pandas 2.3.3 / NumPy 2.0.2 (`compat/goldens/`), all passing. A matrix row
can group several closely-related pandas/NumPy APIs, so these counts are
rows tracked, not individual Python functions.

## pandas compatibility (134 rows tracked, 125 implemented, 93%)

| Area | Rows tracked | Implemented | Coverage |
|---|---:|---:|---:|
| Constructors and core attributes | 9 | 9 | 100% |
| Selection and indexing (incl. MultiIndex, v0.8) | 23 | 20 | 86% |
| Mutation and transforms | 15 | 15 | 100% |
| Missing values | 5 | 5 | 100% |
| Series (incl. categorical v0.7, to_datetime v0.9) | 26 | 26 | 100% |
| String and datetime accessors | 11 | 10 | 90% |
| GroupBy (incl. transform/filter, v0.10) | 8 | 8 | 100% |
| Merge / join / concat | 7 | 7 | 100% |
| Reshape and window (incl. stack/unstack + query, v0.10) | 20 | 16 | 80% |
| IO | 10 | 9 | 90% |

The tracked surface keeps growing with each phase (v0.8 MultiIndex,
v0.9 time series, including planned rows like resample options and
timezone operations) — coverage is counted honestly against the larger
surface, which is why the headline percentage can drop while features
land.

Not implemented in the pandas area: timezone handling
(`tz_localize`/`tz_convert`), resample closed/label/origin options and
MultiIndex-level resample, MultiIndex level operations
(swaplevel/droplevel/xs, merge on levels, label-range slicing), eval,
ewm, Parquet/Excel/SQL IO.

Note that "implemented" includes `partial` rows — 20 of the 125 pandas
rows are partial (e.g. Query grammar subset, tuple-prefix-only
MultiIndex selection, observed-buckets resample, last-level-only
unstack). See the matrix notes for each.

## NumPy compatibility (54 rows tracked, 49 implemented, 91%)

| Area | Rows tracked | Implemented | Coverage |
|---|---:|---:|---:|
| Constructors | 7 | 7 | 100% |
| Shape, views and joining | 9 | 9 | 100% |
| Indexing | 8 | 7 | 87% |
| Math | 9 | 9 | 100% |
| Reductions | 6 | 5 | 83% |
| Sorting and set operations (incl. isin/searchsorted, v0.10) | 5 | 5 | 100% |
| Linear algebra | 5 | 3 | 60% |
| Random | 5 | 4 | 80% |

Not implemented in the NumPy area: det/inv/solve/eig/SVD (planned via the
gonum adapter), keepdims/axis tuples, fancy integer indexing, negative
slice steps, random distributions beyond rand/randn/randint.

**Reshape + query closure (v0.10):** Stack/Unstack over MultiIndex,
pivot_table with multiple values/aggfuncs on the typed groupby engine,
GroupBy Transform (broadcast typed gather) and Filter (size/count
conditions), query grammar with arithmetic/parentheses/not in/datetime
strings, np.isin/np.searchsorted. 17-case golden addition plus 8 fuzz
targets.

**Time series (v0.9):** format-aware ToDatetime (strftime directives,
raise/coerce, deterministic inference, unix units), DatetimeIndex with
NA mask and typed gather, and an observed-buckets Resample engine
(H/D/W/MS/ME; sum/mean/count/min/max/first/last) reusing the typed
groupby reducers. 9-case golden suite plus roundtrip/invariant fuzzing.

**MultiIndex (v0.8):** real levels+codes storage with sorted unique
levels and NA components as code -1 (pandas parity, golden-verified).
Multi-column SetIndex, ResetIndex level expansion, tuple Loc (full via
lookup map, prefix via scan), optional groupby as_index, typed Take
preservation through every engine. 8-case golden suite plus equivalence
fuzzing.

**Categorical (v0.7):** `pd.Category` stores int32 codes into a shared
category list; Astype both ways, Cat() accessor, rank-based ordered
comparisons and code fast paths in groupby/merge/sort/value_counts/
concat/expressions. Verified by a 12-case golden suite against pandas
2.3.3 plus string-vs-categorical equivalence fuzzing.

**Expressions (v0.4):** Where/AssignExpr/Query execute on the columnar
engine for typed columns (10 expression golden cases mirror pandas
boolean indexing, assign and query); the row evaluator remains the
verified fallback.

**Storage (v0.3):** NDArrays and Series columns store real typed
backings (bool/int/int64/float32/float64/string, plus time for Series);
`[]any` object storage remains only for mixed values. Verified by 19
dtype golden cases against real pandas/NumPy plus typed-storage
acceptance tests. See [known_differences.md](known_differences.md).

## How to update

Edit the matrices, then run `go run ./cmd/compat-report` and refresh the
tables above. Do not edit the numbers here by hand without re-running the
tool.

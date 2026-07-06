# go-pandas compatibility report

Counts track the conceptual APIs listed in the matrices; "implemented"
includes partial implementations that cover the common use (a `partial`
status in the matrix). Golden verification: 201 test cases generated from
pandas 2.3 / NumPy 2.0 (see `compat/goldens/`), all passing.

## pandas compatibility

| Area | APIs tracked | APIs implemented | Compatibility |
|---|---:|---:|---:|
| Series core | 42 | 38 | 90% |
| DataFrame core | 45 | 40 | 89% |
| Indexing (loc/iloc/index) | 20 | 14 | 70% |
| GroupBy | 16 | 15 | 94% |
| Merge/join/concat | 15 | 13 | 87% |
| Missing values | 14 | 14 | 100% |
| IO | 18 | 12 | 67% |
| Reshape | 8 | 5 | 63% |
| Window (rolling/expanding) | 14 | 12 | 86% |
| Datetime | 18 | 14 | 78% |
| String accessor | 14 | 13 | 93% |
| **Total** | **224** | **190** | **85%** |

Not implemented in the pandas area: MultiIndex operations, Categorical,
timezone handling, resample, stack/unstack, eval, Excel/SQL/Parquet IO.

## NumPy compatibility

| Area | APIs tracked | APIs implemented | Compatibility |
|---|---:|---:|---:|
| ndarray core (shape/views) | 16 | 15 | 94% |
| constructors | 16 | 16 | 100% |
| broadcasting | 6 | 6 | 100% |
| reductions | 14 | 13 | 93% |
| ufuncs | 26 | 24 | 92% |
| linalg | 10 | 5 | 50% |
| indexing | 12 | 10 | 83% |
| sorting/set ops | 6 | 4 | 67% |
| random | 8 | 4 | 50% |
| **Total** | **114** | **97** | **85%** |

Not implemented in the NumPy area: det/inv/solve/eig/SVD (planned via the
gonum adapter), fancy multi-axis reductions (`keepdims`, axis tuples),
negative slice steps, searchsorted/isin, most random distributions,
typed physical storage (logical dtypes only).

## How the numbers are produced

Each matrix row in `pandas_matrix.md` / `numpy_matrix.md` is one tracked
API; `done` and `partial` count as implemented, `planned` and
`not_supported` do not. Update the matrices and this report together.

# Roadmap

## v0.2 (this release) — aggressive compatibility

- Golden tests generated from real pandas/NumPy (200+ cases).
- Series: rank, diff, pct_change, cumulatives, clip/round/abs, shift,
  argsort, reindex, regex string methods, extended dt accessor.
- DataFrame: duplicated/drop_duplicates, nunique/value_counts, corr/cov,
  clip/round/abs, astype(map), select_dtypes, reindex, expanding,
  dropna thresh/axis.
- NDArray: sort/argsort/unique, concatenate/stack/hstack/vstack,
  isnan/isfinite/isinf, masking, ddof, typed constructors, astype.
- Query: `col.str.contains(...)`, bare boolean columns.
- IO: usecols/nrows/keep_default_na; JSON split/columns orientations.
- Fuzz tests, benchmarks, compatibility scoring.

## v0.3 — stronger pandas

- MultiIndex beyond construction; groupby/set_index integration.
- Categorical dtype.
- Timezone-aware datetimes; to_datetime with formats.
- Resample; time-based rolling windows.
- pivot_table with multiple values/aggfuncs; stack/unstack.
- df.eval; stronger query parser (arithmetic in queries).

## v0.4 — performance backends

- Typed column storage (retire []any).
- Typed NDArray storage (int64/bool buffers).
- Arrow interchange; Parquet and DuckDB adapters; gonum linalg adapter
  (det/inv/solve/eig/SVD).
- Columnar expression engine; optional SIMD kernels.

## v0.5 — compatibility expansion

- Excel and SQL IO.
- ewm; expanding aggregation parity.
- Nullable typed dtypes surfaced in the API.

## v1.0 — stable API

- Frozen public API, documented compatibility level, production-ready
  performance.

# Roadmap

## v0.4.1 (this release) — typed gather

- DataFrame/Series/Index Take without boxing: typed column buffers,
  Int64Index for irregular integer selections, RangeIndex preserved for
  constant-step selections, lazy label lookups. 100K-row filters drop
  from ~260K to ~24 allocations.

## v0.4 — columnar expression engine

- Where/AssignExpr/Query evaluate over typed column buffers with
  three-valued NA masks and Kleene logic; row-map evaluation remains the
  documented fallback (object columns, custom expressions).
- Typed kernels: numeric/string/time comparisons, isin/isna/contains,
  and/or/not, arithmetic with dtype preservation, Where(cond, x, y).
- Plan diagnostics (pd.DebugPlan) and behavior-equivalence tests between
  both engines.

## v0.3 — real typed storage

- Typed column engine behind Series/DataFrame (bool/int/int64/float32/
  float64/string/time + object fallback), typed NDArray backings,
  NumPy-style arithmetic promotion, real Astype, typed CSV/records
  inference, storage introspection, dtype goldens.
- Pulled forward from the old v0.4 plan; the remaining v0.4 items move
  down.

## v0.2 — aggressive compatibility

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

## v0.5 — stronger pandas

- MultiIndex beyond construction; groupby/set_index integration.
- Categorical dtype (typed storage for categories).
- Timezone-aware datetimes; to_datetime with formats.
- Resample; time-based rolling windows.
- pivot_table with multiple values/aggfuncs; stack/unstack.
- df.eval; stronger query parser (arithmetic in queries).

## v0.6 — performance backends

- Integer compute kernels (skip the float64 pass); typed groupby keys.
- Arrow interchange; Parquet and DuckDB adapters; gonum linalg adapter
  (det/inv/solve/eig/SVD).
- Optional SIMD kernels.

## v0.7 — compatibility expansion

- Excel and SQL IO.
- ewm; expanding aggregation parity.

## v1.0 — stable API

- Frozen public API, documented compatibility level, production-ready
  performance.

# Roadmap

## v0.6.1 (this release) — typed concat + stability audit

- Vertical concat appends typed column segments with one-shot numeric
  promotion and NA gaps for missing columns; typed index concatenation;
  pd.ConcatSeries. Every major materialization path (filter, gather,
  groupby, merge, concat) is now typed end to end.

## v0.6 — typed merge / join engine

- Shared-id-space typed join keys, CSR build+probe with exact-size pair
  vectors, deterministic duplicate-key expansion, typed gather
  materialization, typed index joins. 100K-row merges drop from ~700K to
  ~180 allocations.

## v0.5 — typed GroupBy engine

- Group ids from typed key maps (string/bool/time/unified-numeric),
  pairwise [2]int composition for multi-key, segment reducers for every
  aggregation, min/max/first/last as typed index-selector gathers, NA
  key group sorted last (pandas dropna=False parity). 100K-row group
  means drop from ~500K to ~70 allocations.

## v0.4.1 — typed gather

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

## Next: stronger pandas

- MultiIndex beyond construction; groupby/set_index integration.
- Categorical dtype (typed storage for categories).
- Timezone-aware datetimes; to_datetime with formats.
- Resample; time-based rolling windows.
- pivot_table with multiple values/aggfuncs; stack/unstack.
- df.eval; stronger query parser (arithmetic in queries).

## v0.7 — performance backends

- Integer compute kernels (skip the float64 pass).
- Arrow interchange; Parquet and DuckDB adapters; gonum linalg adapter
  (det/inv/solve/eig/SVD).
- Optional SIMD kernels.

## v0.8 — compatibility expansion

- Excel and SQL IO.
- ewm; expanding aggregation parity.

## v1.0 — stable API

- Frozen public API, documented compatibility level, production-ready
  performance.

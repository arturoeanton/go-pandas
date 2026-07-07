# Roadmap

## v0.10 (this release) — reshape depth + prerelease candidate

- Stack/Unstack over MultiIndex (future-stack NA semantics, duplicate
  detection); PivotTable with multiple values/aggfuncs/multi-key index
  on the typed groupby engine; GroupBy Transform (broadcast typed
  gather) and Filter (GroupSize/GroupCount conditions); Query grammar
  closure (arithmetic, parentheses, not in, datetime strings);
  np.isin/np.searchsorted; docs/prerelease.md and full compatibility
  audit (pandas 93% of 134 rows, NumPy 91% of 54).

## v0.10.x — release-candidate hardening

- v0.10.1 shipped the deep hardening pass (panic audit, invariant
  validators, typed NDArray.Take 1-D, typed Stack interleave).
- v0.10.2 shipped the API freeze audit (docs/api_freeze.md), sentinel
  error tests, the feature-tour example and the release checklist.
- Remaining before v1.0: typed Unstack and typed N-D NDArray.Take
  (performance only, semantics already correct), resolution of the
  experimental entries in api_freeze.md.

## v0.9 — to_datetime + basic Resample

- Format-aware pd.ToDatetime: strftime directives (%Y %y %m %d %H %M
  %S .%f %z %%), raise/coerce error modes, deterministic inference
  list (day-first slash form), unix-timestamp units, UTC option.
- DatetimeIndex hardening: NA mask (NaT), typed Take/SlicePos through
  every engine, Start/End/IsMonotonicIncreasing/RawTimes, lookup by
  time.Time or parseable string; SetIndex on a datetime column builds
  a real DatetimeIndex.
- Resample over DatetimeIndex: H/D/W(MS Monday)/MS/ME buckets via
  floor + dense group ids reusing the typed groupby reducers;
  sum/mean/count/min/max/first/last; observed buckets only.

## v0.10 — reshape depth

- stack/unstack over MultiIndex; pivot_table with multiple
  values/aggfuncs.

## v0.8 — real MultiIndex

- Levels + int32 codes storage with sorted unique levels and NA
  components as code -1 (pandas parity, golden-verified). Constructors
  from arrays/Series/tuples with pd.Tuple; multi-column SetIndex;
  ResetIndex level expansion with level_N fallbacks; Loc().Tuple (lazy
  lookup map) and Loc().TuplePrefix (scan); optional
  GroupBy(...).AsIndex(true) MultiIndex output; typed Take/SlicePos
  preserving the index through Where/Take/Head/Tail/DropNA/sort/concat;
  join-by-index via boxed tuple alignment. Stages 1–5 of the plan below
  shipped; merge on levels moved to later.

## v0.9 — datetime depth

- to_datetime with explicit formats; Resample basic (sum/mean/count on
  time buckets).

## v0.7.1 — categorical audit + pre-MultiIndex hardening

- Docs made consistent with v0.7 (removed stale "no typed storage for
  categorical" claims). Implicit categories now require one label
  family (numeric/string/bool/time) so the sorted default order is
  total — mixed families return ErrTypeMismatch; explicit categories
  still accept mixed hashable labels. CodeOf resolves labels through a
  lazily-built lookup map shared per immutable category list (O(1) at
  any cardinality, race-safe). New targeted tests, fuzz targets and
  high-cardinality benchmarks.

The staged v0.8 MultiIndex plan (storage/constructors → multi-column
SetIndex → ResetIndex → Loc by tuple → optional MultiIndex groupby →
merge/join later) shipped stages 1–5 in v0.8.0; merge on index levels
remains for a later phase.

## v0.7 — typed categorical dtype

- pd.Category with real typed storage: int32 codes into a shared
  immutable category list. Astype both ways, Cat() accessor
  (categories/codes/ordered/rename/reorder/set/add/remove), ordered
  rank comparisons, strict explicit categories, WithCategorical CSV
  reads, label-only writers. Code fast paths in every engine: groupby
  slot-array ids (3.4x), counting-sort sort_values (91x), array-pass
  value_counts (134x), code-space merge (2.1x), concat category union,
  ~3.4x smaller storage than strings at 500K rows / 8 labels.

## v0.6.1 — typed concat + stability audit

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

- MultiIndex level operations (swaplevel/droplevel/xs, merge on levels,
  label-range slicing over sorted indexes).
- Timezone-aware datetimes; resample closed/label/origin options and
  MultiIndex-level resample; time-based rolling windows.
- pivot_table with multiple values/aggfuncs; stack/unstack.
- df.eval; stronger query parser (arithmetic in queries).

## Later — performance backends

- Integer compute kernels (skip the float64 pass).
- Arrow interchange; Parquet and DuckDB adapters; gonum linalg adapter
  (det/inv/solve/eig/SVD).
- Optional SIMD kernels.

## Later — compatibility expansion

- Excel and SQL IO.
- ewm; expanding aggregation parity.

## v1.0 — stable API

- Frozen public API, documented compatibility level, production-ready
  performance.

# Changelog

## v0.2.1 - Hardening

Audit-driven patch release: no new subsystems, only correctness fixes,
stronger tests and honest documentation.

### Fixed
- `NDArray.Flatten` returned an array sharing the source buffer for
  contiguous arrays: mutating the "copy" corrupted the source.
- `Series.Unique` / `NUnique` / `ValueCounts` panicked on unhashable
  cell values (e.g. the `[]string` cells produced by `Str().Split`);
  they now hash values safely and unify numeric widths (int 1 ==
  int64 1 == 1.0, like pandas).
- Rolling `center=true` masked every window touching the tail; it now
  clips windows at both edges and lets `MinPeriods` decide, exactly
  matching `s.rolling(w, center=True, min_periods=m)` (verified against
  pandas 2.3.3).
- `Series.Eq(nil)` / `Series.Ne(nil)` now follow the documented uniform
  rule — every comparison against a missing comparand is false — instead
  of `Ne` returning true for present values.
- JSON `columns` orientation sorted row keys lexicographically, so
  frames with 10+ rows round-tripped in scrambled order ("10" before
  "2"); keys now sort numerically when they are all integers.
- The package random source (`Rand`/`Randn`/`RandInt`/`Seed`) is now
  mutex-guarded; `rand.Rand` is not safe for concurrent use.

### Improved
- `cmd/compat-report` computes coverage numbers directly from the
  matrices, so the report can no longer drift from them.
- `compat/coverage_report.md` regenerated from matrix rows (pandas: 98
  rows tracked, 91 implemented; NumPy: 52 rows tracked, 46 implemented)
  with the counting rule stated explicitly.

### Tests
- NDArray: input-mutation suite over arithmetic/sort/unique/astype/
  where/mask, Flatten-independence regression, view write-through
  contract, edge shapes ((0,), (1,), (1,1), (2,1)+(1,2), incompatible).
- Series: NA-vs-NA comparison semantics, diff with periods 2 and -1,
  pct_change over a zero denominator (+Inf), value_counts with NA kept/
  dropped/normalized, unique over unhashable cells, rolling center vs
  pandas values, shift beyond length.
- DataFrame: records with missing keys, no-mutation suite for assign/
  filter/sort/dropna, merge with duplicate keys (inner/left/outer fan-out
  counts) plus validate failures, groupby NA-key ordering and size-vs-
  count semantics, pairwise-complete Corr with NA.
- IO: JSON columns-orientation row order regression, empty-string vs NA
  under custom NA sets, CSV determinism.
- `docs_examples_test.go` executes every runnable README/docs snippet.

### Docs
- README: explicit stability-status and compatibility-testing sections,
  golden generator versions (pandas 2.3.3 / NumPy 2.0.2), how to report
  incompatibilities.
- known_differences: NDArray storage honesty spelled out (logical dtypes
  over float64 storage in v0.2.x).

### Known limitations
- NDArray storage remains float64 (typed storage: v0.4).
- Series arithmetic aligns by position, not labels.
- MultiIndex, Categorical, timezones, resample, stack/unstack remain
  unimplemented and return `ErrNotImplemented` where applicable.

## v0.2.0 — aggressive pandas/NumPy compatibility

### Golden testing
- 200+ golden cases generated from **real pandas 2.3 / NumPy 2.0**
  (`compat/goldens/pandas/*.json`, `compat/goldens/numpy/*.json`),
  covering dataframe core, series core, groupby, merge/join/concat,
  reshape, missing values, datetime, strings, rolling, IO, ndarray core,
  constructors, broadcasting, ufuncs, reductions, linalg, indexing,
  sorting and random properties.
- `internal/testing` assertion helpers (frame/series/array comparison
  with NA semantics and float tolerance).
- `compat/python/run_compat_suite.py` regenerates and re-verifies.
- Compatibility scoring in `compat/coverage_report.md` and
  `compat/known_differences.md`.

### Series
- `Rank` (average/min/max/first/dense), `Argsort`, `Diff`, `PctChange`,
  `Cumsum`/`Cumprod`/`Cummin`/`Cummax`, `Clip`, `Round` (banker's),
  `Abs`, `Shift`, `Reindex`, `ReplaceNA`, `ILoc`/`AtLabel` aliases.
- String accessor: `Match`, `ContainsRegex`, `ReplaceRegex`, `Get`,
  `Slice`.
- Datetime accessor: `DayOfYear`, `Quarter`, `Time`,
  `IsMonthStart/End`, `IsYearStart/End`.
- Expanding windows: count/median/min/max/std/var added.
- Rolling windows: count/median/var added; `pd.MinPeriods` alias.

### DataFrame
- `Duplicated`/`DropDuplicates`, `NUnique`, `ValueCounts`, `Corr`/`Cov`
  (pairwise-complete, ddof=1), `Clip`/`Round`/`Abs`, `Astype(map)`,
  `SelectDTypes` with `pd.Include`/`pd.Exclude`/`pd.Number`,
  `Reindex`/`ReindexColumns`, `ReplaceNA`, `Expanding`.
- `DropNA` gains `Thresh` and `Axis(1)` (drop columns).
- `ResetIndex` now inserts the old index as a column (pandas behavior).
- `SetIndex` takes variadic columns (multi -> ErrNotImplemented).
- `ILoc().Rows/Cols` accept mixed ints and slice specs;
  `Loc().Rows(pd.LabelSlice(a, b))` for inclusive label ranges.
- `Query` supports `col.str.contains/startswith/endswith(...)` and bare
  boolean columns (`active`, `not active`).

### NDArray
- `Sort`/`ArgSort` (last axis, stable), `Unique`, `Concatenate`,
  `Stack`/`HStack`/`VStack`, `IsNaN`/`IsFinite`/`IsInf`, `Mask`,
  `WhereScalar`, `Maximum`/`Minimum`, `VarDDof`/`StdDDof`, `Astype`,
  typed constructors (`ArrayInt`, `ArrayBool`, ...) with logical dtypes.
- `Round` now uses banker's rounding, matching np.round.
- Root NumPy-style functions: `pd.Abs/Sqrt/Exp/Log/.../Clip`,
  `pd.Add/Subtract/Multiply/Divide/Power`, `pd.IsNaN/IsFinite/IsInf`,
  `pd.WhereArray/WhereScalar`. Expression math renamed to
  `pd.AbsExpr/SqrtExpr/LogExpr/ExpExpr` (breaking).

### DTypes and missing values
- `pd.ParseDType` ("int64", "datetime64[ns]", "category", ...),
  `DType.Kind()`, `pd.Number` selector, `pd.ToDatetime`/`pd.ParseDatetime`.

### IO
- CSV: `WithUseCols`, `WithNRows`, `WithKeepDefaultNA`.
- JSON: `split` and `columns` orientations (read and write),
  `pd.JSONOrient` alias, `ReadJSONReader`/`ReadNDJSONReader` at root.

### Testing and tooling
- Fuzz targets: ReadCSV, ReadJSON, Query, FromRecords, Astype, reshape,
  broadcast, slice, series ops.
- Benchmarks: filter/groupby/merge/CSV at 100K rows; ndarray add/
  broadcast/matmul/sum. Notes in `docs/performance.md`.
- Docs: pandas and NumPy translation guides, missing value semantics,
  dtype semantics, roadmap.

### Fixed
- `ndarray.Unique` no longer mutates its input (Data() aliasing bug).
- CSV writer keeps integral floats as `1500.0` so round-trips preserve
  the float dtype.
- `Pivot`/`PivotTable` sort row and column labels like pandas.

## v0.1.0

First public cut: a usable pandas/NumPy-style core for Go — Series,
DataFrame, float64 NDArray with broadcasting and views, indexes, dtype
system with NA/NaT, expression API, groupby, merge/join/concat,
melt/pivot, rolling, CSV/JSON/NDJSON IO, examples and golden tests.

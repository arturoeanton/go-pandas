# Changelog

## v0.2.0 (unreleased) — aggressive pandas/NumPy compatibility

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

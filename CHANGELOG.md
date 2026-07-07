# Changelog

## v0.5.0 - Typed GroupBy Engine

### Added
- `internal/groupby`: typed group-key builders (string, bool, time,
  unified numeric via the float64 buffer; `%v` fallback for
  object-backed keys) producing `GroupIDs` + per-group first rows in one
  pass — no `fmt.Sprint`, no boxed key tuples.
- Multi-key grouping composes per-key ids through comparable `[2]int`
  map keys: one map entry per distinct combination, zero per-row
  allocations.
- Segment reducers driven by group ids: size, count, sum, mean, two-pass
  var/std (ddof=1), median (shared scatter buffer + per-segment sort),
  nunique (shared (group, value) sets), and min/max/first/last as
  **row-index selectors** whose outputs gather typed — min of an int
  column is an int column, of a time column a time column.
- Group label columns are the key columns gathered at each group's first
  row: key dtypes (including NA labels) survive untouched.
- docs/groupby_engine.md; goldens for var and groupby dropna=True/False
  (234 golden cases total).

### Improved
- `GroupBy.Agg/AggList/Sum/Mean/.../Size` no longer build a
  sub-DataFrame per group; `Apply` and object-backed columns keep the
  per-group fallback (same results, more allocations).
- With sorting enabled, the NA-key group (`GroupDropNA(false)`) now
  sorts **last**, matching pandas `dropna=False` (previously it kept its
  first-seen position). Unsorted grouping is unchanged.

### Performance (100K rows, Apple M4, measured)
- String-key mean: ~9.3 ms / ~500K allocs → **0.90 ms / 70 allocs**.
- Int-key mean: 1.1 ms / 45 allocs; multi-key mean (400 groups): 2.9 ms
  / ~3.9K allocs; AggList (3 aggs): 1.2 ms / 88 allocs; nunique: 2.0 ms
  / 89 allocs; object fallback: 4.3 ms / ~100K allocs.

### Compatibility
- All 14 pre-existing pandas groupby goldens pass unchanged on the new
  engine; 3 new golden cases (var, size dropna=True/False) verified
  against pandas 2.3.3.

### Known limitations
- Object-backed keys/values fall back to boxed per-group evaluation.
- `GroupBy.Apply` still materializes per-group sub-frames by design.
- gb.transform / gb.filter remain unimplemented.

## v0.4.1 - Typed gather and Take hardening

Performance patch release: row gathering stops boxing.

### Improved
- `DataFrame.Take` gathers typed column buffers directly and takes the
  row index once, sharing it across result columns (previously every
  column re-took and re-boxed the index, and `newFrame` deep-copied each
  column a second time).
- `Series.Take`/`Slice`/`Head`/`Tail`/`DropNA` preserve typed storage
  with exactly one backing slice + one mask allocation per output.
- `index.Take` is typed: `StringIndex` gathers `[]string`,
  `DatetimeIndex` gathers `[]time.Time`, and a `RangeIndex` selection
  stays a `RangeIndex` when the chosen labels keep a constant step —
  otherwise it becomes the new `Int64Index` (integer labels backed by
  `[]int64`, boxing as plain ints in `At`/`Values`). Negative positions
  (outer-join fills) keep the boxed fallback with missing labels.
- `StringIndex` and `Int64Index` build their label-lookup maps lazily
  (sync.Once), so gather-heavy paths never pay for them.
- `Mask.Selected` pre-counts and allocates once; new
  `expr.PositionsFromMask` / `expr.CountTrueMask` helpers.
- `Series.AsMask` reads bool buffers directly.

### Fixed
- Nothing user-visible: 231 goldens (new: `df.iloc[[0,2,4]]` verifying
  index labels against pandas), preservation, immutability and fuzz
  tests lock the gather behavior.

### Performance (100K rows, Apple M4, measured)
- Where numeric: 4.3 ms / ~260K allocs → **0.89 ms / 24 allocs**.
- Query: 3.2 ms / ~186K allocs → 0.90 ms / 44 allocs.
- DataFrame.Take (33K positions): 0.21 ms / 19 allocs;
  Series.Take: 73 µs / 5 allocs; Index.Take (range→Int64): 26 µs /
  2 allocs; PositionsFromMask: 1 alloc; DropNA: 0.91 ms / 44 allocs.

### Known limitations
- `DataFrame.Slice`/`Series.Slice` return copies, not views (documented).
- GroupBy still boxes group keys; typed groupby keys are the next
  optimization target.

## v0.4.0 - Columnar Expression Engine

### Added
- Columnar expression engine in `expr`: expressions evaluate over the
  v0.3 typed column buffers through an `EvalContext` (column resolver
  closure) producing typed result columns and three-valued predicate
  masks (`Mask{Data, NA}`).
- Typed kernels: numeric/string/time comparisons (scalar and
  column-column), `IsNA`/`NotNA`/`IsIn` (numeric and string sets),
  `Contains`/`StartsWith`/`EndsWith`, Kleene `And`/`Or`/`Not`,
  arithmetic with dtype preservation (int⊗int → Int64 column,
  Div → Float64, string Add → concat), `AbsExpr/SqrtExpr/LogExpr/
  ExpExpr`, `Lower`/`Upper`/`Len` and `Where(cond, x, y)`.
- Plan diagnostics: `df.Plan(expr)` and `pd.DebugPlan(df, expr)` report
  "columnar", "row-fallback" or "error" with the reason.
- 10 expression golden cases generated from real pandas (boolean
  indexing, combined masks, str.contains filters, assign, query).
  Golden total: 220 → 230.
- Engine equivalence tests: 15 predicate shapes run through both paths
  and must produce identical frames; NA/Kleene semantics, dtype
  preservation, immutability and plan diagnostics are unit-tested.

### Improved
- `df.Where`, `df.Filter`-adjacent flows, `df.AssignExpr` and `df.Query`
  use the columnar engine when every operand is typed; the row-map
  evaluator remains as the automatic, behavior-identical fallback
  (object-backed columns, mixed-kind comparisons, custom expressions).
- `AssignExpr` attaches typed result columns without boxing (int*int
  lands as an Int64 column, predicates as Bool columns).

### Performance (100K rows, Apple M4)
- Where numeric: 4.3 ms vs 17.3 ms row-map (~4x).
- AssignExpr numeric: 1.3 ms vs 16.4 ms row-map (~13x; 63 allocs vs
  386K).
- Query (two comparisons + and): 3.2 ms; string contains: 0.62 ms.

### Compatibility
- No public API changes; NA-in-predicate behavior is unchanged
  (filters drop NA rows, assigned predicates store false) and now
  documented in docs/expression_engine.md and docs/missing_values.md.

### Known limitations
- Planning works by attempting evaluation (no static cost model).
- Row gathering still boxes index labels in Take (~260K allocs on a
  100K-row filter); typed index gather is the next optimization.
- Mixed-kind comparisons fall back instead of failing fast.

## v0.3.0 - Real typed storage

The headline: dtypes stopped being labels. NDArray, Series and DataFrame
columns now store real typed Go slices.

### Added
- `internal/column`: the typed column engine — one generic column over
  bool/int/int64/float32/float64/string/time.Time plus the `[]any`
  object fallback, all mask-based for missing values, with a
  boxing-free `Float64s()` buffer accessor.
- NDArray typed backings: `ArrayInt` stores `[]int`, `ArrayInt64`
  `[]int64`, `ArrayFloat32` `[]float32`, `ArrayBool` `[]bool`, and the
  new `ArrayString` stores `[]string` (comparisons, Sort, Unique,
  Astype; arithmetic errors).
- NumPy-style arithmetic dtype promotion: int+int→int, int+int64→int64,
  int+float64→float64, float32+float32→float32, bool arithmetic→int,
  int/int→float64 true division; string arithmetic errors. Integer
  arrays keep integer dtypes through Abs/Clip/Round/scalar ops with
  integral scalars; Sqrt/Exp/... produce floats; ArgMin/ArgMax return
  Int64.
- Real `Astype` everywhere: storage conversion for NDArray, Series and
  DataFrame (float→int truncates toward zero, string parses, →string
  formats, masks survive).
- Typed inference end to end: `SeriesOf([]int)` builds an int column
  without boxing; `[]any{1, nil, 2.5}` promotes to a masked
  Float64Column; `DataFrameFromRecords` and `ReadCSV` produce typed
  columns (string/int/float64/bool/time) with masked NA cells.
- Introspection: `Series.StorageDType`/`IsObjectBacked`,
  `DataFrame.StorageDTypes`, `NDArray.StorageDType`/`RawData`/`Values`/
  `ValueAt`/`SetValue`, root `pd.ArrayString`, `pd.FromSliceTyped`,
  `pd.Invalid`.
- 19 new dtype golden cases generated from real pandas/NumPy (kind
  characters), typed-storage acceptance tests, typed-vs-object
  benchmarks and 4 typed fuzz targets. Golden total: 220.

### Changed (breaking or behavioral)
- `a.Data()` converts non-float64 numeric backings and returns nil for
  string arrays (was: the float64 buffer, always).
- Numeric ufunc methods (`Sqrt`, `Exp`, ...) panic with an
  ErrTypeMismatch message on string arrays (they have no error return).
- `Series.Set` is type-checked: incompatible values return
  ErrTypeMismatch instead of landing in boxed storage; `FillNA` with an
  incompatible fill rebuilds as object-backed.
- Scalar ops keep integer dtypes for integral scalars
  (`ints.MulScalar(2)` is still an int array); reductions over integer
  arrays keep integer dtypes for sum/min/max.
- `ArrayOf`/`AsArray`/`FromSlice`/`MustFromSlice` are generic over all
  supported element types (existing float64 calls compile unchanged).
- Series arithmetic fast path reads typed buffers; float64 reductions no
  longer box per element (~2.6x faster mean on 100K floats).

### Known limitations
- Complex numbers and categorical values stay object-backed.
- `Arange` returns Float64 even for integer arguments (NumPy: int64).
- Integer values above 2^53 lose precision when an operation routes
  through the float64 compute path (arithmetic, Astype).

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

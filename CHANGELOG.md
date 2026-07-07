# Changelog

## v0.10.1 - Release-Candidate Deep Hardening

### Fixed
- NDArray operations on string arrays no longer panic: methods with an
  error channel (arithmetic, axis reductions, argmin/argmax, VarDDof)
  return ErrTypeMismatch; the error-less forms return documented
  results (SumAll/MeanAll/VarAll -> NaN, GtScalar-family -> all-false,
  scalar math/ufuncs -> all-NaN float64). Documented in
  known_differences.md.

### Hardened
- Public API panic audit: TestNoPanicPublicAPIsInvalidInputs hammers
  50+ invalid/empty/degenerate inputs (bad columns, dtypes, tuples,
  formats, frequencies, query syntax, merge specs, pivot specs,
  transform/filter specs, take indices, searchsorted sides, unhashable
  values, empty frames) — none panic. Remaining panics are the
  documented Must* helpers and internal invariant guards.
- internal/checks invariant validators (Series/DataFrame/NDArray/
  Index/MultiIndex/Categorical) used across the new audits: mask/code
  alignment, unique categories/levels, lookup-vs-scan agreement, shape
  products, index-length agreement.
- Table-driven dtype preservation audits (Take/Where/Query/Sort/Head/
  DropNA/Concat/Merge/Transform/Filter/Pivot/Stack/Unstack/Resample)
  across bool/int/float/string/datetime/categorical columns.
- Index preservation audits across RangeIndex/Int64Index/StringIndex/
  DatetimeIndex/MultiIndex for filter/sort/take/transform.
- Aliasing audit: Copy/Take/Slice never share mutable buffers;
  concurrent categorical CodeOf and MultiIndex lookups race-verified.
- Query parser: FuzzQueryParserNoPanic plus Where-vs-Query equivalence
  fuzzing; parse errors always surface (never silent all/none rows).

### Performance
- NDArray.Take typed 1-D gather: ~8.2 ms / ~500K allocs -> **0.24–0.46
  ms / 6 allocs** at 100K elements (int/float64/string; N-D axis form
  keeps the per-slice copier).
- Stack: typed value interleave for same-typed columns plus direct
  MultiIndex code construction (row labels factorize once, column level
  keeps original column order — also pandas' behavior): ~26 ms / ~499K
  allocs -> **~13 ms / ~101K allocs** at 100K x 2. Unstack stays boxed
  (documented deferral: the sparse cell grid makes a typed path
  invasive; revisit post-RC).

### Tests / Fuzzing / Goldens
- New fuzz targets: FuzzQueryParserNoPanic, FuzzSeriesTakeSlice,
  FuzzDataFrameWhereQuery, FuzzNDArrayBroadcastAdd,
  FuzzNDArrayReshapeTranspose, FuzzNDArrayReductions (plus the
  regression corpus). The broadcast fuzzer immediately caught a wrong
  test expectation ((5,1)+(2,) is a valid NumPy broadcast) — kept as a
  seed.
- 6 edge-case goldens (295 total): stack-with-NA, pivot fill_value,
  transform with NA, query precedence, resample all-NA bucket, np.take
  with repeated indexes.

### Docs
- docs/fuzzing.md (targets, smoke/long commands, invariants checked,
  corpus policy) and docs/benchmarking.md (how to run/compare, machine
  notes, expected low-allocation ops, known boxed paths);
  performance.md tables refreshed with measured v0.10.1 numbers.

### Known limitations
- Unstack and N-D NDArray.Take remain boxed; keepdims still planned.

## v0.10.0 - Reshape Depth and Pre-Release Candidate

### Added
- `df.Stack()` — columns pivot into an inner MultiIndex level,
  returning a Series (row-major layout, NA cells KEPT like pandas'
  future_stack; homogeneous columns stay typed). Migration note: the
  v0.1 placeholder returned `(*DataFrame, error)` and always
  ErrNotImplemented — Stack now returns `(*Series, error)`.
- `pd.UnstackSeries(s)` / `df.Unstack()` — the last MultiIndex level
  pivots into columns with pandas-sorted axes and NA fill; duplicate
  (row, column) entries error (aggregate with PivotTable); multi-column
  frames flatten names to column_label (no MultiIndex columns).
- PivotTable depth: multiple values, multiple aggfuncs (`AggFuncs`),
  multi-key index and a columns dimension on the typed groupby engine +
  Unstack; deterministic flat value_agg_label naming; FillValue applies
  to value columns; the historical 1x1x1 path is byte-compatible.
  Multiple Columns keys return ErrNotImplemented (documented).
- `GroupBy.Transform(column, agg)` — group aggregates broadcast back to
  every row with one typed gather (input length/order/index preserved;
  NA keys become NA; any typed-reducer aggregation works).
- `GroupBy.Filter(cond)` with `pd.GroupSize()` / `pd.GroupCount(col)`
  condition builders (Gt/Ge/Lt/Le/Eq/Ne) — whole groups kept or
  dropped, row order and index preserved, typed gather.
- Query grammar closure: arithmetic (+ - * / % with precedence, unary
  minus, parentheses that disambiguate from predicate groups by
  backtracking), `not in`, bool literals, and datetime string
  comparisons against datetime columns (deterministic inference
  layouts; also works in Where/sort via CompareValues). Clear syntax
  errors; existing queries unchanged.
- NumPy small gaps: `a.IsIn(values)` (numeric/bool/string, NaN never
  matches) and `a.SearchSorted(values, side)` (1-D numeric, binary
  search, documented sorted precondition). keepdims stays planned.
- docs/prerelease.md (status, stability, philosophy, comparisons,
  roadmap to v1.0); 17 new goldens (289 total; 12 pandas reshape/query
  + 5 NumPy setops); 8 fuzz targets; 10 benchmarks.

### Improved
- README status section rewritten for pre-v1 framing; translation
  guide and matrices extended; compatibility recount: pandas 93% of
  134 tracked rows, NumPy 91% of 54 — larger surface, honest statuses.

### Performance (Apple M4, 100K rows, measured)
- PivotTable (2 values x 2 aggs x 12 labels) 2.5 ms; Transform mean
  1.0 ms / 62 allocs; Filter 2.1 ms; query arithmetic 0.54 ms
  (columnar); np.isin 0.73 ms / 4 allocs; searchsorted 0.27 ms.
- Stack ~26 ms and Unstack ~22 ms are boxed reshapes (documented
  optimization targets), as is NDArray.Take (~8 ms).

### Compatibility
- Stack keeps NA (future_stack semantics); unstack is last-level-only
  with observed labels; pivot names flatten deterministically; filter
  callbacks unsupported (use Apply); query is not a Python eval. All
  documented in known_differences.md.

### Known limitations
- No MultiIndex columns; single Columns key in PivotTable; keepdims
  planned; stack/unstack boxed path.

## v0.9.0 - to_datetime and Basic Resample

### Added
- Format-aware `pd.ToDatetime` / `Series.ToDatetime`: strftime
  directives %Y %y %m %d %H %M %S .%f (1–6 digits) %z %% translated to
  Go layouts with strict validation (unknown directives error).
  Options: `pd.WithDatetimeFormat`, `pd.WithDatetimeErrors`
  ("raise" default / "coerce" -> NA; "ignore" rejected),
  `pd.WithDatetimeUnit` ("s"/"ms"/"us"/"ns" unix timestamps),
  `pd.WithDatetimeUTC`. nil stays NA, time.Time passes through, empty
  and invalid strings raise/coerce. Without a format, a deterministic
  inference list applies (RFC3339, ISO forms, 2006/01/02, day-first
  slash form) — documented, not dateutil-style broad inference.
- DatetimeIndex hardening: NA mask (NaT labels), typed
  `Take`/`SlicePos` (negative positions become NA labels),
  `Start`/`End`/`IsMonotonicIncreasing`/`RawTimes`, label lookup and
  inclusive `Slice` by time.Time or parseable string. `SetIndex` on a
  datetime column now builds a real DatetimeIndex (previously labels
  were stringified — a documented behavior improvement).
- `DataFrame.Resample(freq)` over a DatetimeIndex: frequencies H, D,
  W (Monday anchor), MS ("M" aliases month-START, documented
  difference; pandas M/ME are month-end) and ME (month-end labels).
  Aggregations Sum/Mean (numeric only, NA skipped, all-NA sum=0
  mean=NA), Count (non-NA per column), Min/Max (typed kernels incl.
  strings/times), First/Last (row order, dtypes preserved). Input
  order irrelevant; NA timestamps skipped; observed buckets only;
  output DatetimeIndex sorted ascending. The engine floors timestamps
  to buckets and reuses the typed GroupBy segment reducers — no
  sub-frame per bucket, no per-row boxing. Resample on a MultiIndex
  returns ErrNotImplemented (planned).
- 9-case timeseries golden suite from pandas 2.3.3 (272 goldens
  total), 4 fuzz targets (format roundtrips, index Take invariants,
  daily/hourly resample sum preservation), 7 benchmarks,
  docs/timeseries.md.

### Improved
- The v0.1 `Resample` placeholder (always ErrNotImplemented, returned
  `(*Resampler, error)`) became the real chainable API
  (`Resample(freq) *Resampler`); errors surface from the aggregation
  calls, like GroupBy.

### Performance (Apple M4, 100K rows, measured)
- ToDatetime explicit format 8.5 ms (~85 ns/row); inference 22 ms.
- Resample: daily sum 2.6 ms / 43 allocs; hourly mean 3.1 ms; monthly
  count 1.8 ms; unsorted input costs the same as sorted.

### Compatibility
- Matrix: 129 tracked rows (+10 time-series rows incl. planned
  timezone/options rows), 91% implemented — counted honestly against
  the larger surface. Golden values verified against pandas resample
  D/h/MS/ME with unsorted input, duplicate timestamps and NA values.

### Known limitations
- No timezone dtype (tz_localize/tz_convert not supported).
- Observed buckets only (pandas fills the frequency grid).
- No closed/label/origin/offset resample options; no MultiIndex-level
  resample; no partial-string datetime indexing; DatetimeIndex lookup
  is a linear scan.

## v0.8.0 - Real MultiIndex

### Added
- Real MultiIndex storage: per-level unique label lists + int32 code
  arrays, the pandas levels/codes model. codes[level][row] == -1 marks
  an NA tuple component; level lists are the sorted unique labels
  (pandas parity, reusing the categorical factorizer) with a
  first-appearance fallback for mixed-family labels. Lazy lookups: a
  per-level label→code map shared by derived indexes and a full-tuple→
  positions map per code layout, both race-safe under sync.Once.
- `pd.Tuple` label type with pandas-style display ("(AR, Buenos
  Aires)"; NA components print as NA); `MultiIndex.Names/Levels/Codes/
  NLevels/Tuple/Tuples/IsNA/PositionsTuple/PositionsPrefix/Take/
  SlicePos`. Constructors: `pd.MultiIndexFromArrays(names, series...)`,
  `pd.MultiIndexFromTuples(names, tuples)`, plus the existing
  `pd.NewMultiIndexFromArrays(arrays, names)` upgraded in place.
- Multi-column `df.SetIndex("c1", "c2", ...)` builds a MultiIndex
  (index columns removed; categorical columns contribute labels; NA
  values become code -1; duplicates allowed). Single-column SetIndex
  keeps the historical simple-index behavior.
- `df.ResetIndex()` expands MultiIndex levels into leading typed
  columns (level names, or level_0/level_1 when unnamed) and restores a
  RangeIndex; SetIndex→ResetIndex round-trips exactly.
- Tuple-based Loc: `df.Loc().Tuple("AR", "BA")` (full tuple via the
  lookup map, duplicates return all rows, nil matches NA) and
  `df.Loc().TuplePrefix("AR")` (leading levels, code scan). Unknown
  tuples error with ErrInvalidIndex like unknown labels.
- Optional groupby index output: `df.GroupBy(a, b).AsIndex(true)` (or
  `pd.GroupAsIndex(true)`) moves group keys into the index — MultiIndex
  for multi-key, plain typed index for one key; aggregations and Size
  honor it; NA key groups keep NA tuple components. The default stays
  as_index=false (keys as columns), a documented difference from
  pandas.
- Concat (preserved index) stacks MultiIndexes with matching level
  counts into one MultiIndex; mixed shapes fall back to boxed tuples.
- 8-case pandas golden suite (set/reset roundtrip, sorted levels,
  codes, loc full tuple + prefix, groupby default + as_index roundtrip,
  NA components). Goldens: 255 -> 263.
- 5 fuzz targets (FromArrays, Take, SetReset roundtrip, tuple lookup vs
  scan, Where preservation) and 8 benchmarks; docs/multiindex.md.

### Improved
- `index.Take` dispatches MultiIndex to a typed code gather (negative
  positions become all-NA tuples), so Where/Take/Head/Tail/DropNA/sort
  preserve the MultiIndex end to end.
- Join BY index now aligns MultiIndexes through boxed tuple keys (the
  index-alignment keyable path understands pd.Tuple); merge ON index
  levels remains unimplemented (use key columns).
- `MultiIndex.At` returns `pd.Tuple` (previously `[]any`) — same
  underlying type, nicer display and comparable behavior via keyable.

### Performance (Apple M4, 100K rows, 8x50 label space, measured)
- Build 5.9 ms; typed Take 0.12 ms / 6 allocs; full tuple lookup
  ~104 ns; prefix lookup ~93 µs (scan); SetIndex 8.3 ms; ResetIndex
  3.4 ms; Where preserving the index 0.46 ms; groupby AsIndex 2.4 ms.

### Compatibility
- Matrix grew from 109 to 119 tracked rows (new MultiIndex rows,
  including planned ones); coverage moves 94% -> 92% honestly against
  the larger surface. Level lists sorted like pandas (golden-verified).

### Known limitations
- No label-range slicing over MultiIndex (needs sorted index); prefix
  lookup scans; no swaplevel/droplevel/xs; merge on levels not
  implemented; levels not compacted after Take; Series MultiIndex
  support is display/Take-level only.

## v0.7.1 - Categorical Audit and Pre-MultiIndex Hardening

### Fixed
- Removed stale documentation saying categorical data has no typed
  storage (known_differences.md, dtype_semantics.md); the storage
  tables now list the category backing (int32 codes + shared list).
- Docs no longer show impossible chaining (`s.Cat().Codes()`): `Cat()`
  returns `(*CategoricalAccessor, error)` and every example handles it.
- `CodeOf` no longer panics (and never did the linear scan's
  uncomparable-type comparison) when asked about an unhashable label —
  it returns -1.

### Improved
- **Implicit category policy**: default (sorted) categories now require
  one label family — numeric (all widths together), string, bool or
  time.Time. Mixed families return `ErrTypeMismatch` with a hint to use
  explicit categories; explicit categories still accept mixed hashable
  labels because their order is user-provided. Documented in
  categorical.md and known_differences.md.
- **CodeOf uses a lazy lookup map**: built at most once per immutable
  category list under a `sync.Once`, shared safely by Take/Slice/Copy
  derivatives (same categories ⇒ same lookup), rebuilt by operations
  that change categories. Constructors that already computed the map
  seed it. 50K-category label resolution: ~18 ns/op.
- Clarified categorical known differences (union concat, observed-only
  groupby, unordered Series comparisons all-false vs accessor/expr
  errors) and performance claims (Apple M4, row counts, 8-label
  cardinality; note that high-cardinality categoricals may not help).

### Tests
- New unit tests: mixed-family policies, default numeric/string/time
  category order, high-cardinality CodeOf, lookup race (with -race),
  Take/accessor immutability, observed-only groupby, CSV/JSON label
  output, concat union, SetCategories NA semantics, and a docs-audit
  test guarding against the stale object-backed claim.
- New fuzz targets: FuzzCategoricalExplicitCategories,
  FuzzCategoricalSetCategories, FuzzCategoricalConcatUnion (invariants:
  no panic, valid codes, code -1 iff NA, unique categories, inputs not
  mutated).
- New benchmarks: BenchmarkCategoricalCodeOfHighCardinality,
  BenchmarkCategoricalOrderedCompareHighCardinality.

### Docs
- categorical.md: implicit family rule, FromAny downgrade note,
  cardinality guidance. performance.md: categorical section with the
  full benchmark table. roadmap.md: staged v0.8 MultiIndex plan
  (storage → SetIndex → ResetIndex → Loc tuple → GroupBy MultiIndex →
  merge later).

### Known limitations
- Goldens unchanged (255; pandas 2.3.3 / NumPy 2.0.2) — no new golden
  needed for this patch.
- `column.FromAny(values, Category)` keeps the general FromAny contract
  and silently downgrades to object on factorization errors;
  `Astype(pd.Category)` and the constructors surface the error.
- Cat-vs-cat column comparisons in Series/expr still compare labels via
  the generic path (scalar comparisons use codes).

## v0.7.0 - Typed Categorical Columns

### Added
- `pd.Category` with real typed storage: `CategoricalColumn` holds
  int32 codes into a shared immutable category list plus the NA mask
  (`codes[i] == -1` iff masked). Category labels must be hashable
  scalars; categories are unique and never mutated in place.
- Constructors: `pd.CategoricalSeries` (strings) and
  `pd.NewCategoricalSeries` (boxed) with `pd.WithCategories(...)`
  (explicit list, strict — out-of-list values error) and
  `pd.WithOrdered(true)`. Default categories are the sorted distinct
  labels, matching pandas' `astype("category")` codes exactly.
- `Series.Astype(pd.Category)` and back (`Astype(pd.String)` restores
  labels). `dtype.IsCategorical`; `ParseDType` accepts "categorical".
- `Series.Cat()` accessor: `Categories`, `Codes`, `Ordered`,
  `RenameCategories`, `ReorderCategories`, `SetCategories` (removed
  categories become NA), `AddCategories`, `RemoveCategories`, and
  checked ordered comparisons `Gt/Ge/Lt/Le`.
- Ordered comparisons by category rank: `Series.Gt/Ge/Lt/Le` and the
  expression engine (`pd.Col("size").Gt("m")`) compare codes; on
  unordered categoricals the accessor and expr kernels return
  `ErrInvalidOperation` (Series methods return all-false — no error
  channel). `Eq`/`Ne`/`IsIn` always work.
- `pd.WithCategorical("col", ...)` CSV option — the
  `read_csv(dtype={"col": "category"})` equivalent. CSV/JSON writers
  emit labels, never codes; round-trips preserve the dtype.
- Category-aware engines, all on codes:
  - GroupBy: codes are dense group ids — slot array instead of a hash
    map; groups sort by category rank.
  - Merge: shared id space from codes (cat↔cat via category remap,
    cat↔string via seeded label map); outer-merge key columns stay
    categorical through code-space coalesce with category union.
  - Sort: stable O(n+k) counting sort over codes, NA last.
  - ValueCounts: one array pass; includes zero-count categories and
    breaks ties in category order, like pandas.
  - Concat: code stacking with category-list union (see known
    differences); ordered survives identical ordered lists.
- docs/categorical.md; 12-case categorical golden suite generated from
  pandas 2.3.3 (goldens total: 255); 5 fuzz targets including
  string-vs-categorical engine equivalence for groupby/merge/sort/
  concat; benchmark suite with memory comparison.

### Performance (Apple M4, 8 categories, measured)
- GroupBy mean 500K rows: 4.5 ms → **1.3 ms** (3.4x).
- SortValues 500K rows: 119 ms → **1.3 ms** (91x).
- ValueCounts 500K rows: 34 ms → **0.25 ms** (134x).
- Merge inner 200K rows: 3.8 ms → **1.8 ms** (2.1x).
- Column storage 500K rows: 8.5 MB → **2.5 MB** (3.4x smaller).

### Compatibility
- Categories are never inferred automatically — explicit opt-in only.
- Differences vs pandas documented in known_differences.md: concat
  unions differing category lists (pandas downgrades to object),
  groupby is observed-only, Series-level unordered comparisons return
  all-false instead of raising.

## v0.6.1 - Typed Concat and Stability Audit

### Added
- `column.ConcatParts`: typed vertical concat engine — same-dtype
  segments append into one typed buffer; compatible numeric mixes
  promote once (int+int64→Int64, int+float64→Float64, bool+int→Int,
  float32+float64→Float64) into one typed buffer; columns missing from a
  frame become masked NA gaps; only genuinely incompatible columns
  (string+numeric, time+string, object inputs) fall back to object — per
  column.
- Typed index concatenation for preserved (non-ignored) indexes:
  integer label families → Int64Index, string → StringIndex, datetime →
  DatetimeIndex, mixed → boxed labels as-is.
- `pd.ConcatSeries(...)`: small typed Series concat with the same
  promotion rules.
- docs/concat_engine.md; 3 new pandas goldens (join=inner, axis=1,
  numeric promotion). Golden total: 243.

### Improved
- `pd.Concat` axis=0 no longer boxes every cell through Values()/Infer;
  axis=1 assembles typed column copies sharing one index (no per-column
  deep re-copy).
- Preserved-index concat used to stringify labels into a StringIndex;
  labels now keep their type (behavior improvement, documented).

### Performance (Apple M4, 100K+100K rows, measured)
- Same schema: 1.24 ms / **17 allocs** (allocations scale with columns,
  not rows).
- Outer with missing column: 0.65 ms / 23 allocs; numeric promotion:
  0.24 ms / 12 allocs; axis=1: 0.92 ms / 24 allocs; object fallback:
  4.7 ms / 22 allocs.

### Compatibility
- All pre-existing concat goldens and tests pass unchanged. With this,
  every major materialization path — filter, gather, groupby, merge,
  concat — is typed end to end.

### Known limitations
- axis=1 aligns positionally (equal row counts required); no label
  alignment.
- Object-backed inputs concat through the boxed path.

## v0.6.0 - Typed Merge / Join Engine

### Added
- `internal/join`: typed hash-join engine. Left and right key tuples map
  into one shared id space through typed maps (string, time, unified
  numeric where int 1 matches 1.0 across frames; `%v` fallback for
  object/mixed keys); multi-key tuples compose pairwise via comparable
  `[2]int` map keys with zero per-row allocations.
- CSR build + ordered probe with pre-counted pair vectors
  (`LeftRows`/`RightRows`/`Match`): duplicate keys expand to their full
  cartesian deterministically, and the quadratic right-join reorder from
  v0.1 is gone.
- `column.GatherCoalesce`: typed same-dtype coalescing gather for merged
  key columns (boxed fallback for mixed key dtypes).
- Typed index joins: `df.Join` derives typed key columns from
  RangeIndex (arithmetic), Int64Index, StringIndex and DatetimeIndex
  backings and runs the same engine.
- Cardinality validation over id vectors (no boxing); `_merge` indicator
  as a typed string column.
- 6 new merge goldens against pandas 2.3.3 (duplicate-key cartesian,
  string and datetime keys, multi-key, outer+indicator, validated
  one_to_one). Golden total: 240.

### Improved
- Merge output materializes through column-level typed gathers sharing
  one RangeIndex — dtypes and NA masks survive; no boxed key columns, no
  per-column index churn.
- Cross joins materialize typed as well.

### Performance (Apple M4, 100K left x 10K right, measured)
- Inner int key: ~17 ms / ~700K allocs → **2.1 ms / 177 allocs**.
- Left string key: 2.5 ms / 175; outer: 2.3 ms / 178; multi-key inner:
  4.5 ms / 133; duplicate keys (1M pairs): 3.7 ms / 30; indicator outer:
  2.6 ms / 182; join by RangeIndex (100K x 100K): 5.7 ms / 563; object
  fallback: 10 ms / ~220K.

### Compatibility
- All pre-existing merge/join goldens and unit tests pass unchanged.
- NA merge keys still never match — now explicitly documented as a
  difference from pandas (which pairs NaN keys) and locked by tests.

### Known limitations
- Object-backed or mixed-kind key pairs use the boxed `%v` id builder.
- Join by MultiIndex remains unsupported.
- pandas sorts outer-join keys; go-pandas preserves probe order (results
  coincide for sorted inputs, golden-verified).

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

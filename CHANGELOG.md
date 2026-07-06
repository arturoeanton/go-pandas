# Changelog

## v0.1.0 (unreleased)

First public cut: a usable pandas/NumPy-style core for Go.

### Added
- `Series`: typed labeled 1-D data with a missing-value mask, arithmetic,
  comparisons, reductions (skipna), sorting, unique/value_counts, string
  and datetime accessors, rolling and expanding windows.
- `DataFrame`: columnar table with stable column order, selection
  (`Col`/`Select`/`Drop`/`Rename`), row access (`Head`/`Tail`/`Slice`/
  `Take`/`Sample`), `Loc`/`ILoc` builders, filtering (`Filter`/`Where`/
  `Query`), assignment (incl. `AssignExpr`), missing-data handling,
  sorting, stats, `Describe`/`Info`, `Apply`/`Map`/`Pipe`.
- `NDArray`: float64 n-dimensional arrays with shape/strides views,
  slicing, NumPy broadcasting, ufunc-style math, axis reductions,
  dot/matmul/trace, random constructors.
- `Index`: `RangeIndex`, `StringIndex`, `DatetimeIndex` (partial),
  `MultiIndex` (construction/display), union/intersection/difference/
  alignment.
- `dtype`: inference, casting, promotion and the NA/NaT missing model.
- Expression system: `pd.Col("x").Gt(1)`, arithmetic, logical combinators
  and a small `Query` string parser.
- GroupBy with hash grouping: count/size/sum/mean/median/min/max/var/std/
  first/last, `Agg`/`AggList`/`Apply`.
- Merge (inner/left/right/outer/cross with suffixes, validate, indicator),
  index `Join`, `Concat` (axis 0/1, outer/inner, ignore_index).
- Reshape: `Melt`, `Pivot`, single-agg `PivotTable`; `Stack`/`Unstack`/
  `Resample` return `ErrNotImplemented`.
- IO: CSV (inference, NA values, parse dates, delimiters, limits), JSON
  (records/values) and NDJSON, read and write.
- Golden compatibility tests against pandas/NumPy outputs plus generator
  scripts, and compatibility matrices in `compat/`.

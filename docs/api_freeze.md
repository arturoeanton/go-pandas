# Public API freeze audit (v1.0.0-rc.1)

This is the v1.0 stability classification of the public surface.
Classes:

- **stable** — golden-tested, fuzz-hardened, no rename planned; the
  v1.0 contract. Breaking changes only with a major version.
- **experimental-post-v1** — usable and verified, but the API shape may
  still adjust in a v1.x (with deprecated aliases). Each entry says why
  and what is safe to rely on.
- **deprecated** — kept for compatibility, do not use in new code.
- **planned-post-v1** — documented but not implemented.

No group is ambiguously classified; the former "experimental" items
were resolved in rc.1 (NDArray string semantics finalized, MultiIndex
level operations implemented, resample scope pinned).

Regenerate the raw inventory with `go doc github.com/arturoeanton/go-pandas`
and `go doc ./<pkg>` per package.

## DataFrame — stable

Constructors (`NewDataFrame`, `DataFrameFromRecords/FromMap/FromRows/
FromStructs/FromNDArray`), `Col/MustCol/Select/Drop/Rename/Assign*`,
`Head/Tail/Take/Slice/Sample`, `Where/Filter/Query`, `SortValues/
SortValuesBy/SortIndex`, `GroupBy/GroupByOpts`, `Merge/Join`, `Concat`,
`DropNA/FillNA/IsNA`, `SetIndex/ResetIndex/Reindex/ReindexColumns`,
`Loc/ILoc`, `Describe/Info/Corr/Cov`, `Melt/Pivot/PivotTable`,
`Stack/Unstack`, `Resample`, CSV/JSON/NDJSON IO, `Copy/String/Shape/
Len/Columns/DTypes/StorageDTypes/Index`.

Notes: `MustCol` panics by contract. `Stack` returns `(*Series, error)`
since v0.10 and `Resample` returns `*Resampler` since v0.9 (both
replaced dead v0.1 placeholders; migration notes in the CHANGELOG).

## Series — stable

Constructors (`NewSeries`, `SeriesOf`, typed `IntSeries/Int64Series/
FloatSeries/StringSeries/BoolSeries/TimeSeries`, `CategoricalSeries/
NewCategoricalSeries`), element access (`ILoc/Loc/IAt/AtLabel/Set`),
comparisons (`Eq/Ne/Gt/Ge/Lt/Le/Between/IsIn`), arithmetic
(`Add/Sub/Mul/Div` + `*Scalar`), reductions, `Astype`, `ToDatetime`,
accessors (`Str()/Dt()/Cat()`), `ValueCounts/Unique/NUnique/Rank/
Rolling/Expanding/Shift/Diff/PctChange/Cum*`, `SortValues/SortIndex`,
`FillNA/DropNA/IsNA`, `Take/Slice/Copy/Rename`.

Notes: `Cat()` returns `(*CategoricalAccessor, error)`; comparisons
have no error channel and use the documented incomparable-is-false
rule.

## NDArray — stable

Constructors (`Array/ArrayOf/ArrayInt/.../Arange/Zeros/Ones/
Linspace/FromSlice/MustFromSlice`), shape ops (`Reshape/T/Flatten/
Slice`), broadcasting arithmetic, ufuncs, reductions (`Sum/Mean/Min/
Max/Std/Var` + `*All` + `DDof`), `MatMul`, `Sort/ArgSort/Unique`,
`Take` (1-D and contiguous N-D typed since rc.1), `IsIn`,
`SearchSorted`, `Where/Mask`, random.

**String-array semantics are FINAL (v1.0 contract):** methods with an
error channel return `ErrTypeMismatch` for numeric operations on
string arrays; the error-less legacy forms are compatibility helpers
with documented results (`*All` reductions → NaN, scalar comparisons →
all-false, scalar math/ufuncs → all-NaN float64). Strings are never
silently reinterpreted as numbers. See known_differences.md and
docs/numpy_translation_guide.md.

Planned-post-v1: linalg beyond MatMul (gonum adapter module),
keepdims/axis tuples.

## Index family — stable

`Index` interface, `RangeIndex`, `StringIndex`, `Int64Index`,
`DatetimeIndex` (NA mask, `Start/End/IsMonotonicIncreasing/Times/
RawTimes`), `index.Take`, `FromLabels`.

## MultiIndex — stable

`NewMultiIndexFromArrays`, `MultiIndexFromArrays`,
`MultiIndexFromTuples`, `pd.Tuple`, `Names/Levels/Codes/NLevels/Tuple/
Tuples/IsNA`, `PositionsTuple/PositionsPrefix`, `Take/SlicePos`,
`Loc().Tuple/TuplePrefix`, and — since rc.1 — `DropLevel`, `SwapLevel`,
`ReorderLevels` (name or position selectors, pandas negative positions)
plus `df.XS(key, level)` cross-sections. `NewMultiIndexFromCodes` is
the engine constructor (caller owns invariants).

Experimental-post-v1: label-range slicing and partial-key selection
beyond TuplePrefix/XS — the shapes may change when sorted-index
slicing lands; TuplePrefix and XS themselves are stable.
Planned-post-v1: merge on index levels.

## Categorical — stable

`pd.Category` dtype, constructors + `WithCategories/WithOrdered`,
`Astype` both ways, `CategoricalAccessor` (categories/codes/ordered/
rename/reorder/set/add/remove + checked `Gt/Ge/Lt/Le`), engine fast
paths, `pd.WithCategorical` CSV option.

## GroupBy — stable

`Mean/Sum/Count/Size/Min/Max/First/Last/Std/Var/NUnique/Agg/AggList/
Apply`, `Transform`, `Filter` (+ `GroupSize/GroupCount` builders),
options `GroupDropNA/GroupSort/GroupAsIndex` and chainable `AsIndex`.

Note: go-pandas defaults to as_index=false (documented difference).

## Resampler — stable within its documented scope (final)

`Resample("H"/"D"/"W"/"MS"/"M"/"ME")` + `Sum/Mean/Count/Min/Max/First/
Last`, observed buckets only — this scope is the v1.0 contract; the
API exposes no unsupported options (unknown frequencies error with
ErrInvalidOperation). Advanced options (closed/label/origin/offset)
are **planned-post-v1** and will arrive as functional options without
changing the existing calls.

## Expr / Query — stable grammar as documented

`Col/Lit`, comparison/arithmetic/logical builders, `ParseQuery`
grammar (comparisons, and/or/not, parentheses, + - * / %, in/not in,
literals, str accessor, datetime strings). Anything outside the
documented grammar errors — it is not a Python eval.

## IO — stable

`ReadCSV/ReadCSVReader/ToCSV/WriteCSV` + options, `ReadJSON/ToJSON`
(4 orientations), NDJSON. Parquet/Excel/SQL are out of scope pre-v1.

## DTypes — stable

The `DType` enum and re-exports, `ParseDType`, `Kind`, `IsNA/NA/NaT`
helpers, promotion rules per docs/dtype_semantics.md.

## Errors — frozen

Sentinels: `ErrColumnNotFound`, `ErrIndexOutOfBounds`,
`ErrInvalidIndex`, `ErrInvalidDType`, `ErrTypeMismatch`,
`ErrShapeMismatch`, `ErrBroadcastMismatch`, `ErrNotImplementedBase`
(+ `ErrNotImplemented(feature)`), `ErrInvalidOperation`,
`ErrLengthMismatch`. All public errors wrap these with `%w`;
`errors.Is` tests pin the mapping (errconsistency test).

## Options — stable style

Functional options per subsystem (`With*`, `Group*`, `ValueCounts*`,
`Rank*`, `Concat*`, `DropNA*`); struct options where pandas uses many
keywords (`MergeOptions`, `JoinOptions`, `PivotTableOptions`,
`MeltOptions`, `ConcatOptions`).

## Deprecated

None currently — the v0.9/v0.10 placeholder replacements (`Resample`,
`Stack`) removed APIs that only ever returned ErrNotImplemented, with
CHANGELOG migration notes instead of aliases.

## Freeze verdict (rc.1)

Every group above marked **stable** is the v1.0 contract. The three
former experimental items were resolved: NDArray string semantics are
final, MultiIndex level operations (DropLevel/SwapLevel/ReorderLevels/
XS) shipped and are stable, and the Resampler scope is pinned. What
remains beyond v1.0 is listed per group as experimental-post-v1 or
planned-post-v1 — see docs/v1_plan.md for policies.

# Public API freeze audit (v0.10.2)

This is the pre-v1 stability classification of the public surface.
Classes:

- **stable** — golden-tested, fuzz-hardened, no rename planned; breaking
  changes after v1.0 only with a major version.
- **experimental** — behavior is verified, but the *shape* of the API
  (names, option style, return types) may still adjust before v1.0,
  with deprecated aliases where feasible.
- **deprecated** — kept for compatibility, do not use in new code.
- **planned** — documented but not implemented.

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

## NDArray — stable core, experimental edges

Stable: constructors (`Array/ArrayOf/ArrayInt/.../Arange/Zeros/Ones/
Linspace/FromSlice/MustFromSlice`), shape ops (`Reshape/T/Flatten/
Slice`), broadcasting arithmetic, ufuncs, reductions (`Sum/Mean/Min/
Max/Std/Var` + `*All` + `DDof`), `MatMul`, `Sort/ArgSort/Unique`,
`Take` (1-D typed), `IsIn`, `SearchSorted`, `Where/Mask`, random.

Experimental: string-array semantics for numeric ops (documented
NaN/all-false results, v0.10.1), N-D `Take` performance, linalg
surface beyond MatMul (planned via adapters).

## Index family — stable

`Index` interface, `RangeIndex`, `StringIndex`, `Int64Index`,
`DatetimeIndex` (NA mask, `Start/End/IsMonotonicIncreasing/Times/
RawTimes`), `index.Take`, `FromLabels`.

## MultiIndex — stable storage, experimental level ops

Stable: `NewMultiIndexFromArrays`, `MultiIndexFromArrays`,
`MultiIndexFromTuples`, `pd.Tuple`, `Names/Levels/Codes/NLevels/Tuple/
Tuples/IsNA`, `PositionsTuple/PositionsPrefix`, `Take/SlicePos`,
`Loc().Tuple/TuplePrefix`, `NewMultiIndexFromCodes` (engine
constructor).

Planned: label-range slicing, swaplevel/droplevel/xs, merge on levels.

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

## Resampler — stable within its documented scope

`Resample("H"/"D"/"W"/"MS"/"M"/"ME")` + `Sum/Mean/Count/Min/Max/First/
Last`. Observed buckets only; options (closed/label/origin) planned.

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

## Freeze verdict

The surface above marked **stable** is the v1.0 candidate API. The
experimental edges (NDArray string semantics, MultiIndex level ops,
resample options) either stabilize or gain documented behavior before
v1.0 — see docs/release_checklist.md.

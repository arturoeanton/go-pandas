# MultiIndex (v0.8)

A MultiIndex is a hierarchical row index: every row label is a tuple,
one component per level — pandas' `df.set_index(["country", "city"])`.

## Storage model

Levels + codes, exactly the pandas model:

```go
// index package
type MultiIndex struct {
    names  []string
    levels [][]any   // unique labels per level
    codes  [][]int32 // codes[level][row]; -1 = NA component
    length int
    // lazy lookups: per-level label->code map (shared by derived
    // indexes) and full-tuple->positions map (per code layout)
}
```

Invariants: `len(codes) == len(levels) == len(names)`; every code array
has the index length; level values are unique and immutable (derived
indexes share level slices); `codes[l][i] == -1` marks an NA tuple
component. Level lists are the **sorted** unique labels (pandas parity,
verified by goldens) when the labels form one orderable family — the
factorization reuses the categorical engine; mixed-family levels fall
back to first-appearance order (implementation-defined, documented).
Numeric label widths collapse in lookups: `int 1`, `int64 1` and `1.0`
resolve to the same level entry.

## Constructing

```go
mi, _ := pd.NewMultiIndexFromArrays(
    [][]any{{"AR", "AR", "BR"}, {"BA", "CO", "SP"}},
    []string{"country", "city"})

mi, _ = pd.MultiIndexFromArrays([]string{"country", "city"}, countrySeries, citySeries)
mi, _ = pd.MultiIndexFromTuples([]string{"country", "city"},
    []pd.Tuple{{"AR", "BA"}, {"AR", nil}}) // nil = NA component

mi.Names()  // [country city]
mi.Levels() // [[AR BR] [BA CO SP]]  (sorted unique labels)
mi.Codes()  // [[0 0 1] [0 1 2]]     (-1 = NA)
mi.Tuple(1) // (AR, CO)
```

Duplicate tuples are allowed (like pandas). Empty input, ragged arrays,
name-count mismatches and unhashable labels error.

## SetIndex / ResetIndex

```go
df2, _ := df.SetIndex("country", "city") // 2+ columns -> MultiIndex
back := df2.ResetIndex()                 // levels -> leading columns
```

- One column keeps the historical simple-index behavior; two or more
  build a MultiIndex. Index columns are removed from the data columns.
- Column values become level values (categorical columns contribute
  their **labels**); NA values become code -1.
- `ResetIndex` inserts one column per level (named after the level, or
  `level_0`, `level_1`, ... when unnamed) and restores a RangeIndex.
  Level dtypes round-trip through inference (string/int/float/time).
- `SetIndex` → `ResetIndex` round-trips the frame exactly
  (golden-verified against pandas).
- Neither operation mutates its input.

## Loc by tuple

```go
rows, _ := df2.Loc().Tuple("AR", "Buenos Aires").Get()  // full tuple
rows, _  = df2.Loc().Tuple(pd.Tuple{"AR", "Buenos Aires"}).Get()
rows, _  = df2.Loc().TuplePrefix("AR").Get()            // leading levels
```

- Full-tuple selection resolves through a lazily-built lookup map
  (~100 ns per lookup at 100K rows); duplicate tuples return every
  matching row; a `nil` component matches NA.
- `TuplePrefix` is pandas' `df.loc[("AR", slice(None))]`: it scans the
  code arrays in v0.8 (documented; ~0.1 ms at 100K rows).
- Unknown tuples/prefixes error with `ErrInvalidIndex`, like unknown
  labels in `Loc().Rows`.

## GroupBy as_index

```go
g, _ := df.GroupBy("country", "city").AsIndex(true).Mean("salary")
// g has one "salary" column; group keys form a MultiIndex whose names
// are the key names. g.ResetIndex() restores the default layout.
```

go-pandas **defaults to as_index=false** — group keys stay regular
columns, the historical behavior (a documented difference from pandas'
`as_index=True` default). `AsIndex(true)` (or the `pd.GroupAsIndex`
option) moves the keys into the index: a MultiIndex for multi-key
groupings, a plain typed index for one key. `Size()` and every
aggregation honor it. NA keys (with `GroupDropNA(false)`) become NA
tuple components.

## Take / Slice / engines

`index.Take` gathers codes typed: filtering, `Take`, `Head`/`Tail`,
`DropNA`, sort and `Where` all preserve the MultiIndex. Negative
positions (outer materializations) produce all-NA tuples. Levels are
**not compacted** after Take — codes may reference a level subset
(documented). Derived indexes share the immutable level lists and the
per-level lookup; the tuple-positions lookup rebuilds lazily.

`Concat` (axis=0, preserved index) stacks MultiIndexes with matching
level counts into one MultiIndex (names from the first frame); mixed
shapes fall back to a boxed index of tuples. Join **by index** aligns
MultiIndexes through boxed tuple keys — it works, but has no typed fast
path yet; merge **on** MultiIndex levels is not supported (use columns).

## Display

Tuple labels print pandas-style; NA components print as NA:

```text
                    salary
(AR, Buenos Aires)  1000
(AR, Cordoba)       800
(BR, Sao Paulo)     1500
```

`MultiIndex.String()` truncates after 10 tuples.

## Limitations (v0.8)

- No label-range slicing (`MultiIndex.Slice` returns ErrNotImplemented;
  pandas requires a lexsorted index for that too).
- Prefix lookup scans rather than using a sorted structure.
- No `swaplevel`/`droplevel`/`reorder_levels`; no partial-level `xs`.
- Merge on index levels not implemented (join-by-index works through
  boxed alignment).
- `Series` MultiIndex support is display/Take-level only.

## Performance (Apple M4, 100K rows, 8x50 label space)

```text
BenchmarkMultiIndexBuild100K            ~5.9 ms/op
BenchmarkMultiIndexTake100K             ~0.12 ms/op, 6 allocs
BenchmarkMultiIndexFullTupleLookup100K  ~104 ns/op (lazy map)
BenchmarkMultiIndexPrefixLookup100K     ~93 µs/op (scan)
BenchmarkSetIndexMultiColumn100K        ~8.3 ms/op (boxes key values)
BenchmarkResetIndexMultiIndex100K       ~3.4 ms/op
BenchmarkWherePreserveMultiIndex100K    ~0.46 ms/op
BenchmarkGroupByAsIndexMultiIndex100K   ~2.4 ms/op
```

See [known_differences.md](../compat/known_differences.md) for the
pandas deltas.

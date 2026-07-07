# The typed merge / join engine (v0.6)

## Typed join keys

`Merge` maps the left and right key tuples into one **shared id space**
using typed maps over the column buffers (no `fmt` in typed paths):

| key pair kind | builder |
|---|---|
| string ↔ string | `map[string]int` |
| time ↔ time | `map[time.Time]int` |
| numeric ↔ numeric (int/int64/float32/float64/bool) | `map[float64]int` over the unified buffer — `1` matches `1.0` across frames |
| anything else (object-backed, mixed kinds) | `%v` fallback with numeric normalization (historical semantics) |

Multi-key tuples compose pairwise through comparable `[2]int` map keys —
one entry per distinct combination, zero allocations per row.

## Row pairs and duplicate keys

The build side is indexed CSR-style (two allocations), the probe side
walks in order and the engine pre-counts the output so the pair vectors
`LeftRows`/`RightRows` allocate exactly once. Duplicate keys expand to
the full cartesian of their matches, deterministically: probe-side row
order, matches in build-side row order.

`left ids [1,1]` × `right ids [1,1,1]` → 6 rows, ordered
`(l0,r0)(l0,r1)(l0,r2)(l1,r0)(l1,r1)(l1,r2)`.

## Join types

inner / left / right / outer / cross. `right` is a left join probed from
the right frame (right row order preserved); `outer` appends unmatched
right rows after the left-ordered pairs; `cross` skips hashing entirely.

## NA key behavior

**NA keys never match** — not even each other. This is a documented
difference from pandas, which pairs NaN merge keys together. NA-key rows
still appear as left_only/right_only in left/right/outer joins, with NA
key labels. Multi-key tuples with any NA component behave the same.

## Suffixes, key columns and column order

Unchanged from previous versions (golden-locked): with `On`, key columns
appear once, coalescing left values with right values for right-only
rows (typed same-dtype coalesce, boxed fallback for mixed dtypes); with
`LeftOn`/`RightOn`, the left key column is kept and the right key column
dropped. Duplicate non-key columns take `_x`/`_y` (or custom) suffixes.
Column order: keys, left non-keys, right non-keys, `_merge`.

## Indicator and validation

`Indicator: true` emits a typed string `_merge` column
(both/left_only/right_only). `Validate` checks key cardinality on the id
vectors (no boxing): one_to_one, one_to_many, many_to_one,
many_to_many.

## Index joins

`df.Join(other)` derives typed key columns from the indexes —
RangeIndex labels generate arithmetically, Int64Index/StringIndex/
DatetimeIndex expose their typed backings — and runs the same engine.
Heterogeneous indexes fall back to boxed keys. The result keeps the left
labels (outer joins coalesce labels, boxed).

## Typed materialization

Output columns are `column.Take(pairRows)` gathers: one backing slice +
one mask each, dtypes and NA masks preserved, `-1` rows becoming NA. Key
columns use a typed `GatherCoalesce`. Everything shares one RangeIndex.

## Performance (Apple M4, measured; 100K left × 10K right)

```text
inner int key       2.1 ms / 177 allocs     (v0.5 engine: ~17 ms / ~700K allocs)
left string key     2.5 ms / 175 allocs
outer int key       2.3 ms / 178 allocs
multi-key inner     4.5 ms / 133 allocs
duplicate keys      3.7 ms / 30 allocs      (1M output pairs)
indicator outer     2.6 ms / 182 allocs
join by RangeIndex  5.7 ms / 563 allocs     (100K x 100K)
object fallback     10 ms / ~220K allocs
```

## Fallback cases

Object-backed or mixed-kind key pairs build ids through `%v` strings
(per-row allocation, same matching semantics). Value columns always
materialize typed regardless of the key path.

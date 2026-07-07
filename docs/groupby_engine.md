# The typed GroupBy engine (v0.5)

## Typed group keys

`df.GroupBy(keys...)` builds **group ids** in one pass per key column,
using a typed map over the column's buffer — never `fmt.Sprint`:

| key dtype | id builder |
|---|---|
| string | `map[string]int` over the string buffer |
| bool | two fixed slots |
| time.Time | `map[time.Time]int` |
| int / int64 / float32 / float64 | `map[float64]int` over the unified numeric buffer (numeric widths group together: `1`, `int64(1)` and `1.0` share a group) |
| object-backed | `%v`-string fallback (historical semantics) |

## Group ids and multi-key grouping

The result is a `Plan{GroupIDs, Count, FirstRow}`: `GroupIDs[i]` is row
i's group in first-seen order (`-1` = dropped NA key), and `FirstRow[g]`
is the group's representative row. Multiple keys combine pairwise
through comparable `[2]int` composite map keys — one map entry per
distinct combination, zero allocations per row.

Group **label columns** are not rebuilt from boxed values: they are the
key columns gathered (typed) at each group's `FirstRow`, so a string key
stays a string column, an int key stays an int column, and an NA key
label stays a masked slot.

## Segment reductions

Aggregations run in one pass over `GroupIDs` — no sub-DataFrame per
group:

- `size` counts all rows; `count` counts non-NA values.
- `sum`/`mean` accumulate float64 per group (`sum` of an empty group is
  0; `mean` is NA). `var`/`std` are two-pass, ddof=1. `median` scatters
  values into per-group segments of one shared buffer and sorts each
  segment.
- `min`/`max`/`first`/`last` are **row-index selectors**: the engine
  finds the winning row per group and gathers the output column typed —
  so `min` of an int column is an int column, of a time column a time
  column, of a string column a string column.
- `nunique` counts through one shared `(group, value)` set.

## Supported aggregations by dtype

| dtype | aggregations |
|---|---|
| numeric (int/int64/float32/float64, bool as 0/1) | size, count, sum, mean, median, min, max, var, std, first, last, nunique |
| string | size, count, min, max, first, last, nunique |
| time | size, count, min, max, first, last, nunique |
| any (incl. object) | size, count, first, last |
| object-backed | everything, through the boxed per-group fallback |

Numeric aggregations on string/time columns (sum, mean, ...) return the
same `ErrTypeMismatch` errors as the Series reductions.

## NA behavior

- NA **keys**: dropped by default (`pd.GroupDropNA(false)` keeps them as
  one group whose label is a masked/NA cell). With sorting enabled
  (default), the NA group sorts **last**, matching pandas
  `dropna=False` (golden-verified).
- NA **values**: numeric reductions skip them; a group with no values is
  NA for mean/median (0 for sum, matching `Series.Sum`); `first`/`last`
  pick the first/last non-NA row; `nunique` ignores them.

## Output dtype rules

- key columns keep their input dtype.
- `size`/`count`/`nunique` → Int columns.
- `sum`/`mean`/`median`/`var`/`std` → Float64 columns.
- `min`/`max`/`first`/`last` → the value column's dtype (typed gather).
- Output column names stay `column_agg` (`salary_mean`, `age_min`).

## Performance (100K rows, Apple M4, measured)

```text
string key mean      0.90 ms / 70 allocs     (v0.4.1: ~9.3 ms / ~500K allocs)
int key mean         1.1 ms / 45 allocs
multi-key mean       2.9 ms / ~3.9K allocs   (400 groups)
agg list (3 aggs)    1.2 ms / 88 allocs
nunique              2.0 ms / 89 allocs
object fallback      4.3 ms / ~100K allocs   (boxed keys, still 5x fewer than before)
```

## Fallback cases

Object-backed key columns group through `%v` strings; object-backed
value columns (and any aggregation without a typed kernel) aggregate
through the pre-v0.5 per-group Take + Series reduction. Behavior is
identical either way; only allocations differ. `GroupBy.Apply` still
materializes per-group sub-frames by design.

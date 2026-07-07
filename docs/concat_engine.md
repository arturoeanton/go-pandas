# The typed concat engine (v0.6.1)

## axis=0: vertical concat

`pd.Concat(frames)` stacks each output column from typed segments:

- **Same dtype everywhere** → one typed buffer append (`[]int` + `[]int`
  → one `[]int`), masks copied alongside.
- **Compatible numeric mix** → promoted once through the shared dtype
  rules and written into one typed buffer:
  int+int64→Int64, int+float64→Float64, float32+float64→Float64,
  bool+int→Int.
- **Column missing from a frame** (outer join) → an NA gap: the segment
  is masked, the column keeps its dtype.
- **Incompatible mix** (string+int, time+string, object-backed inputs)
  → object fallback for that column only; sibling columns stay typed.

Column order: first frame's columns, then new columns from later frames
in encounter order. `pd.Join("inner")` keeps only columns present in
every frame.

## axis=1: horizontal concat

Frames must have equal row counts; columns are copied typed side by
side and duplicate names get `_1`, `_2`, ... suffixes. There is **no
index alignment** — rows pair positionally and the first frame's index
is kept (a documented limitation; use `Merge`/`Join` for label-aligned
composition).

## Index behavior

- `pd.IgnoreIndex(true)` → fresh RangeIndex.
- Otherwise labels concatenate **typed**: integer label families
  (RangeIndex/Int64Index) produce an Int64Index, string indexes a
  StringIndex, datetime indexes a DatetimeIndex; mixed families keep
  boxed labels. (Before v0.6.1 preserved labels were stringified.)

## NA and masks

Existing masks copy verbatim; NA gaps mask whole segments; inputs are
never mutated and outputs never alias input buffers.

## Series concat

`pd.ConcatSeries(a, b, ...)` applies the same engine to Series: typed
append, numeric promotion, object fallback; the result gets a fresh
RangeIndex and the first series' name.

## Performance (Apple M4, 100K+100K rows, measured)

```text
same schema (2 cols)        1.24 ms / 17 allocs
outer w/ missing column     0.65 ms / 23 allocs
numeric promotion           0.24 ms / 12 allocs
axis=1 aligned              0.92 ms / 24 allocs
object fallback             4.7 ms / 22 allocs (boxed conversion cost)
```

Allocations scale with the number of columns, not rows.

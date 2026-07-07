# Categorical dtype (v0.7)

`pd.Category` is a real dtype with typed storage, mirroring pandas'
categorical: values are stored as `int32` codes into a shared, immutable
category list, plus the usual NA mask.

```go
// internal/column
type CategoricalColumn struct {
    codes      []int32 // -1 == missing
    categories []any   // unique, never mutated in place
    ordered    bool
    mask       []bool
}
```

Invariants:

- `codes[i] == -1` **iff** `mask[i] == true`.
- Categories are unique and immutable: accessor operations
  (rename/reorder/set) build a new column with a new category list, so
  `Take`/`Slice`/`Copy` share the categories slice safely.
- Category labels must be hashable scalars (bool, ints, uints, floats,
  string, `time.Time`).

## Constructing categoricals

Categories are **never inferred automatically** — you opt in explicitly:

```go
// From strings; categories default to the SORTED distinct labels,
// exactly like pandas' astype("category").
s, _ := pd.CategoricalSeries("size", []string{"m", "s", "l"})

// Explicit categories fix the list and its order; out-of-list values
// are an error (strict). WithOrdered enables ordered comparisons.
s, _ = pd.CategoricalSeries("size", []string{"m", "s", "l"},
    pd.WithCategories("s", "m", "l"), pd.WithOrdered(true))

// Any series converts with Astype; back to labels with Astype(pd.String).
cat, _ := pd.StringSeries("c", data).Astype(pd.Category)

// CSV: read_csv(dtype={"col": "category"}) equivalent.
df, _ := pd.ReadCSV("data.csv", pd.WithCategorical("size"))
```

Like pandas, default categories are sorted, so
`pd.NewSeries("s", []any{"m","s","l","m",nil,"s"}).Astype(pd.Category)`
produces categories `[l m s]` and codes `[1 2 0 1 -1 2]` (verified
against pandas 2.3.3 goldens).

## The Cat() accessor

```go
cat, err := s.Cat() // ErrInvalidDType on non-categorical series
cat.Categories()    // []any (copy)
cat.Codes()         // []int32 (copy; -1 = NA)
cat.Ordered()       // bool

s2, _ := cat.RenameCategories(map[any]any{"s": "small"}) // keeps codes
s2, _ = cat.ReorderCategories([]any{"l","m","s"}, true)  // same set, new order
s2, _ = cat.SetCategories([]any{"m","l"}, false)         // removed -> NA
s2, _ = cat.AddCategories("xl")
s2, _ = cat.RemoveCategories("s")                        // values -> NA
```

## Comparisons

- `Eq` / `Ne` / `IsIn` always work (by code — one `int32` compare per
  row in the expression engine).
- `Gt` / `Ge` / `Lt` / `Le` compare **category rank**, not label value,
  and require `ordered`:
  - `cat.Gt(v)` (accessor) returns `ErrInvalidOperation` on unordered
    categoricals and `ErrTypeMismatch` for unknown labels.
  - `s.Gt(v)` (Series, no error channel) returns all-false for
    unordered categoricals — the uniform "incomparable is false" rule.
  - `df.Where(pd.Col("size").Gt("m"))` uses the code kernel; on an
    unordered categorical it surfaces `ErrInvalidOperation` (it does
    not silently fall back to lexical comparison).

```go
s, _ := pd.CategoricalSeries("size", []string{"m","s","l"},
    pd.WithCategories("s","m","l"), pd.WithOrdered(true))
s.Gt("m") // [false false true] — l > m by rank, s is not
```

## Engine fast paths

Every hot path consumes codes directly, never boxed labels:

| Operation | Path | 500K rows, 8 categories |
|---|---|---|
| GroupBy | codes are dense group ids: slot array, no hash map | mean 4.5ms → 1.3ms (3.4x) |
| Sort | O(n+k) counting sort over codes, stable, NA last | 119ms → 1.3ms (91x) |
| ValueCounts | one array pass, no hashing | 34ms → 0.25ms (134x) |
| Merge | shared id space built from codes (cat↔cat and cat↔string) | inner 3.8ms → 1.8ms (2.1x, 200K) |
| Concat | code stacking with category-list union | stays categorical |
| Storage | int32 codes + mask vs string headers | 8.5MB → 2.5MB (3.4x) |

(`go test ./benchmarks -bench Categorical` on Apple Silicon; run your own.)

GroupBy orders groups by category rank (pandas `sort=True` on
categoricals), sort_values on a categorical column sorts by rank, and
outer merges keep key columns categorical with the union of both
category lists.

## Writing

CSV and JSON writers emit **labels**, never codes. A written categorical
column round-trips through `pd.WithCategorical("col")`.

## Differences from pandas (documented)

- `pd.concat` of categoricals with *different* category lists stays
  categorical with the **union** of the lists (first-seen order), like
  `pd.api.types.union_categoricals`; pandas' plain `concat` downgrades
  to object. Identical ordered lists stay ordered.
- GroupBy only emits **observed** groups (pandas `observed=True`);
  unused categories do not appear as empty groups.
- `Series.Gt` and friends return all-false on unordered categoricals
  instead of raising (no error channel); the `Cat()` accessor and the
  expression engine return the explicit error.
- `ValueCounts` on a categorical includes zero-count categories and
  breaks count ties in category order, like pandas.

See [known_differences.md](../compat/known_differences.md).

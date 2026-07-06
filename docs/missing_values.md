# Missing value semantics

go-pandas mirrors the pandas missing-value model with one unified mask per
Series.

## What is missing

| Value | Missing? | pandas equivalent |
|---|---|---|
| `nil` | yes | `None` |
| `math.NaN()` (float32/64) | yes | `np.nan` |
| `pd.NA()` | yes | `pd.NA` |
| `pd.NaT()` | yes (datetime) | `pd.NaT` |
| `""` (empty string) | **no** | `""` is data |
| `0`, `false` | no | data |

Scalar tests: `pd.IsNA(v)`, `pd.NotNA(v)` and their aliases `pd.IsNull`,
`pd.NotNull`.

## Series and DataFrame API

```go
s.IsNA()  s.NotNA()  s.DropNA()  s.FillNA(v)  s.ReplaceNA(v)  s.HasNA()
df.IsNA() df.NotNA() df.HasNA()  df.FillNA(map[string]any{...})
df.DropNA(
    pd.DropNAHow("any"),        // default; "all" drops all-NA rows
    pd.DropNASubset("a", "b"),  // restrict the check
    pd.DropNAThresh(2),         // keep rows with >= 2 non-NA values
    pd.DropNAAxis(1),           // drop columns instead of rows
)
df.ReplaceNA(v)                  // one fill value for every column
```

## Behavior rules

- **Reductions skip NA by default** (`skipna=True`). Opt out with
  `pd.SkipNA(false)`, which yields NaN when any value is missing.
- **Comparisons with NA are false**, like pandas: `pd.Col("age").Gt(30)`
  never selects a missing age. `Ne` on NA is also false (documented
  difference: pandas `!=` against NaN is True; go-pandas treats all
  NA comparisons uniformly as false).
- **Arithmetic propagates NA**: `NA + x` is NA, in expressions, Series
  ops and cumulative ops (`Cumsum` keeps accumulating across gaps, the
  gap itself stays NA — pandas semantics).
- **Sorting places NA last**, for both ascending and descending order.
- **GroupBy drops NA keys by default** (`pd.GroupDropNA(false)` keeps
  them as their own group).
- **Joins produce NA** for unmatched rows in left/right/outer merges.
- **Rolling windows yield NA** until `MinPeriods` observations are
  available.

## CSV parsing

```go
pd.ReadCSV(path, pd.WithNAValues("", "NA", "NaN", "null", "NULL", "None"))
```

Those are the defaults. Passing `pd.WithNAValues(...)` **replaces** the
set; add `pd.WithKeepDefaultNA(true)` to extend it instead. An empty CSV
cell is missing by default (it is in the default NA set) even though an
empty string in memory is not.

## Display

Missing cells render as `<NA>`; missing floats as `NaN`; missing
datetimes as `NaT`.

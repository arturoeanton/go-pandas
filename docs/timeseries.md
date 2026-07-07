# Time series (v0.9)

`pd.ToDatetime`, a hardened `DatetimeIndex` and a basic `Resample`
engine cover the common pandas workflow:

```go
dates, _ := pd.ToDatetime(df.MustCol("date"), pd.WithDatetimeFormat("%Y-%m-%d"))
df2, _ := df.Assign("date", dates)
df3, _ := df2.SetIndex("date")     // -> DatetimeIndex
daily, _ := df3.Resample("D").Sum()
```

## pd.ToDatetime

```go
pd.ToDatetime(s)                                   // deterministic inference
pd.ToDatetime(s, pd.WithDatetimeFormat("%d/%m/%Y")) // explicit format
pd.ToDatetime(s, pd.WithDatetimeErrors("coerce"))   // bad values -> NA
pd.ToDatetime(s, pd.WithDatetimeUnit("s"))          // numeric unix timestamps
pd.ToDatetime(s, pd.WithDatetimeUTC(true))          // convert to UTC
```

- nil values stay NA; `time.Time` values pass through unchanged.
- Empty and invalid strings **error** under `"raise"` (the default) and
  become **NA** under `"coerce"`. pandas' `errors="ignore"` is not
  supported (it errors, documented).
- Numeric values need `WithDatetimeUnit` ("s", "ms", "us", "ns");
  without it they error/coerce.
- The result is a typed datetime series (`datetime64` dtype, `[]time.Time`
  backing + mask); `Dt()`, CSV/JSON writing and `Astype(pd.String)` work
  as before.

### Format directives

| Directive | Meaning | Go layout |
|---|---|---|
| `%Y` | 4-digit year | 2006 |
| `%y` | 2-digit year | 06 |
| `%m` | zero-padded month | 01 |
| `%d` | zero-padded day | 02 |
| `%H` | hour 00–23 | 15 |
| `%M` | minute | 04 |
| `%S` | second | 05 |
| `%f` | 1–6 digit fraction, must follow "." | 999999 |
| `%z` | numeric offset | -0700 |
| `%%` | literal % | % |

Unknown directives error (no silent mis-parsing).

### Deterministic inference

Without a format, ToDatetime walks a small fixed list — not
pandas/dateutil-style broad inference (documented):

```text
2006-01-02T15:04:05.999999999Z07:00   (RFC3339, optional fraction)
2006-01-02T15:04:05
2006-01-02 15:04:05.999999
2006-01-02 15:04:05
2006-01-02
2006/01/02
02/01/2006     <- day-first wins for the ambiguous slash form
01/02/2006
```

Prefer explicit formats: they are unambiguous and ~2.5x faster.

## DatetimeIndex

`SetIndex` on a datetime column builds a real `DatetimeIndex` (v0.9;
previously labels were stringified). It carries an NA mask (NaT),
gathers typed through every engine (Where/Take/Head/Tail/DropNA/sort),
and offers:

```go
di.Start() / di.End()          // earliest / latest non-NA timestamp
di.IsMonotonicIncreasing()     // false if any NA, like pandas
di.Times() / di.RawTimes()     // typed backing (engine use)
```

Label lookup accepts `time.Time` or a string parseable by the inference
list. Matching is **exact** — there is no pandas partial-string
indexing (`df.loc["2026-01-03"]` selecting a whole day); use
`Loc().RowsBetween(start, stop)` for ranges (inclusive, like pandas).

## Resample

```go
daily, _  := df.Resample("D").Sum()
hourly, _ := df.Resample("H").Mean()
monthly, _ := df.Resample("MS").Count()
```

Requirements and semantics:

- The frame index must be a `DatetimeIndex` (`ErrInvalidIndex`
  otherwise; a MultiIndex — even with a datetime level — returns
  `ErrNotImplemented`, planned).
- Input order does not matter; rows with NA timestamps are skipped.
- **Only observed buckets are emitted** — pandas fills the full
  frequency grid with empty buckets; go-pandas does not (documented,
  consistent with observed-only groupby).
- Output index is a `DatetimeIndex` of bucket labels, ascending.
- No `closed`/`label`/`origin`/`offset` options — planned post-v1 as
  functional options; the current call surface is the v1.0 contract.

### Frequencies

| Alias | Bucket | Label |
|---|---|---|
| `H`/`h`/`hour` | wall-clock hour | bucket start |
| `D`/`d`/`day` | calendar day | midnight |
| `W`/`w`/`week` | week, **Monday** anchor | Monday 00:00 (pandas W anchors Sunday-end) |
| `MS`/`M`/`m`/`month` | calendar month | first day 00:00 (**"M" means month-start here**; pandas M/ME are month-end) |
| `ME` | calendar month | last day 00:00 (pandas ME labels) |

### Aggregations

`Sum`/`Mean` aggregate numeric columns only (non-numeric are skipped,
like `numeric_only=True`). `Count` counts non-NA values per column.
`Min`/`Max` use the typed kernels (numeric, string and time columns).
`First`/`Last` take the first/last non-NA value in row order and
preserve column dtypes. All-NA buckets: sum → 0, mean/min/max → NA
(pandas parity, golden-verified). Masks are preserved.

The engine floors timestamps to buckets, maps buckets to dense group
ids and reuses the typed GroupBy segment reducers — no sub-frame per
bucket, no per-row boxing.

## Known differences

- No timezone dtype, `tz_localize` or `tz_convert` (`WithDatetimeUTC`
  only calls `.UTC()` on parsed values).
- Limited directive set; deterministic inference list.
- Observed buckets only.
- `"M"` = month-start (use `MS`/`ME` to be explicit); `W` anchors
  Monday-start.
- No resample by MultiIndex level.
- No partial-string datetime indexing.

## Performance (Apple M4, 100K rows, measured)

```text
BenchmarkToDatetimeFormat100K       ~8.5 ms/op  (~85 ns/row)
BenchmarkToDatetimeInfer100K        ~22 ms/op   (explicit format ~2.5x faster)
BenchmarkDatetimeIndexLookup100K    ~185 µs/op  (linear scan)
BenchmarkResampleDailySum100K       ~2.6 ms/op, 43 allocs
BenchmarkResampleHourlyMean100K     ~3.1 ms/op, 66 allocs
BenchmarkResampleMonthlyCount100K   ~1.8 ms/op, 25 allocs
BenchmarkResampleUnsorted100K       ~2.7 ms/op  (order does not matter)
```

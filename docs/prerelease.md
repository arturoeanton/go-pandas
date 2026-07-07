# go-pandas project status

> Historical note: this document framed the v0.10 prerelease. Since
> **v1.0.0 the API is frozen** — see docs/api_freeze.md for the
> contract and docs/v1_plan.md for the policies. The sections below
> remain accurate as an orientation guide.

## Where the project stands

go-pandas is **v1.0**: the stable API keeps its names, signatures and
documented semantics for the whole v1 major. Every core path is
verified against **301 golden outputs generated from real pandas
2.3.3 and NumPy 2.0.2**, fuzz-tested, race-tested and benchmarked, with
zero dependencies outside the Go standard library.

## Stable enough to try

These paths are golden-tested, typed end to end and fuzz-hardened:

- DataFrame/Series construction, selection, iloc/loc, boolean filtering
  and the columnar expression engine (`Where`/`Query`/`AssignExpr`).
- GroupBy (aggregations, Agg maps, Transform/Filter, as_index),
  Merge/Join, Concat, sort, missing-data handling.
- Typed dtypes: numeric widths, string, bool, datetime, categorical
  (int32 codes), object fallback.
- MultiIndex (levels+codes), multi-column SetIndex/ResetIndex, tuple
  Loc, Stack/Unstack, PivotTable.
- Time series: format-aware ToDatetime, DatetimeIndex, Resample
  (H/D/W/MS/ME, observed buckets).
- CSV/JSON/NDJSON IO.
- NDArray basics: construction, broadcasting arithmetic, reductions,
  slicing views, sort/unique/isin/searchsorted/take, matmul.

## Still experimental / missing

- Timezones (no tz dtype), resample options (closed/label/origin),
  partial-string datetime indexing.
- MultiIndex level operations (swaplevel/droplevel/xs), merge on index
  levels, label-range slicing.
- eval; query grammar covers comparisons, boolean logic, arithmetic,
  in/not in, str accessor — not arbitrary Python.
- NumPy: linalg beyond matmul (det/inv/solve/eig planned via adapters),
  keepdims, fancy indexing, negative slice steps.
- Parquet/Excel/SQL/Arrow IO (explicitly out of scope pre-v1).

## Compatibility philosophy

**Behavioral, not syntactic.** The goal is that a pandas user can
translate a workflow line by line (docs/pandas_translation_guide.md)
and get the same numbers — verified by generating goldens from real
pandas/NumPy and comparing in CI without Python. Coverage percentages
in compat/coverage_report.md are computed from the tracked matrices and
are deliberately conservative: the tracked surface grows with each
release, and differences are documented, never hidden
(compat/known_differences.md).

## How it differs from the neighbors

- **Gota / dataframe-go**: interface-driven, mostly boxed storage;
  go-pandas keeps real typed buffers per column and per index, with
  measured allocation counts in benchmarks.
- **QFrame**: immutable typed frames with a query API, but a smaller
  pandas-behavior surface (no MultiIndex/categorical/resample parity
  targets and no golden verification against pandas).
- **Gonum**: linear algebra and stats, not a DataFrame; go-pandas'
  NDArray covers the NumPy array surface and can adapt to gonum later.
- **DuckDB / Arrow bindings**: full engines behind cgo/FFI; go-pandas
  is pure Go, zero dependencies, embeddable anywhere Go compiles.

## Performance summary (Apple M4, see docs/performance.md)

100K-row reference points: filter ~0.9 ms, group-mean ~0.9 ms, merge
~2.1 ms, concat ~1.2 ms, resample daily sum ~2.6 ms, pivot_table
(2 values x 2 aggs) ~2.5 ms, groupby transform ~1.0 ms, query with
arithmetic ~0.54 ms. Categorical sort/value_counts are 91x/134x faster
than string columns at 500K rows. Reshape (stack/unstack) is currently
boxed (~25 ms at 200K cells) — a known optimization target.

## Reporting issues

Open a GitHub issue with a minimal reproduction and, when it is a
compatibility bug, the pandas/NumPy snippet plus its output. The golden
suites in compat/python/ are the template: a failing golden case is the
perfect bug report.

## Road to v1.0

1. v0.10.x — release-candidate hardening: API audit fixes, docs,
   example coverage, fuzz time.
2. v0.11 — MultiIndex level ops + remaining reshape gaps
   (stack/unstack performance, multi-columns pivot).
3. v0.12 — performance backends (integer kernels, optional Arrow
   interchange as a separate module).
4. v1.0 — frozen public API, documented compatibility level.

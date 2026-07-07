# Benchmarking go-pandas

## Running

```bash
go test ./benchmarks/ -bench . -benchmem            # full suite
go test ./benchmarks/ -bench GroupBy -benchtime 10x # one family
go test ./internal/column/ -bench . -benchmem      # internal kernels
```

`-benchmem` matters: allocation counts are the project's primary
regression signal — typed paths advertise their allocs/op in
docs/performance.md and the CHANGELOG.

## Machine and variance

All published numbers were measured on an **Apple M4** with the Go
version in go.mod. Expect ±10–20% wall-clock variance between runs and
machines; **allocs/op are stable** and are what to compare first. Run
with `-benchtime 10x` or more and an idle machine for comparisons.

## Comparing before/after

```bash
go test ./benchmarks/ -bench . -benchmem -count 10 > after.txt
# check out the previous tag, repeat into before.txt
benchstat before.txt after.txt   # golang.org/x/perf (dev-only tool)
```

benchstat is optional developer tooling; the library itself has zero
dependencies.

## Expected low-allocation operations

Allocations scale with **columns, not rows** for: Where/Query
(columnar), typed gather (Take/Head/Tail/DropNA), GroupBy aggregations,
Merge, Concat, Resample, GroupBy Transform, categorical
sort/value_counts, MultiIndex Take, NDArray 1-D Take (v0.10.1),
IsIn/SearchSorted.

## Known boxed paths (documented optimization targets)

- **Unstack** rebuilds cells through `[]any` (~21 ms at 200K cells).
- Stack's row-label factorization boxes labels once per row (typed
  value interleave landed in v0.10.1: ~13 ms at 200K cells, down from
  ~26 ms).
- Object-backed columns always use boxed fallbacks (by design).
- N-D `NDArray.Take` (axis form) copies per slice; only the 1-D
  contiguous form is typed.

See [performance.md](performance.md) for the current reference tables.

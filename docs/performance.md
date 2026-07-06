# Performance notes

Run the benchmarks with:

```bash
go test ./benchmarks/ -bench=. -benchmem
```

## Reference numbers (Apple M4, go1.26)

```text
BenchmarkDataFrameFilter1K       ~0.5 ms/op    ~565 KB/op
BenchmarkDataFrameFilter100K     ~28 ms/op     ~68 MB/op
BenchmarkDataFrameGroupBy100K    ~11 ms/op     ~16 MB/op
BenchmarkDataFrameMerge100K      ~16 ms/op     ~58 MB/op
BenchmarkReadCSV100K             ~30 ms/op     ~72 MB/op
BenchmarkNDArrayAdd1M            ~2.6 ms/op    8 MB/op (6 allocs)
BenchmarkNDArrayBroadcast1M      ~2.3 ms/op    8 MB/op (5 allocs)
BenchmarkNDArrayMatMul100x100    ~0.7 ms/op    82 KB/op
BenchmarkNDArraySum1M            ~2.1 ms/op    0 allocs
```

## Current design

- DataFrame storage is **columnar** with stable column order.
- GroupBy and Merge use **hash** grouping/joining (single pass over keys).
- NDArray broadcasting uses **stride-0 views** — no materialized copies;
  elementwise loops allocate only the output buffer.
- Slicing/reshape/transpose return views; `Copy()` is explicit.
- No reflection in hot loops (only `DataFrameFromStructs` uses it).

## Known bottlenecks

- Series values are stored as `[]any`: every cell is boxed, which
  dominates the DataFrame benchmarks above (allocs scale with rows).
  Typed column storage is the v0.4 milestone and should cut filter/
  groupby/merge times by an order of magnitude.
- Expression evaluation builds a `map[string]any` per row. A columnar
  expression engine is planned alongside typed storage.
- `Where`/`AssignExpr` could fuse mask construction and row selection.
- CSV reading parses via `encoding/csv` + per-cell inference; a streaming
  typed parser is planned.
- MatMul is a straightforward ikj loop; blocked/SIMD kernels or the gonum
  adapter are the path to large-matrix performance.

No pandas comparison is claimed here: these numbers only track go-pandas
against itself across versions.

# Performance notes

Run the benchmarks with:

```bash
go test ./benchmarks/ -bench=. -benchmem
```

## Reference numbers (Apple M4, go1.26, v0.3 typed storage)

```text
BenchmarkDataFrameFilter1K        ~0.16 ms/op
BenchmarkDataFrameFilter100K      ~20 ms/op
BenchmarkDataFrameGroupBy100K     ~9.3 ms/op   (was ~11 ms on []any)
BenchmarkDataFrameMerge100K       ~17 ms/op
BenchmarkReadCSV100K              ~32 ms/op
BenchmarkNDArrayAdd1M             ~1.5 ms/op   (float64+float64 fast path)
BenchmarkNDArrayIntAdd            ~4.7 ms/op   (int backing, loader path)
BenchmarkNDArrayBroadcast1M       ~4.5 ms/op
BenchmarkNDArrayMatMul100x100     ~1.9 ms/op
BenchmarkNDArraySum1M             ~2.7 ms/op
BenchmarkNDArrayAstypeIntToFloat  ~3.4 ms/op, 7 allocs (was ~10 ms / 1M allocs boxed)
```

Columnar expression engine vs row-map fallback (the v0.4 win, 100K rows):

```text
BenchmarkWhereNumericColumnar100K        ~4.3 ms/op, 260K allocs
BenchmarkWhereNumericRowMap100K          ~17.3 ms/op, 560K allocs   (~4x slower)
BenchmarkAssignExprNumericColumnar100K   ~1.3 ms/op, 63 allocs
BenchmarkAssignExprNumericRowMap100K     ~16.4 ms/op, 386K allocs   (~13x slower)
BenchmarkQueryNumericColumnar100K        ~3.2 ms/op
BenchmarkStringContainsColumnar100K      ~0.62 ms/op, 31 allocs
BenchmarkBooleanAndColumnar100K          ~5.1 ms/op
```

Typed vs object storage (the v0.3 win):

```text
BenchmarkSeriesFloatMeanTyped     ~140 µs/op, 0.8 MB   2 allocs
BenchmarkSeriesFloatMeanObject    ~370 µs/op, 1.6 MB   3 allocs   (~2.6x slower)
BenchmarkSeriesIntSumTyped        ~330 µs/op            (int->float conversion pass)
BenchmarkSeriesIntSumObject       ~370 µs/op
BenchmarkDataFrameGroupByTyped    ~9.3 ms/op, 13.9 MB
BenchmarkDataFrameGroupByObject   ~9.8 ms/op, 15.5 MB
```

## Current design

- **Columnar expressions** (v0.4): Where/AssignExpr/Query evaluate over
  typed buffers with three-valued NA masks; the row-map evaluator remains
  as fallback for object columns and custom expressions.
- **Typed storage everywhere** (v0.3): Series columns and NDArrays hold
  real `[]int` / `[]float64` / `[]bool` / `[]string` / `[]time.Time`
  buffers plus a missing mask; `[]any` remains only for mixed data.
- Numeric reductions extract typed buffers in one pass (no per-element
  boxing); float64 columns hand out their buffer directly.
- NDArray elementwise kernels use per-array loader/store closures with a
  dense float64+float64 fast path; broadcasting is stride-0 views.
- GroupBy and Merge use hash grouping/joins.

## Known bottlenecks

- GroupBy still builds per-group key strings and boxes group keys
  (~500K allocs at 100K rows); a typed key-hash path is the next win.
- Where/Query still allocate during row gathering: `Take` boxes index
  labels (the ~260K allocs above). A typed index gather is planned.
- Int arrays sum through a float conversion pass; direct integer
  kernels would remove the remaining gap.
- MatMul is a straightforward ikj loop; blocked kernels or the gonum
  adapter are the path to large-matrix performance.
- CSV parses via `encoding/csv` plus per-cell inference.

No pandas comparison is claimed here: these numbers only track go-pandas
against itself across versions.

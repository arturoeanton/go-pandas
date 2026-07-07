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

Columnar engine + typed gather (v0.4/v0.4.1, 100K rows):

```text
BenchmarkWhereNumericColumnar100K        ~0.89 ms/op, 24 allocs   (v0.4.0: 4.3 ms / 260K allocs)
BenchmarkWhereNumericRowMap100K          ~17 ms/op, 560K allocs   (fallback)
BenchmarkAssignExprNumericColumnar100K   ~1.3 ms/op, 63 allocs    (row-map: 16.4 ms / 386K)
BenchmarkQueryNumericColumnar100K        ~0.90 ms/op, 44 allocs
BenchmarkDataFrameWhereStringColumnar100K ~0.65 ms/op, 23 allocs
BenchmarkDataFrameTake100K (33K rows)    ~0.21 ms/op, 19 allocs
BenchmarkSeriesTake100K                  ~73 µs/op, 5 allocs
BenchmarkIndexTakeRange100K              ~26 µs/op, 2 allocs
BenchmarkIndexTakeString100K             ~0.21 ms/op, 3 allocs
BenchmarkPositionsFromMask100K           ~0.10 ms/op, 1 alloc
BenchmarkDropNA100K                      ~0.91 ms/op, 44 allocs
```

Typed merge / join engine (the v0.6 win, 100K left x 10K right):

```text
BenchmarkMergeInnerIntKey100K      ~2.1 ms/op, 177 allocs   (v0.5: ~17 ms / ~700K allocs)
BenchmarkMergeLeftStringKey100K    ~2.5 ms/op, 175 allocs
BenchmarkMergeOuterIntKey100K      ~2.3 ms/op, 178 allocs
BenchmarkMergeMultiKey100K         ~4.5 ms/op, 133 allocs
BenchmarkMergeDuplicateKeys100K    ~3.7 ms/op, 30 allocs    (1M output pairs)
BenchmarkMergeIndicator100K        ~2.6 ms/op, 182 allocs
BenchmarkJoinByRangeIndex100K      ~5.7 ms/op, 563 allocs   (100K x 100K)
BenchmarkMergeObjectFallback100K   ~10 ms/op, ~220K allocs
```

Typed GroupBy engine (the v0.5 win, 100K rows):

```text
BenchmarkGroupByStringKeyMean100K   ~0.90 ms/op, 70 allocs   (v0.4.1: ~9.3 ms / ~500K allocs)
BenchmarkGroupByStringKeySize100K   ~0.84 ms/op, 66 allocs
BenchmarkGroupByIntKeyMean100K      ~1.1 ms/op, 45 allocs
BenchmarkGroupByMultiKeyMean100K    ~2.9 ms/op, ~3.9K allocs (400 groups)
BenchmarkGroupByAggList100K         ~1.2 ms/op, 88 allocs
BenchmarkGroupByNUnique100K         ~2.0 ms/op, 89 allocs
BenchmarkGroupByObjectFallback100K  ~4.3 ms/op, ~100K allocs
```

Typed vs object storage (the v0.3 win):

```text
BenchmarkSeriesFloatMeanTyped     ~140 µs/op, 0.8 MB   2 allocs
BenchmarkSeriesFloatMeanObject    ~370 µs/op, 1.6 MB   3 allocs   (~2.6x slower)
BenchmarkSeriesIntSumTyped        ~330 µs/op            (int->float conversion pass)
BenchmarkSeriesIntSumObject       ~370 µs/op
BenchmarkDataFrameGroupByTyped    ~0.84 ms/op (v0.5 engine)
BenchmarkDataFrameGroupByObject   ~1.8 ms/op
```

## Current design

- **Typed merge/join** (v0.6): shared-id-space typed key maps, CSR
  build+probe with exact-size pair vectors, typed gather
  materialization and typed key coalescing.
- **Typed GroupBy** (v0.5): group ids from typed key maps, segment
  reducers over group ids, min/max/first/last as typed index-selector
  gathers — no sub-DataFrame per group.
- **Typed gather** (v0.4.1): DataFrame/Series Take, Slice, Head/Tail,
  DropNA and Where materialization gather typed buffers and typed index
  labels directly — a 100K-row numeric filter allocates 24 objects.
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

- GroupBy is typed since v0.5 (typed key maps + segment reducers);
  only object-backed columns keep the boxed fallback.
- Row gathering is fully typed since v0.4.1 (Take gathers column
  buffers and index labels without boxing; RangeIndex selections with a
  constant step stay RangeIndex, irregular ones become Int64Index).
- Int arrays sum through a float conversion pass; direct integer
  kernels would remove the remaining gap.
- MatMul is a straightforward ikj loop; blocked kernels or the gonum
  adapter are the path to large-matrix performance.
- CSV parses via `encoding/csv` plus per-cell inference.

No pandas comparison is claimed here: these numbers only track go-pandas
against itself across versions.

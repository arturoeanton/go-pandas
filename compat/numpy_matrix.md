# NumPy compatibility matrix

Statuses: `done`, `partial`, `planned`, `not_supported`.
Since v0.3 arrays store real typed backings ([]bool, []int, []int64,
[]float32, []float64, []string); see [known_differences.md](known_differences.md).

## Constructors

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| np.array | pd.Array / pd.Array2D / pd.FromSlice | done | |
| np.asarray | pd.AsArray / pd.ArrayOf | done | any numeric slice |
| typed arrays | pd.ArrayInt / ArrayInt64 / ArrayFloat32 / ArrayFloat64 / ArrayBool / ArrayString | done | real typed backings (v0.3) |
| np.zeros / np.ones / np.full / np.empty | pd.Zeros / Ones / Full / Empty | done | |
| np.arange / np.linspace / np.logspace | pd.Arange / Linspace / Logspace | done | |
| np.eye / np.identity / np.diag | pd.Eye / Identity / Diag | done | |
| a.astype(t) | a.Astype(dt) | done | converts real storage; float->int truncates |

## Shape, views and joining

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| a.shape / a.ndim / a.size / a.strides / a.dtype | a.Shape()/NDim()/Size()/Strides()/DType() | done | |
| a.tolist() / raw buffer | a.Values() / a.RawData() / a.StorageDType() | done | typed introspection (v0.3) |
| a.reshape | a.Reshape(...) | done | view when contiguous; -1 inference |
| a.flatten / a.ravel | a.Flatten() / a.Ravel() | done | ravel: view when contiguous |
| a.T / np.transpose | a.T() / a.Transpose(axes...) | done | views |
| np.squeeze / np.expand_dims | a.Squeeze(...) / a.ExpandDims(axis) | done | |
| np.concatenate | pd.Concatenate(arrays, axis) | done | |
| np.stack / np.hstack / np.vstack | pd.Stack / pd.HStack / pd.VStack | done | |
| np.broadcast_to | a.BroadcastTo(shape...) | done | stride-0 view |

## Indexing

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| a[i, j] / a[i, j] = v | a.At(i, j) / a.Set(v, i, j) | done | negative indices |
| a[0:2, 1:3] | a.Slice(pd.Slice(0,2), pd.Slice(1,3)) | partial | views; positive steps only |
| a[::2] | a.Slice(pd.SliceStep(0, n, 2)) | done | |
| np.take | a.Take(indices, axis) | done | |
| a[mask] | a.Mask(mask) | done | flattens like NumPy |
| np.where(m, x, y) | pd.WhereArray / ndarray.Where | partial | equal shapes |
| np.where(m, x, scalar) | pd.WhereScalar | done | |
| fancy integer indexing | — | planned | |

## Math

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| broadcasting | all elementwise ops | done | full trailing-dimension rule, dtype-promoting |
| np.add/subtract/multiply/divide/power | pd.Add/Subtract/Multiply/Divide/Power (+ methods) | done | |
| np.maximum / np.minimum | pd.Maximum / pd.Minimum | done | |
| np.abs/sqrt/exp/log/log2/log10 | pd.Abs/Sqrt/Exp/Log + methods | done | |
| np.sin/cos/tan | pd.Sin/Cos/Tan | done | |
| np.floor/ceil/round | pd.Floor/Ceil/Round | done | banker's rounding |
| np.clip | pd.Clip(a, min, max) | done | |
| np.isnan/isfinite/isinf | pd.IsNaN/IsFinite/IsInf | done | -> *BoolArray |
| comparisons | a.Eq/Ne/Gt/Ge/Lt/Le + *Scalar | done | broadcast |

## Reductions

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| a.sum()/a.sum(axis) | a.SumAll() / a.Sum(pd.Axis(i)) | done | single axis |
| a.mean/min/max | same pattern | done | |
| a.std/a.var | a.StdAll()/a.VarAll() (ddof=0) | done | NumPy default |
| ddof | a.StdDDof(d, axis...) / a.VarDDof | done | |
| a.argmin/argmax | a.ArgMin(axis...) / a.ArgMax | done | |
| keepdims / axis tuples | — | planned | |

## Sorting and set operations

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| np.sort | a.Sort() | done | last axis, ascending |
| np.argsort | a.ArgSort() | done | stable |
| np.unique | pd.Unique(a) | done | sorted distinct |
| np.isin | a.IsIn(values) | done | numeric/bool/string; NaN never matches (v0.10) |
| np.searchsorted | a.SearchSorted(values, side) | done | 1-D numeric; sorted precondition documented (v0.10) |

## Linear algebra

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| np.dot | pd.Dot / a.Dot | done | 1-D·1-D, 2-D·1-D, 2-D·2-D |
| np.matmul | pd.MatMul / a.MatMul | done | 2-D |
| np.trace | a.Trace() | done | |
| np.linalg.det/inv/solve | — | planned | gonum adapter (v0.4) |
| np.linalg.eig/svd/qr/cholesky | — | planned | gonum adapter (v0.4) |

## Random

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| np.random.seed | ndarray.Seed | done | values differ from NumPy |
| np.random.rand | pd.Rand | done | |
| np.random.randn | pd.Randn | done | |
| np.random.randint | ndarray.RandInt | done | |
| choice/shuffle/distributions | — | planned | |

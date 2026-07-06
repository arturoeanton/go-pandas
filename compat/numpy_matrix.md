# NumPy compatibility matrix

Statuses: `done`, `partial`, `planned`, `not_supported`.

v0.1 arrays store float64; typed arrays (int/bool) are planned for v0.2.

| NumPy API | go-pandas API | Status | Notes |
|---|---|---|---|
| np.array | pd.Array / pd.ArrayOf / pd.Array2D / pd.FromSlice | done | float64 storage |
| np.zeros | pd.Zeros | done | |
| np.ones | pd.Ones | done | |
| np.full | pd.Full | done | |
| np.empty | pd.Empty | done | zero-initialized in Go |
| np.arange | pd.Arange | done | 1/2/3-arg forms |
| np.linspace | pd.Linspace | done | |
| np.logspace | pd.Logspace | done | |
| np.eye | pd.Eye | done | |
| np.identity | pd.Identity | done | |
| np.diag | pd.Diag | partial | vector -> matrix |
| ndarray.shape | a.Shape() | done | |
| ndarray.strides | a.Strides() | done | element units, not bytes |
| ndarray.ndim | a.NDim() | done | |
| ndarray.size | a.Size() | done | |
| ndarray.dtype | a.DType() | partial | always float64 in v0.1 |
| ndarray.reshape | a.Reshape(...) | done | view when contiguous; -1 inference |
| ndarray.flatten | a.Flatten() | done | copy |
| ndarray.ravel | a.Ravel() | partial | view when contiguous |
| ndarray.T | a.T() | done | view |
| np.transpose | a.Transpose(axes...) | done | view |
| np.squeeze | a.Squeeze(axis...) | done | |
| np.expand_dims | a.ExpandDims(axis) | done | |
| a[i, j] | a.At(i, j) / a.Set(v, i, j) | done | negative indices supported |
| a[0:2, 1:3] | a.Slice(pd.Slice(0,2), pd.Slice(1,3)) | partial | views; positive steps only |
| np.take | a.Take(indices, axis) | done | |
| broadcasting | elementwise ops | done | full trailing-dimension rule, stride-0 views |
| np.broadcast_to | a.BroadcastTo(shape...) | done | |
| np.add / np.subtract / np.multiply / np.divide | a.Add/Sub/Mul/Div(b) | done | with broadcasting |
| np.power / np.mod | a.Pow(b) / a.Mod(b) | done | |
| a + scalar | a.AddScalar(v) etc. | done | |
| np.sqrt/exp/log/log2/log10/sin/cos/tan | a.Sqrt()/... | done | |
| np.abs/floor/ceil/round/clip | a.Abs()/... | done | |
| comparisons (==, >, ...) | a.Eq/Ne/Gt/Ge/Lt/Le(b), *Scalar variants | done | returns *BoolArray |
| np.where | ndarray.Where(cond, x, y) | partial | equal shapes only |
| np.sum | a.Sum(axis...) / a.SumAll() | partial | single-axis reductions |
| np.mean | a.Mean(axis...) / a.MeanAll() | partial | |
| np.std / np.var | a.Std/Var(axis...) | partial | ddof=0 like NumPy |
| np.min / np.max | a.Min/Max(axis...) | partial | |
| np.argmin / np.argmax | a.ArgMin/ArgMax(axis...) | partial | |
| np.dot | pd.Dot / a.Dot | done | 1-D·1-D, 2-D·1-D, 2-D·2-D |
| np.matmul | pd.MatMul / a.MatMul | done | 2-D only |
| np.trace | a.Trace() | done | |
| np.linalg.inv/det/solve | — | planned | v0.2 via gonum adapter |
| np.random.rand | pd.Rand | done | |
| np.random.randn | pd.Randn | done | |
| np.random.randint | ndarray.RandInt | done | |
| np.random.seed | ndarray.Seed | done | |
| dtype promotion | dtype.Promote | partial | used by Series; NDArray is float64 |
| DataFrame/Series interop | s.ToNDArray(), df.ToNDArray(), pd.DataFrameFromNDArray | done | missing -> NaN |

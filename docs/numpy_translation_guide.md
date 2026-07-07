# NumPy → go-pandas translation guide

Since v0.3 arrays store **real typed backings**: `[]bool`, `[]int`,
`[]int64`, `[]float32`, `[]float64` and `[]string`. Arithmetic promotes
dtypes NumPy-style. See [dtype_semantics.md](dtype_semantics.md).

## String arrays and numeric operations (v1.0 contract)

NumPy raises on invalid ufunc dtypes; go-pandas returns errors where a
channel exists and documented values where it does not — strings are
never silently reinterpreted as numbers:

```go
s := ndarray.ArrayString([]string{"a", "b"})
_, err := s.Add(pd.Array([]float64{1, 2})) // ErrTypeMismatch
_, err = s.Sum(pd.Axis(0))                 // ErrTypeMismatch
total := s.SumAll()                        // NaN (no error channel)
mask := s.GtScalar(1)                      // all-false BoolArray
sq := pd.Sqrt(s)                           // all-NaN float64 array
```

## Array creation

```python
a = np.array([1, 2, 3])        # int64
f = np.array([1.5, 2.5])       # float64
m = np.array([[1, 2], [3, 4]])
```

```go
a := pd.ArrayInt([]int{1, 2, 3})          // []int backing
f := pd.Array([]float64{1.5, 2.5})        // []float64 backing
s := pd.ArrayString([]string{"a", "b"})   // []string backing
m, err := pd.Array2D([][]float64{
    {1, 2},
    {3, 4},
})
typed, err := pd.FromSliceTyped([]int{1, 2, 3, 4}, 2, 2) // shaped, int backing
```

Constructors: `Zeros`, `Ones`, `Full`, `Empty`, `Arange`, `Linspace`,
`Logspace`, `Eye`, `Identity`, `Diag`, `Rand`, `Randn`, `AsArray` (these
produce Float64, matching NumPy defaults; note `Arange` stays Float64
even for integer arguments — a documented difference).

Dtype promotion in arithmetic:

```go
c, _ := pd.ArrayInt([]int{1, 2}).Add(pd.Array([]float64{0.5, 0.5}))
c.DType()        // pd.Float64
ii, _ := pd.ArrayInt([]int{1, 2}).Add(pd.ArrayInt([]int{3, 4}))
ii.RawData()     // []int
```

## Shape manipulation

```python
a.reshape(2, 3)     a.Reshape(2, 3)      // -1 inference supported
a.flatten()         a.Flatten()
a.ravel()           a.Ravel()            // view when contiguous
a.T                 a.T()
np.squeeze(a)       a.Squeeze()
np.expand_dims(a,0) a.ExpandDims(0)
np.concatenate      pd.Concatenate([]*pd.NDArray{a, b}, 0)
np.stack            pd.Stack([]*pd.NDArray{a, b}, 0)
np.hstack           pd.HStack([]*pd.NDArray{a, b})
np.vstack           pd.VStack([]*pd.NDArray{a, b})
```

Reshape/transpose/slice return **views**: mutations propagate to the base
array. `Copy()` detaches.

## Broadcasting

```python
a = np.ones((2, 3))
b = np.array([10, 20, 30])
c = a + b
```

```go
a := pd.Ones(2, 3)
b := pd.Array([]float64{10, 20, 30})
c, err := a.Add(b)
```

Full NumPy trailing-dimension rules, including `(8,1,6,1) + (7,1,5)`.
Incompatible shapes return `pd.ErrBroadcastMismatch`.

## Indexing and slicing

```python
a[1, 2]             a.At(1, 2)           // negative indices work
a[0:2, 1:3]         a.Slice(pd.Slice(0, 2), pd.Slice(1, 3))
a[::2]              a.Slice(pd.SliceStep(0, n, 2))
np.take(a, [0,2], 0) a.Take([]int{0, 2}, pd.Axis(0))
a[a > 0]            a.Mask(a.GtScalar(0))     // flattens, like NumPy
np.where(m, a, b)   pd.WhereArray(mask, a, b)
np.where(m, a, 0)   pd.WhereScalar(mask, a, 0)
np.broadcast_to     a.BroadcastTo(2, 3)
```

## Reductions

```python
a.sum(axis=0)       a.Sum(pd.Axis(0))
a.mean()            a.MeanAll()          // scalar form
a.std()             a.StdAll()           // ddof=0, NumPy default
a.std(ddof=1)       a.StdDDof(1)         // 0-d array; .MustAt() for the scalar
a.var(axis=1)       a.Var(pd.Axis(1))
a.argmax(axis=1)    a.ArgMax(pd.Axis(1))
```

`Sum/Mean/...` with an axis return an array; the `*All` forms return
`float64`. `ddof` defaults to 0 like NumPy (pandas Series use ddof=1).

## Ufuncs

Every ufunc exists as a method and a root function:

```go
pd.Sqrt(a)   a.Sqrt()
pd.Abs(a)    a.Abs()
pd.Exp(a)    a.Exp()
pd.Log(a)    a.Log()      // Log2, Log10
pd.Sin(a)    a.Sin()      // Cos, Tan
pd.Floor(a)  a.Floor()    // Ceil, Round (banker's, like np.round)
pd.Clip(a, 0, 10)
pd.IsNaN(a)  pd.IsFinite(a)  pd.IsInf(a)   // -> *BoolArray
```

Binary: `pd.Add`, `pd.Subtract`, `pd.Multiply`, `pd.Divide`, `pd.Power`,
`pd.Maximum`, `pd.Minimum` (all broadcast).

## Sorting / set operations

```python
np.sort(a)          a.Sort()             // along the last axis
np.argsort(a)       a.ArgSort()          // stable
np.unique(a)        pd.Unique(a)
```

## Linear algebra

```python
np.dot(a, b)        pd.Dot(a, b)         // 1-D·1-D, 2-D·1-D, 2-D·2-D
np.matmul(a, b)     pd.MatMul(a, b)
np.trace(a)         a.Trace()
np.linalg.inv(a)    planned (gonum adapter)
```

## Random

```python
np.random.seed(42)  ndarray.Seed(42)
np.random.rand(2,3) pd.Rand(2, 3)
np.random.randn(9)  pd.Randn(9)
```

Values differ from NumPy (different generators); only distributions and
shapes match.

## Introspecting storage

```go
a.DType()          // logical dtype (== storage dtype since v0.3)
a.StorageDType()   // dtype of the physical backing
a.RawData()        // the typed backing slice ([]int, []float64, ...)
a.Values()         // boxed []any, any dtype
a.Data()           // converted []float64 (nil for string arrays)
a.ValueAt(1, 2)    // boxed element (works for strings; At is numeric-only)
```

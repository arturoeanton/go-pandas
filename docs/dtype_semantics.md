# DType semantics

## The dtype set

go-pandas models the pandas/NumPy dtypes as an enum:

```text
bool, int, int8..int64, uint, uint8..uint64, float32, float64,
complex64, complex128, string, bytes, datetime64, timedelta64,
category, object
```

`pd.ParseDType` accepts the pandas spellings:

```go
pd.ParseDType("int64")          // pd.Int64
pd.ParseDType("float64")        // pd.Float64
pd.ParseDType("datetime64[ns]") // pd.Time
pd.ParseDType("category")       // pd.Category
pd.ParseDType("number")         // pd.Number (selector pseudo-dtype)
```

`DType.Kind()` buckets dtypes NumPy-style (signed int, unsigned int,
float, datetime, ...). `pd.Number` matches any numeric dtype in
`SelectDTypes`.

## Inference

`[]int → Int`, `[]int64 → Int64`, `[]float64 → Float64`, `[]bool → Bool`,
`[]string → String`, `[]time.Time → datetime64`. Mixed int/float promotes
to Float64; incompatible mixes fall back to Object; all-NA is Object.

## Promotion

Simplified NumPy rules, verified in `dtype/promote.go`:

```text
bool + int      -> int
int + float     -> float64
float32+float32 -> float32
float32 + int64 -> float64
int32 + int64   -> int64
signed+unsigned -> int64
anything+object -> object
string + number -> object
```

## Conversion

```go
s2, err := s.Astype(pd.Float64)
df2, err := df.Astype(map[string]pd.DType{"age": pd.Int64})
a2, err := arr.Astype(pd.Int64)   // NDArray: truncates like NumPy
numeric, err := df.SelectDTypes(pd.Include(pd.Number))
```

String parsing is supported ("42" → 42); invalid conversions return
`ErrTypeMismatch` naming the offending value. Missing values pass
through any cast unchanged.

## Nullability

Every Series carries a missing mask independent of its dtype, so every
dtype is effectively nullable (pandas "Int64"-style). `NullableDType`
exists as a marker for the future typed-column storage.

## NDArray storage

v0.2 NDArrays store float64 physically; typed constructors
(`pd.ArrayInt`, `pd.ArrayBool`, ...) and `Astype` record the **logical**
dtype and normalize values (truncation for ints, 0/1 for bool). True
typed storage is the v0.4 milestone.

## Display

`df.DTypes()` maps columns to dtypes; `df.Info()` prints pandas-style:

```text
 0  name            3 non-null  string
 1  age             3 non-null  int
 2  joined_at       3 non-null  datetime64
```

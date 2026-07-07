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

## Typed storage (v0.3)

Storage is **real**, not logical metadata:

| Data | Series column backing | NDArray backing |
|---|---|---|
| bool | `[]bool` + mask | `[]bool` |
| int | `[]int` + mask | `[]int` |
| int64 | `[]int64` + mask | `[]int64` |
| float32 | `[]float32` + mask | `[]float32` |
| float64 | `[]float64` + mask | `[]float64` |
| string | `[]string` + mask | `[]string` |
| time.Time | `[]time.Time` + mask | — |
| category (v0.7) | `[]int32` codes + shared `[]any` category list + mask | — |
| MultiIndex labels (v0.8) | per-level `[]any` unique lists + `[]int32` codes | — |
| mixed / unsupported | `[]any` (object) + mask | — |

Introspection: `s.StorageDType()`, `s.IsObjectBacked()`,
`df.StorageDTypes()`, `a.StorageDType()`, `a.RawData()`. For typed-backed
data `DType()` and `StorageDType()` agree; object-backed Series report
`Object` storage under whatever logical dtype they carry.

Still object-backed: mixed values, complex numbers and exotic integer
widths inside `[]any` input. Categorical data is typed since v0.7 —
int32 codes into a shared immutable category list; see
[categorical.md](categorical.md).

## Typed gather and filtering

v0.4.1 optimizes DataFrame/Series filtering by gathering typed column
buffers directly. Take, Slice, Head/Tail, DropNA and Where never convert
typed values through `any`: each output column allocates one backing
slice plus one mask, and index labels gather typed as well (constant-step
selections over a RangeIndex stay a RangeIndex; irregular ones become an
Int64Index whose labels still box as plain ints in At/Values). Slice
returns copies, not views. Object-backed columns keep the boxed path.

## Inference

`[]int → Int`, `[]int64 → Int64`, `[]float64 → Float64`, `[]bool → Bool`,
`[]string → String`, `[]time.Time → datetime64` — all straight into typed
columns without boxing. Boxed `[]any` input infers: homogeneous values
get typed storage, mixed int/float promotes to a Float64 column,
incompatible mixes fall back to Object, all-NA is Object. Missing values
(nil, NA(), NaT(), NaN) live in the mask, never in the buffer.

## Promotion

NumPy-style rules, applied by NDArray arithmetic and Series inference
(verified by dtype golden cases against real NumPy):

```text
bool (+) bool     -> int          (arithmetic; logical stays bool)
bool + int        -> int
int + int         -> int
int + int64       -> int64
int + float64     -> float64
any int + float32 -> float64      (documented choice)
float32 + float32 -> float32
float32 + float64 -> float64
int / int         -> float64      (true division)
string + numeric  -> error
```

## Conversion

`Astype` converts **storage**, not just labels:

```go
s2, _ := s.Astype(pd.Float64)      // IntColumn -> Float64Column
a2, _ := a.Astype(pd.Int64)        // []float64 -> []int64 (truncated)
df2, _ := df.Astype(map[string]pd.DType{"age": pd.Int64})
```

Rules:
- float → integer truncates toward zero (NumPy astype semantics).
- string → numeric parses; invalid strings return ErrTypeMismatch.
  String→int parses through float, so "2.5" truncates to 2.
- numeric → string formats values.
- → bool stores `v != 0`; string "true"/"false"/"1"/"0" parse.
- Missing values survive every conversion (mask carries over).

## Display

`df.DTypes()` maps columns to dtypes; `df.Info()` prints pandas-style:

```text
 0  name            3 non-null  string
 1  age             3 non-null  int
 2  joined_at       3 non-null  datetime64
```

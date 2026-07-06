# gonum adapter (planned)

Bridges `pd.NDArray` with [Gonum](https://gonum.org) matrices to unlock
advanced linear algebra (inverse, determinant, solve, eigen, SVD, QR,
Cholesky).

Planned API (v0.2), guarded by the `gonum` build tag so the core stays
dependency-free:

```go
//go:build gonum

func ToDense(a *pd.NDArray) (*mat.Dense, error)
func FromDense(m mat.Matrix) (*pd.NDArray, error)
func Inv(a *pd.NDArray) (*pd.NDArray, error)
func Det(a *pd.NDArray) (float64, error)
func Solve(a, b *pd.NDArray) (*pd.NDArray, error)
```

Not yet implemented; this directory only reserves the package path.

package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// Dot mirrors np.dot: 1-D · 1-D gives a 0-d scalar array, 2-D · 1-D a
// vector, 2-D · 2-D a matrix product.
func Dot(a, b *NDArray) (*NDArray, error) {
	switch {
	case a.NDim() == 1 && b.NDim() == 1:
		if a.shape[0] != b.shape[0] {
			return nil, fmt.Errorf("%w: dot of vectors with sizes %d and %d", errs.ErrShapeMismatch, a.shape[0], b.shape[0])
		}
		da, db := a.Data(), b.Data()
		acc := 0.0
		for i := range da {
			acc += da[i] * db[i]
		}
		return scalarArray(acc), nil
	case a.NDim() == 2 && b.NDim() == 1:
		if a.shape[1] != b.shape[0] {
			return nil, fmt.Errorf("%w: matrix (%d,%d) dot vector (%d)", errs.ErrShapeMismatch, a.shape[0], a.shape[1], b.shape[0])
		}
		m, k := a.shape[0], a.shape[1]
		out := Zeros(m)
		db := b.Data()
		for i := 0; i < m; i++ {
			acc := 0.0
			for j := 0; j < k; j++ {
				acc += a.data[a.offset+i*a.strides[0]+j*a.strides[1]] * db[j]
			}
			out.data[i] = acc
		}
		return out, nil
	case a.NDim() == 2 && b.NDim() == 2:
		return MatMul(a, b)
	}
	return nil, errs.NotImplemented(fmt.Sprintf("Dot for %d-D and %d-D arrays", a.NDim(), b.NDim()))
}

// MatMul computes the 2-D matrix product.
func MatMul(a, b *NDArray) (*NDArray, error) {
	if a.NDim() != 2 || b.NDim() != 2 {
		return nil, fmt.Errorf("%w: MatMul requires 2-D arrays, got %d-D and %d-D", errs.ErrShapeMismatch, a.NDim(), b.NDim())
	}
	if a.shape[1] != b.shape[0] {
		return nil, fmt.Errorf("%w: matmul (%d,%d) x (%d,%d)", errs.ErrShapeMismatch, a.shape[0], a.shape[1], b.shape[0], b.shape[1])
	}
	m, k, n := a.shape[0], a.shape[1], b.shape[1]
	out := Zeros(m, n)
	for i := 0; i < m; i++ {
		for p := 0; p < k; p++ {
			av := a.data[a.offset+i*a.strides[0]+p*a.strides[1]]
			if av == 0 {
				continue
			}
			for j := 0; j < n; j++ {
				out.data[i*n+j] += av * b.data[b.offset+p*b.strides[0]+j*b.strides[1]]
			}
		}
	}
	return out, nil
}

// Dot is the method form of the package Dot function.
func (a *NDArray) Dot(b *NDArray) (*NDArray, error) { return Dot(a, b) }

// MatMul is the method form of the package MatMul function.
func (a *NDArray) MatMul(b *NDArray) (*NDArray, error) { return MatMul(a, b) }

// Trace returns the sum of the main diagonal of a 2-D array.
func (a *NDArray) Trace() (float64, error) {
	if a.NDim() != 2 {
		return 0, fmt.Errorf("%w: Trace requires a 2-D array", errs.ErrShapeMismatch)
	}
	n := a.shape[0]
	if a.shape[1] < n {
		n = a.shape[1]
	}
	acc := 0.0
	for i := 0; i < n; i++ {
		acc += a.data[a.offset+i*a.strides[0]+i*a.strides[1]]
	}
	return acc, nil
}

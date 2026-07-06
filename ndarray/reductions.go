package ndarray

import (
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
)

// reduceAll folds every element with f starting from init.
func (a *NDArray) reduceAll(init float64, f func(acc, x float64) float64) float64 {
	acc := init
	a.iter(func(off int) {
		acc = f(acc, a.data[off])
	})
	return acc
}

// SumAll returns the sum of all elements.
func (a *NDArray) SumAll() float64 {
	return a.reduceAll(0, func(acc, x float64) float64 { return acc + x })
}

// MeanAll returns the mean of all elements.
func (a *NDArray) MeanAll() float64 {
	n := a.Size()
	if n == 0 {
		return math.NaN()
	}
	return a.SumAll() / float64(n)
}

// MinAll returns the minimum element.
func (a *NDArray) MinAll() float64 {
	if a.Size() == 0 {
		return math.NaN()
	}
	return a.reduceAll(math.Inf(1), math.Min)
}

// MaxAll returns the maximum element.
func (a *NDArray) MaxAll() float64 {
	if a.Size() == 0 {
		return math.NaN()
	}
	return a.reduceAll(math.Inf(-1), math.Max)
}

// VarAll returns the population variance (ddof=0, NumPy default).
func (a *NDArray) VarAll() float64 {
	n := a.Size()
	if n == 0 {
		return math.NaN()
	}
	mean := a.MeanAll()
	acc := 0.0
	a.iter(func(off int) {
		d := a.data[off] - mean
		acc += d * d
	})
	return acc / float64(n)
}

// StdAll returns the population standard deviation (ddof=0).
func (a *NDArray) StdAll() float64 { return math.Sqrt(a.VarAll()) }

// scalarArray wraps a float64 in a 0-dimensional NDArray.
func scalarArray(v float64) *NDArray {
	return &NDArray{data: []float64{v}, shape: []int{}, strides: []int{}, dtype: dtype.Float64}
}

// reduceAxis reduces along one axis with a running fold. finish
// post-processes each accumulated value (e.g. divide by count for mean).
func (a *NDArray) reduceAxis(axis int, init float64, f func(acc, x float64) float64, finish func(acc float64) float64) (*NDArray, error) {
	if err := a.checkAxis(axis); err != nil {
		return nil, err
	}
	outShape := make([]int, 0, len(a.shape)-1)
	for d, s := range a.shape {
		if d != axis {
			outShape = append(outShape, s)
		}
	}
	out := Full(init, outShape...)
	outStrides := computeStrides(outShape)
	coords := make([]int, len(a.shape))
	size := a.Size()
	for i := 0; i < size; i++ {
		off := a.offset
		for d, c := range coords {
			off += c * a.strides[d]
		}
		outPos := 0
		k := 0
		for d, c := range coords {
			if d == axis {
				continue
			}
			outPos += c * outStrides[k]
			k++
		}
		out.data[outPos] = f(out.data[outPos], a.data[off])
		// increment coords
		d := len(coords) - 1
		for d >= 0 {
			coords[d]++
			if coords[d] < a.shape[d] {
				break
			}
			coords[d] = 0
			d--
		}
	}
	if finish != nil {
		for i := range out.data {
			out.data[i] = finish(out.data[i])
		}
	}
	return out, nil
}

// Sum reduces with addition. Without axis the result is a 0-d array with
// the total; with one axis the result drops that axis.
func (a *NDArray) Sum(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.SumAll()), nil
	}
	return a.reduceAxis(axis[0], 0, func(acc, x float64) float64 { return acc + x }, nil)
}

// Mean reduces with the arithmetic mean.
func (a *NDArray) Mean(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.MeanAll()), nil
	}
	n := float64(a.shape[axis[0]])
	return a.reduceAxis(axis[0], 0,
		func(acc, x float64) float64 { return acc + x },
		func(acc float64) float64 { return acc / n })
}

// Min reduces with the minimum.
func (a *NDArray) Min(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.MinAll()), nil
	}
	return a.reduceAxis(axis[0], math.Inf(1), math.Min, nil)
}

// Max reduces with the maximum.
func (a *NDArray) Max(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.MaxAll()), nil
	}
	return a.reduceAxis(axis[0], math.Inf(-1), math.Max, nil)
}

// Var reduces with the population variance (ddof=0).
func (a *NDArray) Var(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.VarAll()), nil
	}
	ax := axis[0]
	mean, err := a.Mean(ax)
	if err != nil {
		return nil, err
	}
	meanExp, err := mean.ExpandDims(ax)
	if err != nil {
		return nil, err
	}
	dev, err := a.Sub(meanExp)
	if err != nil {
		return nil, err
	}
	sq, err := dev.Mul(dev)
	if err != nil {
		return nil, err
	}
	return sq.Mean(ax)
}

// Std reduces with the population standard deviation (ddof=0).
func (a *NDArray) Std(axis ...int) (*NDArray, error) {
	v, err := a.Var(axis...)
	if err != nil {
		return nil, err
	}
	return v.Sqrt(), nil
}

// argReduce finds the flat index (along the axis or globally) selected by
// better.
func (a *NDArray) argReduce(axis []int, better func(cur, best float64) bool) (*NDArray, error) {
	if len(axis) == 0 {
		best := math.NaN()
		bestPos := 0
		pos := 0
		a.iter(func(off int) {
			if pos == 0 || better(a.data[off], best) {
				best = a.data[off]
				bestPos = pos
			}
			pos++
		})
		return scalarArray(float64(bestPos)), nil
	}
	ax := axis[0]
	if err := a.checkAxis(ax); err != nil {
		return nil, err
	}
	outShape := make([]int, 0, len(a.shape)-1)
	for d, s := range a.shape {
		if d != ax {
			outShape = append(outShape, s)
		}
	}
	out := Zeros(outShape...)
	bestVals := make([]float64, shapeSize(outShape))
	seen := make([]bool, shapeSize(outShape))
	outStrides := computeStrides(outShape)
	coords := make([]int, len(a.shape))
	size := a.Size()
	for i := 0; i < size; i++ {
		off := a.offset
		for d, c := range coords {
			off += c * a.strides[d]
		}
		outPos := 0
		k := 0
		for d, c := range coords {
			if d == ax {
				continue
			}
			outPos += c * outStrides[k]
			k++
		}
		v := a.data[off]
		if !seen[outPos] || better(v, bestVals[outPos]) {
			seen[outPos] = true
			bestVals[outPos] = v
			out.data[outPos] = float64(coords[ax])
		}
		d := len(coords) - 1
		for d >= 0 {
			coords[d]++
			if coords[d] < a.shape[d] {
				break
			}
			coords[d] = 0
			d--
		}
	}
	return out, nil
}

// ArgMin returns the index of the minimum (flat, or per-axis).
func (a *NDArray) ArgMin(axis ...int) (*NDArray, error) {
	return a.argReduce(axis, func(cur, best float64) bool { return cur < best })
}

// ArgMax returns the index of the maximum (flat, or per-axis).
func (a *NDArray) ArgMax(axis ...int) (*NDArray, error) {
	return a.argReduce(axis, func(cur, best float64) bool { return cur > best })
}

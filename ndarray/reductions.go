package ndarray

import (
	"fmt"
	"math"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// reduceAll folds every element with f starting from init. Numeric
// only; a string backing returns NaN (no error channel on the *All
// reductions — documented; v0.10.1, previously panicked).
func (a *NDArray) reduceAll(init float64, f func(acc, x float64) float64) float64 {
	if a.floatLoader() == nil {
		return math.NaN()
	}
	load := a.mustFloatLoader("reduction")
	acc := init
	a.iter(func(off int) {
		acc = f(acc, load(off))
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
	if n == 0 || a.floatLoader() == nil {
		return math.NaN()
	}
	load := a.mustFloatLoader("var")
	mean := a.MeanAll()
	acc := 0.0
	a.iter(func(off int) {
		d := load(off) - mean
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

// reduceAxis reduces along one axis with a running fold in float64,
// wrapping the result into outDT storage. finish post-processes each
// accumulated value (e.g. divide by count for mean).
func (a *NDArray) reduceAxis(axis int, outDT dtype.DType, init float64, f func(acc, x float64) float64, finish func(acc float64) float64) (*NDArray, error) {
	if err := a.checkAxis(axis); err != nil {
		return nil, err
	}
	if a.floatLoader() == nil {
		return nil, fmt.Errorf("%w: reduction on %s array", errs.ErrTypeMismatch, a.dtype)
	}
	load := a.mustFloatLoader("reduction")
	outShape := make([]int, 0, len(a.shape)-1)
	for d, s := range a.shape {
		if d != axis {
			outShape = append(outShape, s)
		}
	}
	work := make([]float64, shapeSize(outShape))
	for i := range work {
		work[i] = init
	}
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
		work[outPos] = f(work[outPos], load(off))
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
		for i := range work {
			work[i] = finish(work[i])
		}
	}
	data := allocData(outDT, len(work))
	store := floatStore(data)
	for i, v := range work {
		store(i, v)
	}
	return newDense(data, outShape, outDT), nil
}

// intReductionDType keeps integer dtype for closed reductions.
func (a *NDArray) intReductionDType() dtype.DType {
	if dtype.IsInteger(a.dtype) {
		return a.dtype
	}
	if a.dtype == dtype.Bool {
		return dtype.Int
	}
	return dtype.Float64
}

// Sum reduces with addition. Without axis the result is a 0-d array with
// the total; with one axis the result drops that axis. Integer arrays
// keep an integer result dtype.
func (a *NDArray) Sum(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.SumAll()), nil
	}
	return a.reduceAxis(axis[0], a.intReductionDType(), 0, func(acc, x float64) float64 { return acc + x }, nil)
}

// Mean reduces with the arithmetic mean (always floating point).
func (a *NDArray) Mean(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.MeanAll()), nil
	}
	n := float64(a.shape[axis[0]])
	return a.reduceAxis(axis[0], dtype.Float64, 0,
		func(acc, x float64) float64 { return acc + x },
		func(acc float64) float64 { return acc / n })
}

// Min reduces with the minimum, keeping integer dtypes.
func (a *NDArray) Min(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.MinAll()), nil
	}
	return a.reduceAxis(axis[0], a.intReductionDType(), math.Inf(1), math.Min, nil)
}

// Max reduces with the maximum, keeping integer dtypes.
func (a *NDArray) Max(axis ...int) (*NDArray, error) {
	if len(axis) == 0 {
		return scalarArray(a.MaxAll()), nil
	}
	return a.reduceAxis(axis[0], a.intReductionDType(), math.Inf(-1), math.Max, nil)
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
	if a.floatLoader() == nil {
		return nil, fmt.Errorf("%w: argmin/argmax on %s array", errs.ErrTypeMismatch, a.dtype)
	}
	load := a.mustFloatLoader("argmin/argmax")
	if len(axis) == 0 {
		best := math.NaN()
		bestPos := 0
		pos := 0
		a.iter(func(off int) {
			if pos == 0 || better(load(off), best) {
				best = load(off)
				bestPos = pos
			}
			pos++
		})
		out := newDense([]int64{int64(bestPos)}, []int{}, dtype.Int64)
		return out, nil
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
	outData := make([]int64, shapeSize(outShape))
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
		v := load(off)
		if !seen[outPos] || better(v, bestVals[outPos]) {
			seen[outPos] = true
			bestVals[outPos] = v
			outData[outPos] = int64(coords[ax])
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
	return newDense(outData, outShape, dtype.Int64), nil
}

// ArgMin returns the index of the minimum (flat, or per-axis) as Int64.
func (a *NDArray) ArgMin(axis ...int) (*NDArray, error) {
	return a.argReduce(axis, func(cur, best float64) bool { return cur < best })
}

// ArgMax returns the index of the maximum (flat, or per-axis) as Int64.
func (a *NDArray) ArgMax(axis ...int) (*NDArray, error) {
	return a.argReduce(axis, func(cur, best float64) bool { return cur > best })
}

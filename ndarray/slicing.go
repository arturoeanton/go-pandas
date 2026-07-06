package ndarray

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
)

// SliceSpec describes a range along one axis, like Python's start:stop:step
// with an exclusive stop. Nil Start/Stop mean "from the beginning" / "to
// the end".
type SliceSpec struct {
	Start *int
	Stop  *int
	Step  int
}

// Slice builds a SliceSpec for start:stop.
func Slice(start, stop int) SliceSpec {
	return SliceSpec{Start: &start, Stop: &stop, Step: 1}
}

// SliceStep builds a SliceSpec for start:stop:step.
func SliceStep(start, stop, step int) SliceSpec {
	return SliceSpec{Start: &start, Stop: &stop, Step: step}
}

// All selects the whole axis.
func All() SliceSpec { return SliceSpec{Step: 1} }

// resolve clamps the spec against an axis of size dim and returns
// (start, count, step).
func (s SliceSpec) resolve(dim int) (int, int, int, error) {
	step := s.Step
	if step == 0 {
		step = 1
	}
	if step < 0 {
		return 0, 0, 0, errs.NotImplemented("negative slice step")
	}
	start := 0
	if s.Start != nil {
		start = *s.Start
		if start < 0 {
			start += dim
		}
	}
	stop := dim
	if s.Stop != nil {
		stop = *s.Stop
		if stop < 0 {
			stop += dim
		}
	}
	if start < 0 {
		start = 0
	}
	if stop > dim {
		stop = dim
	}
	if start > dim {
		start = dim
	}
	count := 0
	if stop > start {
		count = (stop - start + step - 1) / step
	}
	return start, count, step, nil
}

// Slice returns a view over the selected ranges. Trailing axes without a
// spec are taken whole.
func (a *NDArray) Slice(specs ...SliceSpec) (*NDArray, error) {
	if len(specs) > len(a.shape) {
		return nil, fmt.Errorf("%w: %d slice specs for %d dimensions", errs.ErrIndexOutOfBounds, len(specs), len(a.shape))
	}
	shape := make([]int, len(a.shape))
	strides := make([]int, len(a.shape))
	offset := a.offset
	for d := range a.shape {
		spec := All()
		if d < len(specs) {
			spec = specs[d]
		}
		start, count, step, err := spec.resolve(a.shape[d])
		if err != nil {
			return nil, err
		}
		offset += start * a.strides[d]
		shape[d] = count
		strides[d] = a.strides[d] * step
	}
	return &NDArray{
		data:    a.data,
		shape:   shape,
		strides: strides,
		offset:  offset,
		dtype:   a.dtype,
		view:    true,
	}, nil
}

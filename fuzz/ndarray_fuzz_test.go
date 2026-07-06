package fuzz_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func FuzzNDArrayReshape(f *testing.F) {
	f.Add(6, 2, 3)
	f.Add(6, -1, 3)
	f.Add(0, 0, 0)
	f.Add(4, 5, 5)
	f.Add(10, -1, -1)
	f.Fuzz(func(t *testing.T, size, d0, d1 int) {
		if size < 0 || size > 1<<12 {
			return
		}
		a := pd.Arange(float64(size))
		out, err := a.Reshape(d0, d1)
		if err != nil {
			return
		}
		if out.Size() != a.Size() {
			t.Fatalf("reshape changed size: %d -> %d", a.Size(), out.Size())
		}
	})
}

func FuzzNDArrayBroadcast(f *testing.F) {
	f.Add(2, 3, 3, 1)
	f.Add(1, 1, 1, 1)
	f.Add(5, 1, 6, 1)
	f.Add(3, 4, 4, 3)
	f.Fuzz(func(t *testing.T, a0, a1, b0, b1 int) {
		clamp := func(v int) int {
			if v < 1 {
				return 1
			}
			if v > 8 {
				return 8
			}
			return v
		}
		a := pd.Zeros(clamp(a0), clamp(a1))
		b := pd.Ones(clamp(b0), clamp(b1))
		out, err := a.Add(b)
		if err != nil {
			return
		}
		// Every element of 0 + 1 must be 1.
		for _, v := range out.Data() {
			if v != 1 {
				t.Fatalf("broadcast add produced %v", v)
			}
		}
	})
}

func FuzzNDArraySlice(f *testing.F) {
	f.Add(10, 0, 5, 1)
	f.Add(10, -3, -1, 2)
	f.Add(10, 5, 2, 1)
	f.Fuzz(func(t *testing.T, size, start, stop, step int) {
		if size < 0 || size > 1<<10 {
			return
		}
		a := pd.Arange(float64(size))
		out, err := a.Slice(pd.SliceStep(start, stop, step))
		if err != nil {
			return
		}
		if out.Size() > a.Size() {
			t.Fatalf("slice grew the array: %d > %d", out.Size(), a.Size())
		}
	})
}

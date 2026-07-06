package compat_test

import (
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func TestNumpyGoldens(t *testing.T) {
	runSuites(t, "numpy", numpyCases)
}

// arrCase adapts an error-free array expression.
func arrCase(f func(t *testing.T) *pd.NDArray) caseFn {
	return func(t *testing.T) (any, error) { return f(t), nil }
}

var numpyCases = map[string]caseFn{
	// constructors ---------------------------------------------------------
	"array_1d":  arrCase(func(t *testing.T) *pd.NDArray { return vArr() }),
	"array_2d":  arrCase(func(t *testing.T) *pd.NDArray { return sqArr(t) }),
	"zeros_2x3": arrCase(func(t *testing.T) *pd.NDArray { return pd.Zeros(2, 3) }),
	"ones_2x3":  arrCase(func(t *testing.T) *pd.NDArray { return pd.Ones(2, 3) }),
	"full_7":    arrCase(func(t *testing.T) *pd.NDArray { return pd.Full(7, 2, 3) }),
	"arange_0_10_2": arrCase(func(t *testing.T) *pd.NDArray {
		return pd.Arange(0, 10, 2)
	}),
	"linspace_0_1_5": arrCase(func(t *testing.T) *pd.NDArray {
		return pd.Linspace(0, 1, 5)
	}),
	"logspace_0_2_3": arrCase(func(t *testing.T) *pd.NDArray {
		return pd.Logspace(0, 2, 3)
	}),
	"eye_3":      arrCase(func(t *testing.T) *pd.NDArray { return pd.Eye(3) }),
	"identity_3": arrCase(func(t *testing.T) *pd.NDArray { return pd.Identity(3) }),
	"diag": func(t *testing.T) (any, error) {
		return pd.Diag(pd.Array([]float64{2, 3}))
	},

	// ndarray_core ------------------------------------------------------------
	"reshape_2x3": func(t *testing.T) (any, error) { return pd.Arange(6).Reshape(2, 3) },
	"reshape_infer": func(t *testing.T) (any, error) {
		return pd.Arange(6).Reshape(3, -1)
	},
	"flatten": arrCase(func(t *testing.T) *pd.NDArray { return mArr(t).Flatten() }),
	"ravel_t": func(t *testing.T) (any, error) {
		tr, err := mArr(t).T()
		if err != nil {
			return nil, err
		}
		return tr.Ravel(), nil
	},
	"transpose": func(t *testing.T) (any, error) { return mArr(t).T() },
	"squeeze": func(t *testing.T) (any, error) {
		return pd.Ones(1, 3, 1).Squeeze()
	},
	"expand_dims": func(t *testing.T) (any, error) { return vArr().ExpandDims(0) },
	"concatenate_axis0": func(t *testing.T) (any, error) {
		return pd.Concatenate([]*pd.NDArray{sqArr(t), sqArr(t)}, 0)
	},
	"concatenate_axis1": func(t *testing.T) (any, error) {
		return pd.Concatenate([]*pd.NDArray{sqArr(t), sqArr(t)}, 1)
	},
	"stack_axis0": func(t *testing.T) (any, error) {
		return pd.Stack([]*pd.NDArray{vArr(), wArr()}, 0)
	},
	"hstack": func(t *testing.T) (any, error) {
		return pd.HStack([]*pd.NDArray{vArr(), wArr()})
	},
	"vstack": func(t *testing.T) (any, error) {
		return pd.VStack([]*pd.NDArray{vArr(), wArr()})
	},
	"astype_int": func(t *testing.T) (any, error) {
		return pd.Array([]float64{1.7, -2.7}).Astype(pd.Int64)
	},

	// broadcasting ---------------------------------------------------------------
	"scalar_add": arrCase(func(t *testing.T) *pd.NDArray { return vArr().AddScalar(10) }),
	"vector_to_matrix": func(t *testing.T) (any, error) {
		return mArr(t).Add(pd.Array([]float64{10, 20, 30}))
	},
	"col_plus_row": func(t *testing.T) (any, error) {
		col, err := pd.FromSlice([]float64{1, 2}, 2, 1)
		if err != nil {
			return nil, err
		}
		row, err := pd.FromSlice([]float64{10, 20, 30}, 1, 3)
		if err != nil {
			return nil, err
		}
		return col.Add(row)
	},
	"ones51_plus_arange6": func(t *testing.T) (any, error) {
		return pd.Ones(5, 1).Add(pd.Arange(6))
	},
	"big_shapes_sum": func(t *testing.T) (any, error) {
		sum, err := pd.Ones(8, 1, 6, 1).Add(pd.Ones(7, 1, 5))
		if err != nil {
			return nil, err
		}
		return sum.SumAll(), nil
	},
	"incompatible_3_4": func(t *testing.T) (any, error) {
		return vArr().Add(pd.Zeros(4))
	},
	"incompatible_43_4": func(t *testing.T) (any, error) {
		return pd.Zeros(4, 3).Add(pd.Zeros(4))
	},

	// ufuncs -------------------------------------------------------------------
	"abs":   arrCase(func(t *testing.T) *pd.NDArray { return pd.Abs(negArr()) }),
	"sqrt":  arrCase(func(t *testing.T) *pd.NDArray { return pd.Sqrt(pd.Array([]float64{1, 4, 9})) }),
	"exp":   arrCase(func(t *testing.T) *pd.NDArray { return pd.Exp(pd.Array([]float64{0, 1})) }),
	"log":   arrCase(func(t *testing.T) *pd.NDArray { return pd.Log(pd.Array([]float64{1, 2.718281828459045})) }),
	"log10": arrCase(func(t *testing.T) *pd.NDArray { return pd.Log10(pd.Array([]float64{1, 10, 100})) }),
	"sin":   arrCase(func(t *testing.T) *pd.NDArray { return pd.Sin(pd.Array([]float64{0, 1.5707963267948966})) }),
	"cos":   arrCase(func(t *testing.T) *pd.NDArray { return pd.Cos(pd.Array([]float64{0, 3.141592653589793})) }),
	"tan":   arrCase(func(t *testing.T) *pd.NDArray { return pd.Tan(pd.Array([]float64{0, 0.7853981633974483})) }),
	"floor": arrCase(func(t *testing.T) *pd.NDArray { return pd.Floor(negArr()) }),
	"ceil":  arrCase(func(t *testing.T) *pd.NDArray { return pd.Ceil(negArr()) }),
	"round": arrCase(func(t *testing.T) *pd.NDArray { return pd.Round(negArr()) }),
	"clip":  arrCase(func(t *testing.T) *pd.NDArray { return pd.Clip(negArr(), -2, 2) }),
	"isnan": func(t *testing.T) (any, error) { return pd.IsNaN(withNaNArr()), nil },
	"isfinite": func(t *testing.T) (any, error) {
		return pd.IsFinite(withNaNArr()), nil
	},
	"isinf": func(t *testing.T) (any, error) { return pd.IsInf(withNaNArr()), nil },
	"maximum": func(t *testing.T) (any, error) {
		return pd.Maximum(vArr(), pd.Array([]float64{2, 1, 4}))
	},
	"minimum": func(t *testing.T) (any, error) {
		return pd.Minimum(vArr(), pd.Array([]float64{2, 1, 4}))
	},
	"power": arrCase(func(t *testing.T) *pd.NDArray { return vArr().PowScalar(2) }),

	// reductions ------------------------------------------------------------------
	"sum_all":   func(t *testing.T) (any, error) { return mArr(t).SumAll(), nil },
	"sum_axis0": func(t *testing.T) (any, error) { return mArr(t).Sum(pd.Axis(0)) },
	"sum_axis1": func(t *testing.T) (any, error) { return mArr(t).Sum(pd.Axis(1)) },
	"mean_all":  func(t *testing.T) (any, error) { return mArr(t).MeanAll(), nil },
	"mean_axis0": func(t *testing.T) (any, error) {
		return mArr(t).Mean(pd.Axis(0))
	},
	"std_default": func(t *testing.T) (any, error) { return mArr(t).StdAll(), nil },
	"std_ddof1": func(t *testing.T) (any, error) {
		out, err := mArr(t).StdDDof(1)
		if err != nil {
			return nil, err
		}
		return out.MustAt(), nil
	},
	"var_default": func(t *testing.T) (any, error) { return mArr(t).VarAll(), nil },
	"var_axis1_ddof1": func(t *testing.T) (any, error) {
		return mArr(t).VarDDof(1, pd.Axis(1))
	},
	"min_all":   func(t *testing.T) (any, error) { return mArr(t).MinAll(), nil },
	"max_axis0": func(t *testing.T) (any, error) { return mArr(t).Max(pd.Axis(0)) },
	"argmin_all": func(t *testing.T) (any, error) {
		out, err := mArr(t).ArgMin()
		if err != nil {
			return nil, err
		}
		return out.MustAt(), nil
	},
	"argmax_axis1": func(t *testing.T) (any, error) {
		return mArr(t).ArgMax(pd.Axis(1))
	},

	// linalg ----------------------------------------------------------------------
	"dot_vectors": func(t *testing.T) (any, error) {
		out, err := pd.Dot(vArr(), wArr())
		if err != nil {
			return nil, err
		}
		return out.MustAt(), nil
	},
	"matvec": func(t *testing.T) (any, error) {
		return sqArr(t).Dot(pd.Array([]float64{5, 6}))
	},
	"matmul": func(t *testing.T) (any, error) {
		b, err := pd.Array2D([][]float64{{5, 6}, {7, 8}})
		if err != nil {
			return nil, err
		}
		return pd.MatMul(sqArr(t), b)
	},
	"trace": func(t *testing.T) (any, error) { return sqArr(t).Trace() },

	// indexing -----------------------------------------------------------------------
	"at_1_2":         func(t *testing.T) (any, error) { return mArr(t).At(1, 2) },
	"negative_index": func(t *testing.T) (any, error) { return mArr(t).At(-1, -1) },
	"slice_rows": func(t *testing.T) (any, error) {
		return mArr(t).Slice(pd.Slice(0, 1))
	},
	"slice_cols": func(t *testing.T) (any, error) {
		return mArr(t).Slice(pd.All(), pd.Slice(1, 3))
	},
	"slice_step": func(t *testing.T) (any, error) {
		return pd.Arange(10).Slice(pd.SliceStep(0, 10, 3))
	},
	"take_axis0": func(t *testing.T) (any, error) {
		return mArr(t).Take([]int{1, 0}, pd.Axis(0))
	},
	"mask_gt_2": func(t *testing.T) (any, error) {
		m := mArr(t)
		return m.Mask(m.GtScalar(2))
	},
	"where_scalar": func(t *testing.T) (any, error) {
		m := mArr(t)
		return pd.WhereScalar(m.GtScalar(2), m, 0)
	},
	"where_arrays": func(t *testing.T) (any, error) {
		half := wArr().DivScalar(2)
		mask, err := vArr().Gt(half)
		if err != nil {
			return nil, err
		}
		return pd.WhereArray(mask, vArr(), wArr())
	},
	"broadcast_to": func(t *testing.T) (any, error) {
		return vArr().BroadcastTo(2, 3)
	},

	// sorting -----------------------------------------------------------------------
	"sort_1d": arrCase(func(t *testing.T) *pd.NDArray {
		return pd.Array([]float64{3, 1, 2, 3, 1}).Sort()
	}),
	"sort_2d_last_axis": func(t *testing.T) (any, error) {
		m, err := pd.Array2D([][]float64{{3, 1, 2}, {9, 7, 8}})
		if err != nil {
			return nil, err
		}
		return m.Sort(), nil
	},
	"argsort_1d": arrCase(func(t *testing.T) *pd.NDArray {
		return pd.Array([]float64{3, 1, 2, 3, 1}).ArgSort()
	}),
	"unique": arrCase(func(t *testing.T) *pd.NDArray {
		return pd.Unique(pd.Array([]float64{3, 1, 2, 3, 1}))
	}),

	// random ------------------------------------------------------------------------
	"rand_2x3":  arrCase(func(t *testing.T) *pd.NDArray { return pd.Rand(2, 3) }),
	"randn_100": arrCase(func(t *testing.T) *pd.NDArray { return pd.Randn(100) }),
}

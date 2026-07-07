#!/usr/bin/env python3
"""Generate the NumPy golden suites in compat/goldens/numpy/.

    python3 compat/python/generate_numpy_goldens.py

Requires numpy. Case names must stay in sync with compat/compat_test.go.
"""
import json
import math
import os

import numpy as np

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "goldens", "numpy")


def ser_array(a):
    arr = np.asarray(a, dtype=float)
    flat = arr.ravel()
    data, nan_at = [], []
    for i, v in enumerate(flat):
        if math.isnan(v):
            data.append(None)
            nan_at.append(i)
        elif math.isinf(v):
            # goldens avoid inf except where explicitly tested
            data.append(None)
            nan_at.append(i)
        else:
            data.append(float(v))
    out = {"shape": list(arr.shape), "data": data}
    if nan_at:
        out["nan_at"] = nan_at
    return out


def ser_bool(a):
    arr = np.asarray(a, dtype=bool)
    return {"shape": list(arr.shape), "bool_data": [bool(v) for v in arr.ravel()]}


def ser_scalar(v):
    return {"scalar": float(v)}


def write(name, suite, cases):
    os.makedirs(OUT, exist_ok=True)
    path = os.path.join(OUT, name)
    with open(path, "w") as f:
        json.dump({"suite": suite, "numpy_version": np.__version__, "cases": cases}, f, indent=1)
        f.write("\n")
    print("wrote", path)


def case(name, operation, expected):
    return {"name": name, "operation": operation, "expected": expected}


# Shared fixtures — keep in sync with compat_test.go
M = np.arange(6, dtype=float).reshape(2, 3)          # [[0,1,2],[4,5,6... no: [3,4,5]]
V = np.array([1.0, 2.0, 3.0])
W = np.array([4.0, 5.0, 6.0])
SQ = np.array([[1.0, 2.0], [3.0, 4.0]])
NEG = np.array([-1.5, 2.5, -3.5])
WITH_NAN = np.array([1.0, float("nan"), 3.0, float("inf")])


def constructors():
    cases = [
        case("array_1d", "np.array([1,2,3])", ser_array(np.array([1.0, 2.0, 3.0]))),
        case("array_2d", "np.array([[1,2],[3,4]])", ser_array(SQ)),
        case("zeros_2x3", "np.zeros((2,3))", ser_array(np.zeros((2, 3)))),
        case("ones_2x3", "np.ones((2,3))", ser_array(np.ones((2, 3)))),
        case("full_7", "np.full((2,3), 7)", ser_array(np.full((2, 3), 7.0))),
        case("arange_0_10_2", "np.arange(0, 10, 2)", ser_array(np.arange(0, 10, 2))),
        case("linspace_0_1_5", "np.linspace(0, 1, 5)", ser_array(np.linspace(0, 1, 5))),
        case("logspace_0_2_3", "np.logspace(0, 2, 3)", ser_array(np.logspace(0, 2, 3))),
        case("eye_3", "np.eye(3)", ser_array(np.eye(3))),
        case("identity_3", "np.identity(3)", ser_array(np.identity(3))),
        case("diag", "np.diag([2,3])", ser_array(np.diag([2.0, 3.0]))),
    ]
    write("constructors.json", "numpy.constructors", cases)


def ndarray_core():
    a = np.arange(6, dtype=float)
    cases = [
        case("reshape_2x3", "a.reshape(2,3)", ser_array(a.reshape(2, 3))),
        case("reshape_infer", "a.reshape(3,-1)", ser_array(a.reshape(3, -1))),
        case("flatten", "m.flatten()", ser_array(M.flatten())),
        case("ravel_t", "m.T.ravel()", ser_array(M.T.ravel())),
        case("transpose", "m.T", ser_array(M.T)),
        case("squeeze", "np.squeeze(ones((1,3,1)))", ser_array(np.squeeze(np.ones((1, 3, 1))))),
        case("expand_dims", "np.expand_dims(v, 0)", ser_array(np.expand_dims(V, 0))),
        case("concatenate_axis0", "np.concatenate([sq, sq], 0)", ser_array(np.concatenate([SQ, SQ], axis=0))),
        case("concatenate_axis1", "np.concatenate([sq, sq], 1)", ser_array(np.concatenate([SQ, SQ], axis=1))),
        case("stack_axis0", "np.stack([v, w], 0)", ser_array(np.stack([V, W], axis=0))),
        case("hstack", "np.hstack([v, w])", ser_array(np.hstack([V, W]))),
        case("vstack", "np.vstack([v, w])", ser_array(np.vstack([V, W]))),
        case("astype_int", "np.array([1.7,-2.7]).astype(int)", ser_array(np.array([1.7, -2.7]).astype(int))),
    ]
    write("ndarray_core.json", "numpy.ndarray_core", cases)


def broadcasting():
    big = np.ones((8, 1, 6, 1)) + np.ones((7, 1, 5))
    cases = [
        case("scalar_add", "v + 10", ser_array(V + 10)),
        case("vector_to_matrix", "m + [10,20,30]", ser_array(M + np.array([10.0, 20.0, 30.0]))),
        case("col_plus_row", "(2,1) + (1,3)", ser_array(np.array([[1.0], [2.0]]) + np.array([[10.0, 20.0, 30.0]]))),
        case("ones51_plus_arange6", "np.ones((5,1)) + np.arange(6)", ser_array(np.ones((5, 1)) + np.arange(6))),
        case("big_shapes_sum", "((8,1,6,1)+(7,1,5)).sum()", ser_scalar(big.sum())),
        case("incompatible_3_4", "(3,) + (4,)", {"error": True}),
        case("incompatible_43_4", "(4,3) + (4,)", {"error": True}),
    ]
    write("broadcasting.json", "numpy.broadcasting", cases)


def ufuncs():
    cases = [
        case("abs", "np.abs(neg)", ser_array(np.abs(NEG))),
        case("sqrt", "np.sqrt([1,4,9])", ser_array(np.sqrt([1.0, 4.0, 9.0]))),
        case("exp", "np.exp([0,1])", ser_array(np.exp([0.0, 1.0]))),
        case("log", "np.log([1,e])", ser_array(np.log([1.0, math.e]))),
        case("log10", "np.log10([1,10,100])", ser_array(np.log10([1.0, 10.0, 100.0]))),
        case("sin", "np.sin([0, pi/2])", ser_array(np.sin([0.0, math.pi / 2]))),
        case("cos", "np.cos([0, pi])", ser_array(np.cos([0.0, math.pi]))),
        case("tan", "np.tan([0, pi/4])", ser_array(np.tan([0.0, math.pi / 4]))),
        case("floor", "np.floor(neg)", ser_array(np.floor(NEG))),
        case("ceil", "np.ceil(neg)", ser_array(np.ceil(NEG))),
        case("round", "np.round(neg)", ser_array(np.round(NEG))),
        case("clip", "np.clip(neg, -2, 2)", ser_array(np.clip(NEG, -2, 2))),
        case("isnan", "np.isnan(with_nan)", ser_bool(np.isnan(WITH_NAN))),
        case("isfinite", "np.isfinite(with_nan)", ser_bool(np.isfinite(WITH_NAN))),
        case("isinf", "np.isinf(with_nan)", ser_bool(np.isinf(WITH_NAN))),
        case("maximum", "np.maximum(v, [2,1,4])", ser_array(np.maximum(V, np.array([2.0, 1.0, 4.0])))),
        case("minimum", "np.minimum(v, [2,1,4])", ser_array(np.minimum(V, np.array([2.0, 1.0, 4.0])))),
        case("power", "np.power(v, 2)", ser_array(np.power(V, 2.0))),
    ]
    write("ufuncs.json", "numpy.ufuncs", cases)


def reductions():
    cases = [
        case("sum_all", "m.sum()", ser_scalar(M.sum())),
        case("sum_axis0", "m.sum(axis=0)", ser_array(M.sum(axis=0))),
        case("sum_axis1", "m.sum(axis=1)", ser_array(M.sum(axis=1))),
        case("mean_all", "m.mean()", ser_scalar(M.mean())),
        case("mean_axis0", "m.mean(axis=0)", ser_array(M.mean(axis=0))),
        case("std_default", "m.std()", ser_scalar(M.std())),
        case("std_ddof1", "m.std(ddof=1)", ser_scalar(M.std(ddof=1))),
        case("var_default", "m.var()", ser_scalar(M.var())),
        case("var_axis1_ddof1", "m.var(axis=1, ddof=1)", ser_array(M.var(axis=1, ddof=1))),
        case("min_all", "m.min()", ser_scalar(M.min())),
        case("max_axis0", "m.max(axis=0)", ser_array(M.max(axis=0))),
        case("argmin_all", "m.argmin()", ser_scalar(M.argmin())),
        case("argmax_axis1", "m.argmax(axis=1)", ser_array(M.argmax(axis=1))),
    ]
    write("reductions.json", "numpy.reductions", cases)


def linalg():
    cases = [
        case("dot_vectors", "np.dot(v, w)", ser_scalar(np.dot(V, W))),
        case("matvec", "sq.dot([5,6])", ser_array(SQ.dot(np.array([5.0, 6.0])))),
        case("matmul", "np.matmul(sq, [[5,6],[7,8]])", ser_array(np.matmul(SQ, np.array([[5.0, 6.0], [7.0, 8.0]])))),
        case("trace", "np.trace(sq)", ser_scalar(np.trace(SQ))),
    ]
    write("linalg.json", "numpy.linalg", cases)


def indexing():
    cases = [
        case("at_1_2", "m[1, 2]", ser_scalar(M[1, 2])),
        case("negative_index", "m[-1, -1]", ser_scalar(M[-1, -1])),
        case("slice_rows", "m[0:1]", ser_array(M[0:1])),
        case("slice_cols", "m[:, 1:3]", ser_array(M[:, 1:3])),
        case("slice_step", "np.arange(10)[0:10:3]", ser_array(np.arange(10.0)[0:10:3])),
        case("take_axis0", "np.take(m, [1, 0], 0)", ser_array(np.take(M, [1, 0], axis=0))),
        case("mask_gt_2", "m[m > 2]", ser_array(M[M > 2])),
        case("where_scalar", "np.where(m > 2, m, 0)", ser_array(np.where(M > 2, M, 0.0))),
        case("where_arrays", "np.where(v > w/2, v, w)", ser_array(np.where(V > W / 2, V, W))),
        case("broadcast_to", "np.broadcast_to(v, (2,3))", ser_array(np.broadcast_to(V, (2, 3)))),
    ]
    write("indexing.json", "numpy.indexing", cases)


def sorting():
    unsorted = np.array([3.0, 1.0, 2.0, 3.0, 1.0])
    m2 = np.array([[3.0, 1.0, 2.0], [9.0, 7.0, 8.0]])
    cases = [
        case("sort_1d", "np.sort(a)", ser_array(np.sort(unsorted))),
        case("sort_2d_last_axis", "np.sort(m2)", ser_array(np.sort(m2))),
        case("argsort_1d", "np.argsort(a)", ser_array(np.argsort(unsorted, kind="stable"))),
        case("unique", "np.unique(a)", ser_array(np.unique(unsorted))),
    ]
    write("sorting.json", "numpy.sorting", cases)


def random_suite():
    # Random output cannot match across languages: goldens record
    # properties (shape, range), not values.
    cases = [
        case("rand_2x3", "np.random.rand(2,3)", {"shape": [2, 3], "min": 0.0, "max": 1.0}),
        case("randn_100", "np.random.randn(100)", {"shape": [100]}),
    ]
    write("random.json", "numpy.random", cases)


def dtypes_suite():
    # Kind characters ('i' int, 'f' float, 'b' bool, 'U' unicode) keep the
    # comparison independent of bit-width naming.
    cases = [
        case("dtype_int_array", "np.array([1,2,3]).dtype.kind", {"kind": np.array([1, 2, 3]).dtype.kind}),
        case("dtype_float_array", "np.array([1.0,2.0]).dtype.kind", {"kind": np.array([1.0, 2.0]).dtype.kind}),
        case("dtype_bool_array", "np.array([True,False]).dtype.kind", {"kind": np.array([True, False]).dtype.kind}),
        case("dtype_string_array", "np.array(['a','b']).dtype.kind", {"kind": np.array(["a", "b"]).dtype.kind}),
        case("dtype_int_plus_float", "(int_arr + float_arr).dtype.kind", {"kind": (np.array([1, 2, 3]) + np.array([1.5, 2.5, 3.5])).dtype.kind}),
        case("dtype_int_plus_int", "(int_arr + int_arr).dtype.kind", {"kind": (np.array([1, 2]) + np.array([3, 4])).dtype.kind}),
        case("dtype_int_div_int", "(int_arr / int_arr).dtype.kind", {"kind": (np.array([1, 2]) / np.array([3, 4])).dtype.kind}),
        case("dtype_bool_plus_int", "(bool_arr + int_arr).dtype.kind", {"kind": (np.array([True, False]) + np.array([1, 2])).dtype.kind}),
        case("dtype_astype_int", "float_arr.astype(int).dtype.kind", {"kind": np.array([1.7, -2.7]).astype(int).dtype.kind}),
        case("dtype_abs_int", "np.abs(int_arr).dtype.kind", {"kind": np.abs(np.array([-1, 2])).dtype.kind}),
        case("dtype_sqrt_int", "np.sqrt(int_arr).dtype.kind", {"kind": np.sqrt(np.array([1, 4])).dtype.kind}),
    ]
    write("dtypes.json", "numpy.dtypes", cases)


def main():
    dtypes_suite()
    constructors()
    ndarray_core()
    broadcasting()
    ufuncs()
    reductions()
    linalg()
    indexing()
    sorting()
    random_suite()


if __name__ == "__main__":
    main()

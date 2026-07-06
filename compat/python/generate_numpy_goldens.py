#!/usr/bin/env python3
"""Regenerate the NumPy golden files used by compat_test.go.

Requires: numpy. Run from the repo root:

    python3 compat/python/generate_numpy_goldens.py
"""
import json
import math
import os

import numpy as np

OUT = os.path.join(os.path.dirname(__file__), "..", "goldens")


def expected(a) -> dict:
    arr = np.asarray(a, dtype=float)
    return {"shape": list(arr.shape), "data": [float(v) for v in arr.ravel()]}


def write(name: str, cases: list) -> None:
    path = os.path.join(OUT, name)
    with open(path, "w") as f:
        json.dump({"cases": cases}, f, indent=2)
        f.write("\n")
    print("wrote", path)


def basic_cases() -> list:
    return [
        {"name": "array_creation", "expected": expected(np.array([1.0, 2.0, 3.0]))},
        {"name": "zeros_2x3", "expected": expected(np.zeros((2, 3)))},
        {"name": "ones_3", "expected": expected(np.ones(3))},
        {"name": "arange_5", "expected": expected(np.arange(5))},
        {"name": "arange_2_10_2", "expected": expected(np.arange(2, 10, 2))},
        {"name": "linspace_0_1_5", "expected": expected(np.linspace(0, 1, 5))},
        {"name": "reshape_2x3", "expected": expected(np.arange(6).reshape(2, 3))},
        {"name": "transpose_2x3", "expected": expected(np.arange(6).reshape(2, 3).T)},
        {"name": "sum_all", "expected": expected(np.arange(6).sum())},
        {"name": "mean_all", "expected": expected(np.arange(6).mean())},
        {"name": "std_all", "expected": expected(np.arange(6).std())},
        {"name": "sqrt", "expected": expected(np.sqrt([1.0, 4.0, 9.0]))},
        {"name": "exp", "expected": expected(np.exp([0.0, 1.0]))},
        {"name": "log", "expected": expected(np.log([1.0, math.e]))},
    ]


def broadcast_cases() -> list:
    m = np.array([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])
    col = np.array([[1.0], [2.0]])
    row = np.array([[10.0, 20.0, 30.0]])
    return [
        {"name": "broadcast_scalar", "expected": expected(np.array([1.0, 2.0, 3.0]) + 10)},
        {"name": "broadcast_vector_to_matrix", "expected": expected(m + np.array([10.0, 20.0, 30.0]))},
        {"name": "broadcast_col_row", "expected": expected(col + row)},
        {"name": "broadcast_incompatible", "expected": {"error": True}},
    ]


def linalg_cases() -> list:
    m = np.array([[1.0, 2.0], [3.0, 4.0]])
    return [
        {"name": "dot_vectors", "expected": expected(np.dot([1.0, 2.0, 3.0], [4.0, 5.0, 6.0]))},
        {"name": "matvec", "expected": expected(m.dot([5.0, 6.0]))},
        {"name": "matmul_2x2", "expected": expected(np.matmul(m, np.array([[5.0, 6.0], [7.0, 8.0]])))},
    ]


def main() -> None:
    write("numpy_basic.json", basic_cases())
    write("numpy_broadcast.json", broadcast_cases())
    write("numpy_linalg.json", linalg_cases())


if __name__ == "__main__":
    main()

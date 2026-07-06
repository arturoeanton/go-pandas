#!/usr/bin/env python3
"""Regenerate the pandas golden files used by compat_test.go.

Requires: pandas. Run from the repo root:

    python3 compat/python/generate_pandas_goldens.py

The committed goldens were verified against pandas 2.x. Keep the case
names in sync with compat_test.go.
"""
import json
import math
import os

import pandas as pd

OUT = os.path.join(os.path.dirname(__file__), "..", "goldens")


def frame_expected(df: "pd.DataFrame") -> dict:
    rows = []
    for _, row in df.iterrows():
        cells = []
        for v in row.tolist():
            if v is None or (isinstance(v, float) and math.isnan(v)) or v is pd.NA or v is pd.NaT:
                cells.append(None)
            elif hasattr(v, "item"):
                cells.append(v.item())
            else:
                cells.append(v)
        rows.append(cells)
    return {"columns": [str(c) for c in df.columns], "rows": rows}


def write(name: str, cases: list) -> None:
    path = os.path.join(OUT, name)
    with open(path, "w") as f:
        json.dump({"cases": cases}, f, indent=2)
        f.write("\n")
    print("wrote", path)


def people() -> "pd.DataFrame":
    return pd.DataFrame(
        [
            {"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0},
            {"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0},
            {"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0},
        ],
        columns=["country", "name", "age", "salary"],
    )


def basic_cases() -> list:
    df = people()
    grades = pd.DataFrame(
        [
            {"name": "Ana", "math": 9.0, "bio": 8.0},
            {"name": "Luis", "math": 7.0, "bio": 6.0},
        ],
        columns=["name", "math", "bio"],
    )
    rolling = pd.DataFrame({"v": [1.0, 2.0, 3.0, 4.0, 5.0]})
    return [
        {"name": "head_2", "expected": frame_expected(df.head(2))},
        {"name": "tail_1", "expected": frame_expected(df.tail(1))},
        {"name": "select_name_salary", "expected": frame_expected(df[["name", "salary"]])},
        {"name": "filter_age_gt_30", "expected": frame_expected(df[df["age"] > 30])},
        {"name": "sort_salary_desc", "expected": frame_expected(df.sort_values("salary", ascending=False))},
        {
            "name": "assign_bonus",
            "expected": frame_expected(df.assign(bonus=df["salary"] * 0.1)[["name", "bonus"]]),
        },
        {
            "name": "melt",
            "expected": frame_expected(grades.melt(id_vars=["name"])),
        },
        {
            "name": "rolling_mean_3",
            "expected": frame_expected(rolling.rolling(3).mean()),
        },
    ]


def missing_cases() -> list:
    df = pd.DataFrame(
        [
            {"a": 1, "b": "x"},
            {"a": None, "b": "y"},
            {"a": 3, "b": None},
        ],
        columns=["a", "b"],
    )
    filled = df.fillna({"a": 0, "b": "?"})
    filled["a"] = filled["a"].astype(int)
    return [
        {"name": "isna", "expected": frame_expected(df.isna())},
        {"name": "dropna", "expected": frame_expected(df.dropna().astype({"a": int}))},
        {"name": "fillna", "expected": frame_expected(filled)},
    ]


def groupby_cases() -> list:
    df = people()
    agg = (
        df.groupby("country")
        .agg(age_max=("age", "max"), salary_mean=("salary", "mean"))
        .reset_index()
    )
    return [
        {
            "name": "groupby_sum",
            "expected": frame_expected(df.groupby("country")["salary"].sum().reset_index()),
        },
        {
            "name": "groupby_mean",
            "expected": frame_expected(df.groupby("country")["salary"].mean().reset_index()),
        },
        {
            "name": "groupby_count",
            "expected": frame_expected(df.groupby("country")["name"].count().reset_index()),
        },
        {"name": "groupby_agg", "expected": frame_expected(agg)},
    ]


def merge_cases() -> list:
    left = pd.DataFrame(
        [{"id": 1, "name": "Ana"}, {"id": 2, "name": "Luis"}, {"id": 3, "name": "Marta"}],
        columns=["id", "name"],
    )
    right = pd.DataFrame(
        [{"id": 1, "salary": 1000.0}, {"id": 2, "salary": 2000.0}, {"id": 4, "salary": 4000.0}],
        columns=["id", "salary"],
    )

    def norm_id(df: "pd.DataFrame") -> "pd.DataFrame":
        df = df.copy()
        df["id"] = df["id"].astype("Int64")
        return df

    concat = pd.concat(
        [
            pd.DataFrame([{"x": 1, "y": "a"}, {"x": 2, "y": "b"}], columns=["x", "y"]),
            pd.DataFrame([{"x": 3, "y": "c"}], columns=["x", "y"]),
        ],
        ignore_index=True,
    )
    return [
        {"name": "merge_inner", "expected": frame_expected(left.merge(right, on="id", how="inner"))},
        {"name": "merge_left", "expected": frame_expected(left.merge(right, on="id", how="left"))},
        {"name": "merge_outer", "expected": frame_expected(norm_id(left.merge(right, on="id", how="outer")))},
        {"name": "concat", "expected": frame_expected(concat)},
    ]


def main() -> None:
    write("pandas_basic.json", basic_cases())
    write("pandas_missing.json", missing_cases())
    write("pandas_groupby.json", groupby_cases())
    write("pandas_merge.json", merge_cases())


if __name__ == "__main__":
    main()

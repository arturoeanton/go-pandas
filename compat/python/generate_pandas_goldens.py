#!/usr/bin/env python3
"""Generate the pandas golden suites in compat/goldens/pandas/.

Run from anywhere:

    python3 compat/python/generate_pandas_goldens.py

Requires pandas. The committed goldens were generated with pandas 2.x.
Case names must stay in sync with compat/compat_test.go.
"""
import io
import json
import math
import os

import pandas as pd

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "goldens", "pandas")


def _cell(v):
    if v is None or v is pd.NA or v is pd.NaT:
        return None
    if isinstance(v, float) and math.isnan(v):
        return None
    if isinstance(v, pd.Timestamp):
        if v.hour == 0 and v.minute == 0 and v.second == 0:
            return v.strftime("%Y-%m-%d")
        return v.strftime("%Y-%m-%d %H:%M:%S")
    if hasattr(v, "item"):
        v = v.item()
    if isinstance(v, float) and math.isnan(v):
        return None
    return v


def ser_frame(df, with_index=False):
    out = {
        "columns": [str(c) for c in df.columns],
        "rows": [[_cell(v) for v in row] for row in df.itertuples(index=False, name=None)],
    }
    if with_index:
        out["index"] = [str(i) for i in df.index]
    return out


def ser_series(s, with_index=False):
    out = {"values": [_cell(v) for v in s.tolist()]}
    if with_index:
        out["index"] = [str(i) for i in s.index]
    return out


def ser_scalar(v):
    return {"scalar": float(v)}


def write(name, suite, cases):
    os.makedirs(OUT, exist_ok=True)
    path = os.path.join(OUT, name)
    with open(path, "w") as f:
        json.dump({"suite": suite, "pandas_version": pd.__version__, "cases": cases}, f, indent=1)
        f.write("\n")
    print("wrote", path)


def case(name, operation, expected):
    return {"name": name, "operation": operation, "expected": expected}


# Shared fixtures — keep in sync with compat_test.go -------------------------

def people():
    return pd.DataFrame(
        [
            {"country": "AR", "name": "Ana", "age": 30, "salary": 1000.0, "dept": "eng"},
            {"country": "AR", "name": "Luis", "age": 40, "salary": 2000.0, "dept": "sales"},
            {"country": "BR", "name": "Joao", "age": 35, "salary": 1500.0, "dept": "eng"},
            {"country": "BR", "name": "Bia", "age": 28, "salary": 1200.0, "dept": "eng"},
            {"country": "AR", "name": "Mia", "age": 22, "salary": 800.0, "dept": "sales"},
        ],
        columns=["country", "name", "age", "salary", "dept"],
    )


def missing_frame():
    return pd.DataFrame(
        {
            "a": [1, None, 3, None],
            "b": ["x", "y", None, None],
            "c": [1.5, 2.5, 3.5, None],
        },
        columns=["a", "b", "c"],
    )


def merge_frames():
    left = pd.DataFrame(
        [{"id": 1, "name": "Ana"}, {"id": 2, "name": "Luis"}, {"id": 3, "name": "Marta"}],
        columns=["id", "name"],
    )
    right = pd.DataFrame(
        [{"id": 1, "salary": 1000.0}, {"id": 2, "salary": 2000.0}, {"id": 4, "salary": 4000.0}],
        columns=["id", "salary"],
    )
    return left, right


def grades():
    return pd.DataFrame(
        [
            {"name": "Ana", "math": 9.0, "bio": 8.0},
            {"name": "Luis", "math": 7.0, "bio": 6.0},
        ],
        columns=["name", "math", "bio"],
    )


def num_series():
    return pd.Series([3.0, 1.0, 4.0, None, 5.0], name="v")


def int_series():
    return pd.Series([3, 1, 4, 1, 5], name="v")


def str_series():
    return pd.Series(["Hello", "world", " Go ", "Anaconda", None], name="s")


def date_series():
    return pd.to_datetime(pd.Series(
        ["2024-01-01", "2024-03-15 10:30:45", "2023-12-31", "2024-06-01"], name="d"),
        format="mixed")


def rolling_series():
    return pd.Series([1.0, 2.0, 3.0, 4.0, 5.0], name="v")


# Suites ------------------------------------------------------------------------

def dataframe_core():
    df = people()
    dup = pd.DataFrame(
        [{"a": 1, "b": "x"}, {"a": 1, "b": "x"}, {"a": 2, "b": "y"}],
        columns=["a", "b"],
    )
    cases = [
        case("head_2", "df.head(2)", ser_frame(df.head(2))),
        case("tail_2", "df.tail(2)", ser_frame(df.tail(2))),
        case("select_name_age", "df[['name','age']]", ser_frame(df[["name", "age"]])),
        case("drop_columns", "df.drop(columns=['dept','name'])", ser_frame(df.drop(columns=["dept", "name"]))),
        case("rename_age", "df.rename(columns={'age':'years'})", ser_frame(df.rename(columns={"age": "years"}))),
        case("sort_salary_desc", "df.sort_values('salary', ascending=False)", ser_frame(df.sort_values("salary", ascending=False))),
        case("sort_multi", "df.sort_values(['country','age'], ascending=[True,False])", ser_frame(df.sort_values(["country", "age"], ascending=[True, False]))),
        case("assign_bonus", "df.assign(bonus=df.salary*0.1)[['name','bonus']]", ser_frame(df.assign(bonus=df.salary * 0.1)[["name", "bonus"]])),
        case("query_age_salary", "df.query('age > 25 and salary < 1600')", ser_frame(df.query("age > 25 and salary < 1600"))),
        case("query_in", "df.query('country in [\"BR\"]')", ser_frame(df.query('country in ["BR"]'))),
        case("describe", "df[['age','salary']].describe().reset_index()", ser_frame(df[["age", "salary"]].describe().reset_index())),
        case("round_1", "(df[['salary']]/3).round(1)", ser_frame((df[["salary"]] / 3).round(1))),
        case("clip_age", "df[['age']].clip(25, 35)", ser_frame(df[["age"]].clip(25, 35))),
        case("duplicated", "dup.duplicated()", ser_series(dup.duplicated())),
        case("drop_duplicates", "dup.drop_duplicates()", ser_frame(dup.drop_duplicates())),
        case("nunique", "df.nunique()", ser_series(people().nunique(), with_index=True)),
        case("corr", "df[['age','salary']].corr().reset_index()", ser_frame(df[["age", "salary"]].corr().reset_index())),
        case("select_dtypes_number", "df.select_dtypes(include=['number'])", ser_frame(df.select_dtypes(include=["number"]))),
    ]
    write("dataframe_core.json", "pandas.dataframe_core", cases)


def series_core():
    s = num_series()
    si = int_series()
    neg = pd.Series([-1.5, 2.0, -3.0], name="n")
    cases = [
        case("head_3", "s.head(3)", ser_series(s.head(3))),
        case("astype_float", "si.astype('float64')", ser_series(si.astype("float64"))),
        case("isna", "s.isna()", ser_series(s.isna())),
        case("notna", "s.notna()", ser_series(s.notna())),
        case("dropna", "s.dropna()", ser_series(s.dropna())),
        case("fillna_0", "s.fillna(0)", ser_series(s.fillna(0))),
        case("unique", "si.unique()", {"values": [_cell(v) for v in si.unique().tolist()]}),
        case("nunique_series", "si.nunique()", ser_scalar(si.nunique())),
        case("value_counts", "si.value_counts()", ser_series(si.value_counts(), with_index=True)),
        case("sort_values", "s.sort_values()", ser_series(s.sort_values())),
        case("mean", "s.mean()", ser_scalar(s.mean())),
        case("median", "s.median()", ser_scalar(s.median())),
        case("std", "s.std()", ser_scalar(s.std())),
        case("var", "s.var()", ser_scalar(s.var())),
        case("quantile_25", "s.quantile(0.25)", ser_scalar(s.quantile(0.25))),
        case("sum", "s.sum()", ser_scalar(s.sum())),
        case("between_2_4", "si.between(2, 4)", ser_series(si.between(2, 4))),
        case("isin", "si.isin([1, 5])", ser_series(si.isin([1, 5]))),
        case("cumsum", "s.cumsum()", ser_series(s.cumsum())),
        case("cummax", "s.cummax()", ser_series(s.cummax())),
        case("cumprod", "si.cumprod()", ser_series(si.cumprod())),
        case("diff_1", "s.diff()", ser_series(s.diff())),
        case("pct_change_1", "s.pct_change()", ser_series(s.pct_change(fill_method=None))),
        case("rank_average", "si.rank()", ser_series(si.rank())),
        case("rank_dense", "si.rank(method='dense')", ser_series(si.rank(method="dense"))),
        case("clip_2_4", "si.clip(2, 4)", ser_series(si.clip(2, 4))),
        case("round_0", "s.round(0)", ser_series(s.round(0))),
        case("abs", "neg.abs()", ser_series(neg.abs())),
        case("shift_1", "si.shift(1)", ser_series(si.shift(1))),
    ]
    write("series_core.json", "pandas.series_core", cases)


def groupby():
    df = people()
    agg = df.groupby("country").agg(
        age_min=("age", "min"),
        salary_max=("salary", "max"),
        salary_mean=("salary", "mean"),
    ).reset_index()
    cases = [
        case("size", "df.groupby('country').size()", ser_frame(df.groupby("country").size().reset_index(name="size"))),
        case("count_name", "df.groupby('country')['name'].count()", ser_frame(df.groupby("country")["name"].count().reset_index())),
        case("sum_salary", "df.groupby('country')['salary'].sum()", ser_frame(df.groupby("country")["salary"].sum().reset_index())),
        case("mean_salary", "df.groupby('country')['salary'].mean()", ser_frame(df.groupby("country")["salary"].mean().reset_index())),
        case("median_salary", "df.groupby('country')['salary'].median()", ser_frame(df.groupby("country")["salary"].median().reset_index())),
        case("min_salary", "df.groupby('country')['salary'].min()", ser_frame(df.groupby("country")["salary"].min().reset_index())),
        case("max_salary", "df.groupby('country')['salary'].max()", ser_frame(df.groupby("country")["salary"].max().reset_index())),
        case("std_salary", "df.groupby('country')['salary'].std()", ser_frame(df.groupby("country")["salary"].std().reset_index())),
        case("mean_two_keys", "df.groupby(['country','dept'])['salary'].mean()", ser_frame(df.groupby(["country", "dept"])["salary"].mean().reset_index())),
        case("agg_named", "df.groupby('country').agg(age_min=..., salary_max=..., salary_mean=...)", ser_frame(agg)),
        case("nunique_dept", "df.groupby('country')['dept'].nunique()", ser_frame(df.groupby("country")["dept"].nunique().reset_index())),
        case("first", "df.groupby('country')[['name']].first()", ser_frame(df.groupby("country")[["name"]].first().reset_index())),
        case("last", "df.groupby('country')[['name']].last()", ser_frame(df.groupby("country")[["name"]].last().reset_index())),
    ]
    write("groupby.json", "pandas.groupby", cases)


def merge_join_concat():
    left, right = merge_frames()
    lr_left = pd.DataFrame([{"user_id": 1, "v": 10}, {"user_id": 2, "v": 20}], columns=["user_id", "v"])
    cross_l = pd.DataFrame([{"x": 1}, {"x": 2}], columns=["x"])
    cross_r = pd.DataFrame([{"y": "a"}, {"y": "b"}], columns=["y"])
    join_l = pd.DataFrame({"v": [1, 2]})
    join_r = pd.DataFrame({"w": [10, 20]})
    c1 = pd.DataFrame([{"x": 1, "y": "a"}, {"x": 2, "y": "b"}], columns=["x", "y"])
    c2 = pd.DataFrame([{"x": 3, "y": "c"}], columns=["x", "y"])
    c3 = pd.DataFrame([{"x": 4, "z": True}], columns=["x", "z"])
    outer = left.merge(right, on="id", how="outer")
    outer["id"] = outer["id"].astype("int64")
    cases = [
        case("merge_inner", "merge how=inner", ser_frame(left.merge(right, on="id", how="inner"))),
        case("merge_left", "merge how=left", ser_frame(left.merge(right, on="id", how="left"))),
        case("merge_right", "merge how=right", ser_frame(left.merge(right, on="id", how="right"))),
        case("merge_outer", "merge how=outer", ser_frame(outer)),
        case("merge_left_on_right_on", "merge left_on/right_on (right key dropped)", ser_frame(lr_left.merge(right, left_on="user_id", right_on="id").drop(columns=["id"]))),
        case("merge_cross", "merge how=cross", ser_frame(cross_l.merge(cross_r, how="cross"))),
        case("concat_rows", "pd.concat ignore_index", ser_frame(pd.concat([c1, c2], ignore_index=True))),
        case("concat_union", "pd.concat column union", ser_frame(pd.concat([c1, c3], ignore_index=True))),
        case("join_index", "left.join(right)", ser_frame(join_l.join(join_r))),
    ]
    write("merge_join_concat.json", "pandas.merge_join_concat", cases)


def reshape():
    g = grades()
    long = g.melt(id_vars=["name"])
    dup = pd.DataFrame(
        [
            {"country": "AR", "dept": "eng", "salary": 1000.0},
            {"country": "AR", "dept": "eng", "salary": 2000.0},
            {"country": "AR", "dept": "sales", "salary": 800.0},
            {"country": "BR", "dept": "eng", "salary": 1500.0},
        ],
        columns=["country", "dept", "salary"],
    )
    pt = dup.pivot_table(index="country", columns="dept", values="salary", aggfunc="mean", fill_value=0).reset_index()
    pt.columns.name = None
    pv = long.pivot(index="name", columns="variable", values="value").reset_index()
    pv.columns.name = None
    cases = [
        case("melt", "grades.melt(id_vars=['name'])", ser_frame(long)),
        case("melt_value_vars", "grades.melt(id_vars=['name'], value_vars=['math'])", ser_frame(g.melt(id_vars=["name"], value_vars=["math"]))),
        case("pivot", "long.pivot(index, columns, values)", ser_frame(pv)),
        case("pivot_table_mean", "dup.pivot_table(aggfunc=mean, fill_value=0)", ser_frame(pt)),
    ]
    write("reshape.json", "pandas.reshape", cases)


def missing_values():
    df = missing_frame()
    filled = df.fillna({"a": 0, "b": "?", "c": 0})
    cases = [
        case("isna_frame", "df.isna()", ser_frame(df.isna())),
        case("dropna_any", "df.dropna()", ser_frame(df.dropna())),
        case("dropna_all", "df.dropna(how='all')", ser_frame(df.dropna(how="all"))),
        case("dropna_thresh_2", "df.dropna(thresh=2)", ser_frame(df.dropna(thresh=2))),
        case("dropna_subset_a", "df.dropna(subset=['a'])", ser_frame(df.dropna(subset=["a"]))),
        case("fillna_map", "df.fillna({'a':0,'b':'?','c':0})", ser_frame(filled)),
        case("notna_series", "df['a'].notna()", ser_series(df["a"].notna())),
    ]
    write("missing_values.json", "pandas.missing_values", cases)


def datetime_suite():
    d = date_series()
    cases = [
        case("year", "s.dt.year", ser_series(d.dt.year)),
        case("month", "s.dt.month", ser_series(d.dt.month)),
        case("day", "s.dt.day", ser_series(d.dt.day)),
        case("hour", "s.dt.hour", ser_series(d.dt.hour)),
        case("minute", "s.dt.minute", ser_series(d.dt.minute)),
        case("second", "s.dt.second", ser_series(d.dt.second)),
        case("weekday", "s.dt.weekday", ser_series(d.dt.weekday)),
        case("dayofyear", "s.dt.dayofyear", ser_series(d.dt.dayofyear)),
        case("quarter", "s.dt.quarter", ser_series(d.dt.quarter)),
        case("is_month_start", "s.dt.is_month_start", ser_series(d.dt.is_month_start)),
        case("is_month_end", "s.dt.is_month_end", ser_series(d.dt.is_month_end)),
        case("is_year_start", "s.dt.is_year_start", ser_series(d.dt.is_year_start)),
        case("is_year_end", "s.dt.is_year_end", ser_series(d.dt.is_year_end)),
    ]
    write("datetime.json", "pandas.datetime", cases)


def string_accessor():
    s = str_series()
    cases = [
        case("contains_o", "s.str.contains('o')", ser_series(s.str.contains("o"))),
        case("lower", "s.str.lower()", ser_series(s.str.lower())),
        case("upper", "s.str.upper()", ser_series(s.str.upper())),
        case("len", "s.str.len()", ser_series(s.str.len())),
        case("strip", "s.str.strip()", ser_series(s.str.strip())),
        case("replace_l_L", "s.str.replace('l','L')", ser_series(s.str.replace("l", "L"))),
        case("startswith_A", "s.str.startswith('A')", ser_series(s.str.startswith("A"))),
        case("endswith_d", "s.str.endswith('d')", ser_series(s.str.endswith("d"))),
        case("get_0", "s.str.get(0)", ser_series(s.str.get(0))),
        case("slice_1_3", "s.str.slice(1, 3)", ser_series(s.str.slice(1, 3))),
    ]
    write("string_accessor.json", "pandas.string_accessor", cases)


def rolling():
    s = rolling_series()
    df = pd.DataFrame({"open": [1.0, 2.0, 3.0, 4.0], "close": [2.0, 3.0, 4.0, 5.0]})
    cases = [
        case("rolling_mean_3", "s.rolling(3).mean()", ser_series(s.rolling(3).mean())),
        case("rolling_sum_3", "s.rolling(3).sum()", ser_series(s.rolling(3).sum())),
        case("rolling_min_periods_1", "s.rolling(3, min_periods=1).mean()", ser_series(s.rolling(3, min_periods=1).mean())),
        case("rolling_std_3", "s.rolling(3).std()", ser_series(s.rolling(3).std())),
        case("rolling_median_3", "s.rolling(3).median()", ser_series(s.rolling(3).median())),
        case("rolling_max_2", "s.rolling(2).max()", ser_series(s.rolling(2).max())),
        case("expanding_mean", "s.expanding().mean()", ser_series(s.expanding().mean())),
        case("expanding_sum", "s.expanding().sum()", ser_series(s.expanding().sum())),
        case("df_rolling_mean_2", "df.rolling(2).mean()", ser_frame(df.rolling(2).mean())),
    ]
    write("rolling.json", "pandas.rolling", cases)


def io_suite():
    basic_csv = "name,age,score\nAna,30,9.5\nLuis,40,8.0\n"
    na_csv = "a,b\n1,x\nNA,y\n3,\n"
    semi_csv = "a;b\n1;x\n2;y\n"
    usecols_csv = "a,b,c\n1,2,3\n4,5,6\n"
    dates_csv = "day,v\n2024-01-02,1\n2024-02-03,2\n"
    cases = [
        case("read_csv_basic", "pd.read_csv(basic)", ser_frame(pd.read_csv(io.StringIO(basic_csv)))),
        case("read_csv_na", "pd.read_csv(na)", ser_frame(pd.read_csv(io.StringIO(na_csv)))),
        case("read_csv_semicolon", "pd.read_csv(sep=';')", ser_frame(pd.read_csv(io.StringIO(semi_csv), sep=";"))),
        case("read_csv_usecols", "pd.read_csv(usecols=['a','c'])", ser_frame(pd.read_csv(io.StringIO(usecols_csv), usecols=["a", "c"]))),
        case("read_csv_nrows", "pd.read_csv(nrows=1)", ser_frame(pd.read_csv(io.StringIO(usecols_csv), nrows=1))),
        case("read_csv_parse_dates", "pd.read_csv(parse_dates=['day'])", ser_frame(pd.read_csv(io.StringIO(dates_csv), parse_dates=["day"]))),
        case("read_csv_no_header", "pd.read_csv(header=None)", ser_frame(pd.read_csv(io.StringIO("1,x\n2,y\n"), header=None, names=["column_0", "column_1"]))),
    ]
    write("io.json", "pandas.io", cases)


def dtypes_suite():
    # go-pandas models missing integers with a mask, matching pandas'
    # NULLABLE dtypes ("Int64"), not the classic float64 coercion. The
    # nullable spellings are used here on purpose; see known_differences.
    int_na = pd.Series([1, None, 3], dtype="Int64")
    df_na = pd.DataFrame({"age": pd.array([1, None, 3], dtype="Int64")})
    cases = [
        case("dtype_series_int", "pd.Series([1,2,3]).dtype.kind", {"kind": pd.Series([1, 2, 3]).dtype.kind}),
        case("dtype_series_int_na", "pd.Series([1,None,3], dtype='Int64').dtype.kind", {"kind": int_na.dtype.kind}),
        case("dtype_series_mixed_na", "pd.Series([1,2.5,None]).dtype.kind", {"kind": pd.Series([1, 2.5, None]).dtype.kind}),
        case("dtype_series_bool", "pd.Series([True,False]).dtype.kind", {"kind": pd.Series([True, False]).dtype.kind}),
        case("dtype_series_string", "pd.Series(['a','b']).dtype.kind", {"kind": pd.Series(["a", "b"]).dtype.kind}),
        case("dtype_frame_int_na", "df['age'].dtype.kind (nullable)", {"kind": df_na["age"].dtype.kind}),
        case("dtype_astype_float", "df.astype({'age':'float64'})['age'].dtype.kind", {"kind": df_na.astype({"age": "float64"})["age"].dtype.kind}),
        case("dtype_to_datetime", "pd.to_datetime(s).dtype.kind", {"kind": pd.to_datetime(pd.Series(["2024-01-02"])).dtype.kind}),
    ]
    write("dtypes.json", "pandas.dtypes", cases)


def expressions_suite():
    df = people()
    shop = pd.DataFrame(
        [
            {"item": "pen", "price": 1.5, "qty": 10},
            {"item": "book", "price": 12.0, "qty": 2},
            {"item": "mug", "price": 7.25, "qty": 4},
        ],
        columns=["item", "price", "qty"],
    )
    cases = [
        case("expr_filter_gt", "df[df['age'] > 30]", ser_frame(df[df["age"] > 30])),
        case("expr_filter_and", "df[(df['age'] > 30) & (df['salary'] < 2000)]", ser_frame(df[(df["age"] > 30) & (df["salary"] < 2000)])),
        case("expr_filter_or_not", "df[(df['age'] >= 40) | ~(df['dept'] == 'eng')]", ser_frame(df[(df["age"] >= 40) | ~(df["dept"] == "eng")])),
        case("expr_filter_contains", "df[df['name'].str.contains('a')]", ser_frame(df[df["name"].str.contains("a")])),
        case("expr_filter_isin", "df[df['country'].isin(['BR'])]", ser_frame(df[df["country"].isin(["BR"])])),
        case("expr_assign_total", "shop.assign(total=price*qty)", ser_frame(shop.assign(total=shop["price"] * shop["qty"]))),
        case("expr_assign_flag", "df.assign(flag=age>30)[['name','flag']]", ser_frame(df.assign(flag=df["age"] > 30)[["name", "flag"]])),
        case("expr_assign_ratio", "shop.assign(r=price/qty)[['item','r']]", ser_frame(shop.assign(r=shop["price"] / shop["qty"])[["item", "r"]])),
        case("expr_query_gt", "df.query('age > 30')", ser_frame(df.query("age > 30"))),
        case("expr_query_and", "df.query('age > 30 and salary < 2000')", ser_frame(df.query("age > 30 and salary < 2000"))),
    ]
    write("expressions.json", "pandas.expressions", cases)


def main():
    dtypes_suite()
    expressions_suite()
    dataframe_core()
    series_core()
    groupby()
    merge_join_concat()
    reshape()
    missing_values()
    datetime_suite()
    string_accessor()
    rolling()
    io_suite()


if __name__ == "__main__":
    main()

# pandas → go-pandas translation guide

Each snippet shows the Python idiom and its direct Go equivalent. Go
returns errors where Python raises; everything else maps one-to-one.

## Constructors

```python
df = pd.DataFrame([
    {"name": "Ana", "age": 30},
    {"name": "Luis", "age": 40},
])
```

```go
df, err := pd.DataFrameFromRecords([]map[string]any{
    {"name": "Ana", "age": 30},
    {"name": "Luis", "age": 40},
})
```

Go maps are unordered: pass `pd.WithColumnOrder("name", "age")` to fix the
column order (otherwise it is alphabetical).

Other constructors: `pd.DataFrameFromMap`, `pd.DataFrameFromRows`,
`pd.DataFrameFromStructs` (uses `pd:"name"` field tags), `pd.NewDataFrame(series...)`.

## Column selection

```python
df["age"]
df[["name", "age"]]
```

```go
age, err := df.Col("age")          // Column() is an alias
small, err := df.Select("name", "age")
```

## Boolean filtering

```python
df[df["age"] > 30]
df[(df.country == "AR") & (df.salary > 1000)]
```

```go
filtered, err := df.Where(pd.Col("age").Gt(30))
both, err := df.Where(pd.And(
    pd.Col("country").Eq("AR"),
    pd.Col("salary").Gt(1000),
))
```

Or with a query string:

```go
out, err := df.Query(`age > 30 and country in ["AR", "BR"]`)
out, err = df.Query(`name.str.contains("Ana") or not active`)
```

## Assign

```python
df["total"] = df["price"] * df["qty"]
```

```go
df2, err := df.AssignExpr("total", pd.Col("price").Mul(pd.Col("qty")))
```

Also: `AssignValue` (scalar), `Assign` (series), `AssignFunc` (row func).

## loc / iloc

```python
df.iloc[0:10, 1:3]
df.loc["a":"d", ["name", "age"]]
```

```go
out, err := df.ILoc().Rows(pd.Slice(0, 10)).Cols(pd.Slice(1, 3)).Get()
out, err = df.ILoc().Rows(0, 2, 4).Get()                  // explicit positions
out, err = df.Loc().Rows(pd.LabelSlice("a", "d")).Cols("name", "age").Get()
```

Positional slices are Go-style `[start, stop)`. Label slices
(`pd.LabelSlice`) are **inclusive** on both ends, like pandas.

## GroupBy

```python
df.groupby("country")["salary"].mean()
df.groupby("country").agg(salary_mean=("salary", "mean"), age_max=("age", "max"))
```

```go
out, err := df.GroupBy("country").Mean("salary")
out, err = df.GroupBy("country").Agg(map[string]string{
    "salary": "mean",
    "age":    "max",
})
// several aggregations per column:
out, err = df.GroupBy("country").AggList(map[string][]string{
    "salary": {"mean", "max"},
})
```

Output columns are named `column_agg` (`salary_mean`, `age_max`). Group
keys sort ascending by default (`pd.GroupSort(false)` to disable); NA keys
drop by default (`pd.GroupDropNA(false)` to keep).

## Merge / join / concat

```python
pd.merge(left, right, on="id", how="left")
pd.concat([df1, df2], ignore_index=True)
left.join(right)
```

```go
out, err := pd.Merge(left, right, pd.MergeOptions{
    On:  []string{"id"},
    How: "left", // inner, left, right, outer, cross
})
out, err = pd.Concat([]*pd.DataFrame{df1, df2}, pd.IgnoreIndex(true))
out, err = left.Join(right, pd.JoinOptions{})
```

`MergeOptions` also supports `LeftOn`/`RightOn`, `Suffixes`, `Validate`
("one_to_one", ...) and `Indicator` (adds the `_merge` column).

## Missing values

```python
df.isna()
df.dropna(thresh=2)
df.fillna({"age": 0})
```

```go
df.IsNA()
df.DropNA(pd.DropNAThresh(2))
df.FillNA(map[string]any{"age": 0})
```

See [missing_values.md](missing_values.md) for the full model.

## Reshape

```python
df.melt(id_vars=["id"])
df.pivot(index="id", columns="metric", values="value")
df.pivot_table(index="country", columns="dept", values="salary", aggfunc="mean")
```

```go
out, err := df.Melt(pd.MeltOptions{IDVars: []string{"id"}})
out, err = df.Pivot(pd.PivotOptions{Index: "id", Columns: "metric", Values: "value"})
out, err = df.PivotTable(pd.PivotTableOptions{
    Index: []string{"country"}, Columns: []string{"dept"},
    Values: []string{"salary"}, AggFunc: "mean", FillValue: 0.0,
})
```

## Rolling / expanding

```python
s.rolling(3, min_periods=1).mean()
df.expanding().sum()
```

```go
out, err := s.Rolling(3, pd.MinPeriods(1)).Mean()
out, err = df.Expanding().Sum()
```

## Series methods

```python
s.astype("float64")     s.Astype(pd.Float64)
s.value_counts()        s.ValueCounts()
s.rank(method="dense")  s.Rank(pd.RankMethod("dense"))
s.diff()                s.Diff(1)
s.pct_change()          s.PctChange(1)
s.cumsum()              s.Cumsum()
s.clip(0, 10)           s.Clip(0, 10)
s.str.upper()           s.Str().Upper()
s.str.contains("x")     s.Str().Contains("x")       // regex: ContainsRegex
s.dt.year               s.Dt().Year()
s.dt.quarter            s.Dt().Quarter()
pd.to_datetime(s)       pd.ToDatetime(s)
```

## Categorical (v0.7)

```python
s.astype("category")                       s.Astype(pd.Category)
pd.Categorical(v, categories=c, ordered=True)
                                           pd.CategoricalSeries("s", v,
                                               pd.WithCategories(...), pd.WithOrdered(true))
s.cat.categories / s.cat.codes             s.Cat().Categories() / s.Cat().Codes()
s.cat.rename_categories({"s": "small"})    s.Cat().RenameCategories(map[any]any{"s": "small"})
s.cat.reorder_categories(c, ordered=True)  s.Cat().ReorderCategories(c, true)
s.cat.set_categories(c)                    s.Cat().SetCategories(c, false)
s > "m"  (ordered)                         s.Gt("m")  // by category rank
pd.read_csv(f, dtype={"c": "category"})    pd.ReadCSV(f, pd.WithCategorical("c"))
```

See [categorical.md](categorical.md) for storage, fast paths and the
documented differences.

## IO

```python
pd.read_csv("f.csv", usecols=["a"], nrows=100, parse_dates=["day"])
df.to_json("f.json", orient="records")
```

```go
df, err := pd.ReadCSV("f.csv",
    pd.WithUseCols("a"), pd.WithNRows(100), pd.WithParseDates("day"))
err = df.ToJSON("f.json", pd.JSONOrient("records")) // records, split, columns, values
```

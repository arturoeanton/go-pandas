# pandas compatibility matrix

Statuses: `done`, `partial`, `planned`, `not_supported`.

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| pd.DataFrame(...) | pd.NewDataFrame / pd.DataFrameFromRecords / pd.DataFrameFromMap / pd.DataFrameFromRows / pd.DataFrameFromStructs | partial | constructor variants; map-based constructors sort columns unless WithColumnOrder |
| pd.Series(...) | pd.NewSeries / pd.SeriesOf / typed constructors | partial | |
| df.head(n) | df.Head(n) | done | |
| df.tail(n) | df.Tail(n) | done | |
| df.shape | df.Shape() | done | |
| df.columns | df.Columns() | done | |
| df.dtypes | df.DTypes() | done | |
| df.index | df.Index() | done | |
| df.values | df.Values() / df.ToRows() | done | |
| df["col"] | df.Col("col") | done | Go syntax limitation |
| df[["a","b"]] | df.Select("a","b") | done | |
| df[df.x > 1] | df.Where(pd.Col("x").Gt(1)) / df.Filter(mask) | done | Go syntax limitation |
| df.loc[...] | df.Loc().Rows(...).Cols(...).Get() | partial | labels + inclusive RowsBetween; no boolean loc |
| df.iloc[...] | df.ILoc().Rows(...).Cols(...).Get() | partial | positive steps only |
| df.query(...) | df.Query("age > 30 and ...") | partial | comparison, and/or/not, in [..] |
| df.assign(...) | df.Assign / df.AssignValue / df.AssignFunc / df.AssignExpr | done | |
| df.drop(columns=...) | df.Drop(...) | done | column drop only |
| df.rename(columns=...) | df.Rename(...) | done | |
| df.set_index(...) | df.SetIndex(...) | partial | string labels |
| df.reset_index() | df.ResetIndex() | done | |
| df.sort_values(...) | df.SortValues / df.SortValuesBy | done | NA last, stable |
| df.sort_index() | df.SortIndex(...) | done | |
| df.isna() | df.IsNA() | done | |
| df.notna() | df.NotNA() | done | |
| df.dropna() | df.DropNA(...) | done | how=any/all, subset |
| df.fillna(...) | df.FillNA(map) | done | per-column values |
| df.describe() | df.Describe() | done | numeric only in v0.1 |
| df.info() | df.Info() | done | |
| df.count()/sum()/mean()/median()/min()/max()/var()/std()/quantile() | df.Count()/Sum()/... | done | skipna default; ddof=1 for var/std |
| df.apply(...) | df.Apply(axis, fn) | partial | |
| df.applymap / df.map | df.Map(fn) | done | |
| df.pipe(...) | df.Pipe(fn) | done | |
| df.sample(...) | df.Sample(n, ...) | partial | without replacement |
| df.copy() | df.Copy() | done | |
| df.groupby(...) | df.GroupBy(...) / df.GroupByOpts(...) | done | hash grouping, sorted keys by default |
| gb.count()/size()/sum()/mean()/median()/min()/max()/var()/std()/first()/last() | gb.Count()/... | done | |
| gb.agg(...) | gb.Agg(map) / gb.AggList(map) | partial | named col_agg output columns |
| gb.apply(...) | gb.Apply(fn) | partial | |
| df.merge(...) | df.Merge(...) | done | inner/left/right/outer/cross, suffixes, validate, indicator |
| pd.merge(...) | pd.Merge(...) | done | |
| pd.concat(...) | pd.Concat(...) | partial | axis 0/1, outer/inner join, ignore_index |
| df.join(...) | df.Join(...) | partial | index join; left/inner/outer |
| df.melt(...) | df.Melt(...) | done | |
| df.pivot(...) | df.Pivot(...) | partial | single index/columns/values |
| df.pivot_table(...) | df.PivotTable(...) | partial | single value + single aggfunc |
| df.stack()/unstack() | df.Stack()/df.Unstack() | not_supported | returns ErrNotImplemented |
| df.resample(...) | df.Resample(...) | not_supported | returns ErrNotImplemented |
| df.rolling(...) | df.Rolling(w, ...) | partial | fixed windows, numeric columns |
| s.rolling(...) | s.Rolling(w, ...) | partial | min_periods, center |
| s.expanding() | s.Expanding() | partial | sum/mean |
| pd.read_csv(...) | pd.ReadCSV(...) | done | header, sep, dtype inference, na_values, parse_dates, nrows |
| df.to_csv(...) | df.ToCSV / df.WriteCSV | done | |
| pd.read_json(...) | pd.ReadJSON(...) | partial | records + values orientation |
| df.to_json(...) | df.ToJSON / df.WriteJSON | partial | records + values orientation |
| pd.read_json(lines=True) | pd.ReadNDJSON(...) | done | |
| s.astype(...) | s.Astype(dt) | done | |
| s.isin(...) | s.IsIn(...) | done | |
| s.between(...) | s.Between(l, r, inclusive) | done | |
| s.value_counts() | s.ValueCounts(...) | partial | returns a Series (labels as index) |
| s.unique()/nunique() | s.Unique() / s.NUnique(dropNA) | done | |
| s.str.* | s.Str().Contains/Lower/Upper/Len/Strip/Replace/Split/HasPrefix/HasSuffix | partial | |
| s.dt.* | s.Dt().Year/Month/Day/Hour/Minute/Second/Weekday | partial | Weekday: Monday=0 |
| s.describe() | s.Describe() | partial | returns a labeled Series |
| pd.NA / pd.NaT / pd.isna | pd.NA() / pd.NaT() / pd.IsNA(v) | done | nil and NaN are also missing |
| pd.MultiIndex.from_arrays | pd.NewMultiIndexFromArrays | partial | construction + display only |
| pd.Categorical | — | planned | v0.3 |
| pd.set_option("display...") | pd.SetDisplayOptions(...) | partial | MaxRows/MaxCols/Width/Precision |

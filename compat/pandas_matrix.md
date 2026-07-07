# pandas compatibility matrix

Statuses: `done`, `partial`, `planned`, `not_supported`.
Differences in behavior are documented in [known_differences.md](known_differences.md).

## Constructors and core attributes

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| pd.DataFrame(...) | pd.NewDataFrame / DataFrameFromRecords / FromMap / FromRows / FromStructs | partial | map constructors sort columns unless WithColumnOrder |
| pd.Series(...) | pd.NewSeries / pd.SeriesOf / typed constructors | partial | |
| df.shape / len(df) | df.Shape() / df.Len() | done | |
| df.columns / df.dtypes / df.index | df.Columns() / df.DTypes() / df.Index() | done | |
| df.values / df.to_dict("records") | df.Values() / df.ToRecords() | done | |
| df.to_numpy() | df.ToNDArray(...) | done | numeric columns |
| df.copy() | df.Copy() | done | |
| df.head / df.tail | df.Head(n) / df.Tail(n) | done | |
| df.info() / df.describe() | df.Info() / df.Describe() | done | describe: numeric only |

## Selection and indexing

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| df["col"] | df.Col("col") (alias Column) | done | |
| df[["a","b"]] | df.Select("a","b") | done | |
| df[df.x > 1] | df.Where(pd.Col("x").Gt(1)) / df.Filter(mask) | done | columnar engine (v0.4) |
| df.query(...) | df.Query("age > 30 and c in [...] and name.str.contains(...)") | partial | columnar execution (v0.4); comparisons, and/or/not, in, str accessor, bare bool columns |
| df.eval(...) | — | planned | expression API covers the use case |
| df.loc[...] | df.Loc().Rows(...)/RowsBetween/Cols(...).Get() | partial | inclusive pd.LabelSlice |
| df.iloc[...] | df.ILoc().Rows(ints/slices).Cols(...).Get() | partial | Go-style [start:stop); no negative step |
| s.iloc[i] / s.loc[l] / s.iat / s.at | s.ILoc(i) / s.Loc(l) / s.IAt(i) / s.AtLabel(l) | done | |
| df.set_index("c") | df.SetIndex("c") | done | single column keeps historical simple index |
| df.set_index([c1, c2]) | df.SetIndex("c1", "c2") | done | real MultiIndex, levels+codes (v0.8) |
| df.reset_index() | df.ResetIndex() | done | MultiIndex levels become leading columns (v0.8) |
| df.reindex(index=...) | df.Reindex(idx) | done | |
| df.reindex(columns=...) | df.ReindexColumns(...) | done | |
| pd.MultiIndex.from_arrays | pd.NewMultiIndexFromArrays / pd.MultiIndexFromArrays | done | real levels+codes storage (v0.8) |
| pd.MultiIndex.from_tuples | pd.MultiIndexFromTuples | done | pd.Tuple components; nil = NA |
| mi.names / mi.levels / mi.codes | mi.Names() / mi.Levels() / mi.Codes() | done | sorted unique levels, -1 = NA (pandas parity) |
| df.loc[(a, b)] | df.Loc().Tuple(a, b).Get() | done | lazy lookup map; duplicates return all rows |
| df.loc[(a, slice(None))] | df.Loc().TuplePrefix(a).Get() | partial | prefix scan (v0.8); no label-range slicing |
| MultiIndex take/slice | index.Take / mi.SlicePos | done | typed code gather; levels not compacted |
| groupby(as_index=True) | GroupBy(...).AsIndex(true) | partial | go-pandas defaults to as_index=false (documented) |
| concat with MultiIndex | pd.Concat | partial | same level count stacks; mixed shapes -> boxed tuples |
| merge/join on MultiIndex levels | — | planned | join BY index works via boxed tuple alignment |
| mi.swaplevel / droplevel / xs | — | planned | |

## Mutation and transforms

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| df.assign(...) | df.Assign / AssignValue / AssignFunc / AssignExpr | done | AssignExpr columnar + typed result (v0.4) |
| df.drop(columns=...) | df.Drop(...) | done | |
| df.rename(columns=...) | df.Rename(map) | done | |
| df.sort_values(...) | df.SortValues / df.SortValuesBy | done | NA last, stable |
| df.sort_index() | df.SortIndex(asc) | done | |
| df.drop_duplicates(...) | df.DropDuplicates(subset...) | done | keep="first" |
| df.duplicated(...) | df.Duplicated(subset...) | done | |
| df.nunique() | df.NUnique(...) | done | returns map |
| df.value_counts() | df.ValueCounts(...) | done | |
| df.corr() / df.cov() | df.Corr() / df.Cov() | done | pairwise-complete, ddof=1 |
| df.clip / df.round / df.abs | df.Clip / df.Round / df.Abs | done | numeric columns; banker's rounding |
| df.astype({...}) | df.Astype(map) | done | |
| df.select_dtypes(...) | df.SelectDTypes(pd.Include/pd.Exclude) | done | pd.Number pseudo-dtype |
| df.apply / df.map / df.pipe | df.Apply(axis, fn) / df.Map(fn) / df.Pipe(fn) | partial | |
| df.sample(...) | df.Sample(n, ...) | partial | without replacement |

## Missing values

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| pd.NA / pd.NaT / pd.isna / pd.notna | pd.NA() / pd.NaT() / pd.IsNA / pd.NotNA (+IsNull/NotNull) | done | |
| df.isna() / df.notna() | df.IsNA() / df.NotNA() | done | |
| df.dropna(how/subset/thresh/axis) | df.DropNA(DropNAHow/Subset/Thresh/Axis) | done | |
| df.fillna(...) | df.FillNA(map) / df.ReplaceNA(v) | done | |
| s.isna/dropna/fillna | s.IsNA()/s.DropNA()/s.FillNA(v)/s.ReplaceNA(v) | done | |

## Series

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| s.astype(...) | s.Astype(dt) | done | pd.ParseDType for names |
| s.astype("category") | s.Astype(pd.Category) | done | typed int32 codes; sorted default categories (v0.7) |
| pd.Categorical(...) / CategoricalDtype | pd.CategoricalSeries / NewCategoricalSeries + WithCategories/WithOrdered | done | explicit categories are strict |
| s.cat.categories / codes / ordered | cat.Categories()/Codes()/Ordered() | done | cat, err := s.Cat() — errors on non-categorical |
| s.cat.rename_categories | cat.RenameCategories(map) | done | keeps codes |
| s.cat.reorder_categories | cat.ReorderCategories(list, ordered) | done | must keep the set |
| s.cat.set_categories | cat.SetCategories(list, ordered) | done | removed categories become NA |
| s.cat.add_categories / remove_categories | cat.AddCategories / RemoveCategories | done | |
| ordered comparisons (s > 'm') | s.Gt / cat.Gt / expr Col().Gt | done | by category rank; unordered -> ErrInvalidOperation (accessor/expr), Series methods all-false |
| s.unique()/nunique() | s.Unique() / s.NUnique(dropNA) | done | |
| s.value_counts() | s.ValueCounts(...) | done | returns Series |
| s.sort_values()/sort_index() | s.SortValues(asc)/s.SortIndex(asc) | done | |
| s.argsort() | s.Argsort() | done | |
| s.rank(...) | s.Rank(RankMethod/RankAscending) | done | average/min/max/first/dense |
| s.diff()/pct_change() | s.Diff(p) / s.PctChange(p) | done | |
| s.cumsum/cumprod/cummin/cummax | s.Cumsum()/Cumprod()/Cummin()/Cummax() | done | |
| s.clip/round/abs | s.Clip(lo,hi)/s.Round(d)/s.Abs() | done | banker's rounding |
| s.shift(p) | s.Shift(p) | done | |
| s.between/isin | s.Between(l,r,incl) / s.IsIn(...) | done | |
| s.mean/median/std/var/quantile/sum/count/min/max | same names | done | skipna default; ddof=1 |
| s.reindex(...) | s.Reindex(idx) | done | |
| s.describe() | s.Describe() | partial | returns labeled Series |
| pd.to_datetime(s) | pd.ToDatetime(s) | partial | common layouts, no format arg |

## String and datetime accessors

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| s.str.contains | s.Str().Contains / ContainsRegex | done | |
| s.str.match | s.Str().Match(pattern) | done | |
| s.str.lower/upper/len/strip | same names | done | |
| s.str.replace | s.Str().Replace / ReplaceRegex | done | |
| s.str.startswith/endswith | s.Str().HasPrefix / HasSuffix | done | |
| s.str.split/get/slice | s.Str().Split / Get(i) / Slice(a,b) | done | |
| s.dt.year/month/day/hour/minute/second | same names | done | |
| s.dt.weekday/dayofyear/quarter | s.Dt().Weekday()/DayOfYear()/Quarter() | done | Monday=0 |
| s.dt.is_month_start/end, is_year_start/end | same names | done | |
| s.dt.date/time | s.Dt().Date() / Time() | partial | Time returns strings |
| s.dt.tz_localize / tz_convert | — | not_supported | see known differences |

## GroupBy

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| df.groupby(keys) | df.GroupBy(keys...) / GroupByOpts | done | typed engine (v0.5); GroupSort / GroupDropNA; multi-key; dropna=False NA group sorts last |
| gb.size()/count() | gb.Size() / gb.Count(cols...) | done | |
| gb.sum/mean/median/min/max/std/var | same names | done | segment reducers; NA-skipping; ddof=1 |
| gb.first()/last()/nunique() | gb.First/Last/NUnique | done | typed index-selector gather |
| gb.agg({...}) | gb.Agg(map) / gb.AggList(map) | partial | column_agg naming, sorted column order |
| gb.apply(fn) | gb.Apply(fn) | partial | |
| gb.transform / gb.filter | — | planned | |

## Merge / join / concat

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| pd.merge how=inner/left/right/outer/cross | pd.Merge / df.Merge | done | typed hash join (v0.6); duplicate keys expand; NA keys never match (documented) |
| left_on/right_on | MergeOptions.LeftOn/RightOn | done | right key column dropped |
| multi-key merge | MergeOptions.On: []string{...} | done | typed composite ids |
| suffixes / validate / indicator | MergeOptions fields | done | |
| df.join(other) | df.Join(other, JoinOptions) | partial | typed index join (Range/Int64/String/Datetime); left/inner/outer |
| pd.concat axis=0/1 | pd.Concat(frames, ConcatAxis/pd.Join/pd.IgnoreIndex) | partial | typed engine (v0.6.1): typed append, numeric promotion, NA gaps; axis=1 positional only (no index alignment) |
| pd.concat of Series | pd.ConcatSeries(...) | done | typed append + promotion (v0.6.1) |

## Reshape and window

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| df.melt(...) | df.Melt(MeltOptions) | done | preserves row order |
| df.pivot(...) | df.Pivot(PivotOptions) | done | sorted labels; duplicates error |
| df.pivot_table(...) | df.PivotTable(PivotTableOptions) | partial | single index/columns/values |
| df.stack()/unstack() | Stack()/Unstack() | not_supported | ErrNotImplemented |
| s.rolling(w, min_periods, center) | s.Rolling(w, MinPeriods/RollingCenter) | done | count/sum/mean/median/min/max/std/var |
| df.rolling(...) | df.Rolling(w, ...) | done | numeric columns |
| s.expanding()/df.expanding() | s.Expanding() / df.Expanding() | done | count/sum/mean/median/min/max/std/var |
| df.resample(...) | df.Resample(...) | not_supported | ErrNotImplemented |
| s.ewm(...) | — | planned | |

## IO

| pandas API | go-pandas API | Status | Notes |
|---|---|---|---|
| pd.read_csv basics | pd.ReadCSV / ReadCSVReader | done | header, sep, dtype inference |
| na_values / keep_default_na | WithNAValues / WithKeepDefaultNA | done | |
| parse_dates / date_format | WithParseDates / WithDateFormat | done | |
| usecols / nrows | WithUseCols / WithNRows | done | |
| dtype={"col": "category"} | pd.WithCategorical("col", ...) | done | writers emit labels, never codes (v0.7) |
| df.to_csv | df.ToCSV / df.WriteCSV | done | |
| pd.read_json orient=records/split/columns/values | pd.ReadJSON + pd.JSONOrient | done | |
| df.to_json | df.ToJSON | done | all four orientations |
| lines=True | pd.ReadNDJSON / df.ToNDJSON | done | |
| read_parquet / read_excel / read_sql | — | planned | adapters |

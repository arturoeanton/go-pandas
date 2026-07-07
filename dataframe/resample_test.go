package dataframe_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

func day(d int, hour int) time.Time {
	return time.Date(2026, 1, d, hour, 0, 0, 0, time.UTC)
}

// tsFrame: unsorted rows, a duplicate timestamp, an NA timestamp and an
// NA value.
func tsFrame(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	dates := series.SeriesOf("date", []any{
		day(2, 10), day(1, 9), day(1, 15), day(2, 10), nil, day(3, 8),
	})
	converted, err := series.ToDatetime(dates)
	if err != nil {
		t.Fatal(err)
	}
	df, err := dataframe.NewDataFrame(
		converted,
		series.NewSeries("v", []any{2.0, 1.0, nil, 4.0, 99.0, 10.0}),
		series.StringSeries("tag", []string{"b", "a", "c", "d", "e", "f"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := df.SetIndex("date")
	if err != nil {
		t.Fatal(err)
	}
	return indexed
}

func TestSetIndexDatetimeCreatesDatetimeIndex(t *testing.T) {
	df := tsFrame(t)
	di, ok := df.Index().(*index.DatetimeIndex)
	if !ok {
		t.Fatalf("index = %T", df.Index())
	}
	if di.Name() != "date" {
		t.Fatalf("name = %q", di.Name())
	}
	if di.At(4) != nil {
		t.Fatal("NA timestamp must be an NA label")
	}
	if di.IsMonotonicIncreasing() {
		t.Fatal("unsorted index with NA must not be monotonic")
	}
	if got := di.Start(); !got.Equal(day(1, 9)) {
		t.Fatalf("Start = %v", got)
	}
	if got := di.End(); !got.Equal(day(3, 8)) {
		t.Fatalf("End = %v", got)
	}
}

func TestDatetimeIndexEnginesPreserve(t *testing.T) {
	df := tsFrame(t)

	taken, err := df.Take([]int{0, 2, 4})
	if err != nil {
		t.Fatal(err)
	}
	ti, ok := taken.Index().(*index.DatetimeIndex)
	if !ok {
		t.Fatalf("Take index = %T", taken.Index())
	}
	if ti.At(2) != nil {
		t.Fatal("Take must keep NA label")
	}
	if _, ok := df.Head(3).Index().(*index.DatetimeIndex); !ok {
		t.Fatal("Head must preserve DatetimeIndex")
	}

	where, err := df.Where(expr.Col("v").Gt(3.0))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := where.Index().(*index.DatetimeIndex); !ok {
		t.Fatalf("Where index = %T", where.Index())
	}

	sorted, err := df.SortValues("v", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sorted.Index().(*index.DatetimeIndex); !ok {
		t.Fatalf("Sort index = %T", sorted.Index())
	}
}

func TestDatetimeIndexLocExact(t *testing.T) {
	df := tsFrame(t)
	rows, err := df.Loc().Rows(day(2, 10)).Get()
	if err != nil {
		t.Fatal(err)
	}
	if rows.Len() != 2 {
		t.Fatalf("duplicate timestamp rows = %d", rows.Len())
	}
	// String labels parse through the inference list; matching is EXACT
	// (no pandas partial-day indexing, documented).
	rows2, err := df.Loc().Rows("2026-01-03 08:00:00").Get()
	if err != nil {
		t.Fatal(err)
	}
	if rows2.Len() != 1 {
		t.Fatalf("string label rows = %d", rows2.Len())
	}
	if _, err := df.Loc().Rows("2026-01-03").Get(); err == nil {
		t.Fatal("midnight label with no midnight row must error (exact match only)")
	}
}

func TestResampleRequiresDatetimeIndex(t *testing.T) {
	df, _ := dataframe.DataFrameFromMap(map[string][]any{"v": {1.0}})
	if _, err := df.Resample("D").Sum(); !errors.Is(err, errs.ErrInvalidIndex) {
		t.Fatalf("flat index error = %v", err)
	}
	mi, _ := dataframe.DataFrameFromRecords([]map[string]any{
		{"a": "x", "b": "y", "v": 1.0},
	}, dataframe.WithColumnOrder("a", "b", "v"))
	indexed, _ := mi.SetIndex("a", "b")
	if _, err := indexed.Resample("D").Sum(); !errors.Is(err, errs.ErrNotImplementedBase) {
		t.Fatalf("MultiIndex error = %v", err)
	}
	if _, err := tsFrame(t).Resample("5min").Sum(); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("unknown frequency error = %v", err)
	}
}

func TestResampleDailyAggregations(t *testing.T) {
	df := tsFrame(t)
	before := fmt.Sprint(df.ToRows())

	sum, err := df.Resample("D").Sum()
	if err != nil {
		t.Fatal(err)
	}
	// NA timestamp row (v=99) skipped; buckets ascending; duplicate
	// timestamps aggregate together; NA value skipped.
	if sum.Len() != 3 {
		t.Fatalf("buckets = %d", sum.Len())
	}
	di := sum.Index().(*index.DatetimeIndex)
	if !di.IsMonotonicIncreasing() {
		t.Fatal("bucket labels must be ascending")
	}
	if got := di.At(0).(time.Time); !got.Equal(day(1, 0)) {
		t.Fatalf("first bucket = %v", got)
	}
	if v := sum.MustCol("v").Values(); v[0] != 1.0 || v[1] != 6.0 || v[2] != 10.0 {
		t.Fatalf("sum = %v", v)
	}
	if got := sum.Columns(); len(got) != 1 || got[0] != "v" {
		t.Fatalf("sum must skip non-numeric columns: %v", got)
	}

	mean, err := df.Resample("D").Mean()
	if err != nil {
		t.Fatal(err)
	}
	if v := mean.MustCol("v").Values(); v[0] != 1.0 || v[1] != 3.0 {
		t.Fatalf("mean = %v", v)
	}

	count, err := df.Resample("D").Count()
	if err != nil {
		t.Fatal(err)
	}
	if got := count.Columns(); len(got) != 2 {
		t.Fatalf("count covers all columns: %v", got)
	}
	if v := count.MustCol("v").Values(); v[0] != 1 || v[1] != 2 {
		t.Fatalf("count v = %v", v)
	}
	if v := count.MustCol("tag").Values(); v[0] != 2 {
		t.Fatalf("count tag = %v", v)
	}

	mn, err := df.Resample("D").Min()
	if err != nil {
		t.Fatal(err)
	}
	if v := mn.MustCol("tag").Values(); v[0] != "a" {
		t.Fatalf("min tag = %v", v)
	}
	first, err := df.Resample("D").First()
	if err != nil {
		t.Fatal(err)
	}
	if dt := first.DTypes()["tag"]; dt != dtype.String {
		t.Fatalf("first must preserve dtype, got %v", dt)
	}
	if v := first.MustCol("tag").Values(); v[1] != "b" {
		t.Fatalf("first tag (row order within bucket) = %v", v)
	}
	last, err := df.Resample("D").Last()
	if err != nil {
		t.Fatal(err)
	}
	if v := last.MustCol("tag").Values(); v[1] != "d" {
		t.Fatalf("last tag = %v", v)
	}

	if fmt.Sprint(df.ToRows()) != before {
		t.Fatal("Resample mutated the input")
	}
}

func TestResampleFrequencies(t *testing.T) {
	mk := func(times ...time.Time) *dataframe.DataFrame {
		t.Helper()
		vals := make([]float64, len(times))
		for i := range vals {
			vals[i] = float64(i + 1)
		}
		df, err := dataframe.NewDataFrame(
			series.TimeSeries("date", times),
			series.FloatSeries("v", vals))
		if err != nil {
			t.Fatal(err)
		}
		indexed, err := df.SetIndex("date")
		if err != nil {
			t.Fatal(err)
		}
		return indexed
	}

	hourly, err := mk(
		time.Date(2026, 1, 1, 9, 10, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 9, 50, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 10, 20, 0, 0, time.UTC),
	).Resample("H").Sum()
	if err != nil {
		t.Fatal(err)
	}
	if hourly.Len() != 2 || hourly.MustCol("v").Values()[0] != 3.0 {
		t.Fatalf("hourly = %v", hourly.MustCol("v").Values())
	}

	weekly, err := mk(
		time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC),  // Wednesday
		time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),  // Friday, same week
		time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC), // next Monday
	).Resample("W").Sum()
	if err != nil {
		t.Fatal(err)
	}
	if weekly.Len() != 2 {
		t.Fatalf("weekly buckets = %d", weekly.Len())
	}
	if got := weekly.Index().At(0).(time.Time); got.Weekday() != time.Monday {
		t.Fatalf("week anchor = %v", got.Weekday())
	}

	months := mk(
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
	)
	ms, err := months.Resample("MS").Sum()
	if err != nil {
		t.Fatal(err)
	}
	if got := ms.Index().At(0).(time.Time); got.Day() != 1 {
		t.Fatalf("MS label = %v", got)
	}
	// "M" aliases month-start (documented difference from pandas).
	m, err := months.Resample("M").Sum()
	if err != nil {
		t.Fatal(err)
	}
	if !m.Index().At(0).(time.Time).Equal(ms.Index().At(0).(time.Time)) {
		t.Fatal("M must alias MS")
	}
	me, err := months.Resample("ME").Sum()
	if err != nil {
		t.Fatal(err)
	}
	if got := me.Index().At(0).(time.Time); got.Day() != 31 {
		t.Fatalf("ME label = %v", got)
	}
}

func TestResampleAllNABucket(t *testing.T) {
	df, err := dataframe.NewDataFrame(
		series.TimeSeries("date", []time.Time{day(1, 0), day(1, 5), day(2, 0)}),
		series.NewSeries("v", []any{nil, nil, 3.0}))
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := df.SetIndex("date")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := indexed.Resample("D").Sum()
	if err != nil {
		t.Fatal(err)
	}
	// pandas sum of an all-NA bucket is 0 (golden-consistent).
	if v := sum.MustCol("v").Values(); v[0] != 0.0 || v[1] != 3.0 {
		t.Fatalf("all-NA sum = %v", v)
	}
	mean, err := indexed.Resample("D").Mean()
	if err != nil {
		t.Fatal(err)
	}
	if v := mean.MustCol("v").Values(); v[0] != nil {
		t.Fatalf("all-NA mean must be NA, got %v", v)
	}
}

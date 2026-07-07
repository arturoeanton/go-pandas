package compat_test

import (
	"testing"
	"time"

	pd "github.com/arturoeanton/go-pandas"
)

// Fixtures mirroring generate_pandas_goldens.py timeseries_suite().

func tsIndexed(t *testing.T) *pd.DataFrame {
	t.Helper()
	dates, err := pd.ToDatetime(pd.StringSeries("date", []string{
		"2026-01-02 10:00:00",
		"2026-01-01 09:30:00",
		"2026-01-01 15:00:00",
		"2026-01-02 10:00:00",
		"2026-01-03 08:00:00",
	}))
	if err != nil {
		t.Fatal(err)
	}
	df, err := pd.NewDataFrame(dates,
		pd.NewSeries("v", []any{2.0, 1.0, nil, 4.0, 10.0}),
		pd.StringSeries("tag", []string{"b", "a", "c", "d", "f"}))
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := df.SetIndex("date")
	if err != nil {
		t.Fatal(err)
	}
	return indexed
}

func tsSmall(t *testing.T, dateStrs []string, vals []float64) *pd.DataFrame {
	t.Helper()
	dates, err := pd.ToDatetime(pd.StringSeries("date", dateStrs))
	if err != nil {
		t.Fatal(err)
	}
	df, err := pd.NewDataFrame(dates, pd.FloatSeries("v", vals))
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := df.SetIndex("date")
	if err != nil {
		t.Fatal(err)
	}
	return indexed
}

// dateStrings formats a datetime series like pandas .dt.strftime("%Y-%m-%d").
func dateStrings(s *pd.Series) *pd.Series {
	values := make([]any, s.Len())
	for i, v := range s.Values() {
		if v == nil {
			continue
		}
		values[i] = v.(time.Time).Format("2006-01-02")
	}
	return pd.NewSeries(s.Name(), values)
}

var timeseriesCases = map[string]caseFn{
	"ts_to_datetime_format": func(t *testing.T) (any, error) {
		s, err := pd.ToDatetime(pd.StringSeries("d", []string{"01/02/2026", "28/12/2026"}),
			pd.WithDatetimeFormat("%d/%m/%Y"))
		if err != nil {
			return nil, err
		}
		return dateStrings(s), nil
	},
	"ts_to_datetime_coerce": func(t *testing.T) (any, error) {
		s, err := pd.ToDatetime(pd.StringSeries("d", []string{"2026-01-01", "bad", ""}),
			pd.WithDatetimeErrors("coerce"))
		if err != nil {
			return nil, err
		}
		return dateStrings(s), nil
	},
	"ts_resample_d_sum": func(t *testing.T) (any, error) {
		out, err := tsIndexed(t).Resample("D").Sum()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"ts_resample_d_mean": func(t *testing.T) (any, error) {
		out, err := tsIndexed(t).Resample("D").Mean()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"ts_resample_d_count": func(t *testing.T) (any, error) {
		out, err := tsIndexed(t).Resample("D").Count()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"ts_resample_d_min": func(t *testing.T) (any, error) {
		out, err := tsIndexed(t).Select("v")
		if err != nil {
			return nil, err
		}
		min, err := out.Resample("D").Min()
		if err != nil {
			return nil, err
		}
		return min.ResetIndex(), nil
	},
	"ts_resample_h_sum": func(t *testing.T) (any, error) {
		hours := tsSmall(t, []string{
			"2026-01-01 09:10:00", "2026-01-01 09:50:00",
			"2026-01-01 10:20:00", "2026-01-01 11:59:00",
		}, []float64{1, 2, 3, 4})
		out, err := hours.Resample("H").Sum()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"ts_resample_ms_sum": func(t *testing.T) (any, error) {
		months := tsSmall(t, []string{"2026-01-15", "2026-01-20", "2026-02-03"}, []float64{1, 2, 3})
		out, err := months.Resample("MS").Sum()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
	"ts_resample_me_sum": func(t *testing.T) (any, error) {
		months := tsSmall(t, []string{"2026-01-15", "2026-01-20", "2026-02-03"}, []float64{1, 2, 3})
		out, err := months.Resample("ME").Sum()
		if err != nil {
			return nil, err
		}
		return out.ResetIndex(), nil
	},
}

func init() {
	for name, fn := range timeseriesCases {
		pandasCases[name] = fn
	}
}

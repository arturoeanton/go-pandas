package series

import "time"

// DatetimeAccessor exposes vectorized datetime components, like Series.dt.
type DatetimeAccessor struct {
	s *Series
}

// Dt returns the datetime accessor.
func (s *Series) Dt() *DatetimeAccessor { return &DatetimeAccessor{s: s} }

func (da *DatetimeAccessor) mapTime(f func(t time.Time) any) *Series {
	src := da.s
	data := make([]any, src.Len())
	mask := make([]bool, src.Len())
	for i := range src.data {
		if src.mask[i] {
			mask[i] = true
			continue
		}
		t, ok := src.data[i].(time.Time)
		if !ok {
			mask[i] = true
			continue
		}
		data[i] = f(t)
	}
	return NewSeries(src.name, dataWithMask(data, mask), WithIndex(src.index))
}

// Year extracts the year.
func (da *DatetimeAccessor) Year() *Series {
	return da.mapTime(func(t time.Time) any { return t.Year() })
}

// Month extracts the month (1-12).
func (da *DatetimeAccessor) Month() *Series {
	return da.mapTime(func(t time.Time) any { return int(t.Month()) })
}

// Day extracts the day of month.
func (da *DatetimeAccessor) Day() *Series {
	return da.mapTime(func(t time.Time) any { return t.Day() })
}

// Hour extracts the hour.
func (da *DatetimeAccessor) Hour() *Series {
	return da.mapTime(func(t time.Time) any { return t.Hour() })
}

// Minute extracts the minute.
func (da *DatetimeAccessor) Minute() *Series {
	return da.mapTime(func(t time.Time) any { return t.Minute() })
}

// Second extracts the second.
func (da *DatetimeAccessor) Second() *Series {
	return da.mapTime(func(t time.Time) any { return t.Second() })
}

// Weekday extracts the day of week with Monday=0, like pandas.
func (da *DatetimeAccessor) Weekday() *Series {
	return da.mapTime(func(t time.Time) any {
		// Go: Sunday=0 ... Saturday=6; pandas: Monday=0 ... Sunday=6.
		return (int(t.Weekday()) + 6) % 7
	})
}

// Date truncates to midnight.
func (da *DatetimeAccessor) Date() *Series {
	return da.mapTime(func(t time.Time) any {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	})
}

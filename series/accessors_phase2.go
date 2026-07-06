package series

import (
	"regexp"
	"time"
)

// Match reports whether each string matches the regular expression
// anchored at the start, like s.str.match(pattern).
func (sa *StringAccessor) Match(pattern string) (*Series, error) {
	re, err := regexp.Compile("^(?:" + pattern + ")")
	if err != nil {
		return nil, err
	}
	return sa.mapString(func(s string) any { return re.MatchString(s) }), nil
}

// ContainsRegex reports whether each string contains a regex match, like
// s.str.contains(pattern, regex=True).
func (sa *StringAccessor) ContainsRegex(pattern string) (*Series, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return sa.mapString(func(s string) any { return re.MatchString(s) }), nil
}

// ReplaceRegex substitutes every regex match, like
// s.str.replace(pattern, repl, regex=True).
func (sa *StringAccessor) ReplaceRegex(pattern, repl string) (*Series, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return sa.mapString(func(s string) any { return re.ReplaceAllString(s, repl) }), nil
}

// Get returns the i-th rune of each string (negative counts from the
// end); out-of-range positions become NA, like s.str.get(i).
func (sa *StringAccessor) Get(i int) *Series {
	return sa.mapString(func(s string) any {
		runes := []rune(s)
		pos := i
		if pos < 0 {
			pos += len(runes)
		}
		if pos < 0 || pos >= len(runes) {
			return nil
		}
		return string(runes[pos])
	})
}

// Slice returns the [start, stop) substring of each string with Python
// negative-index semantics, like s.str.slice(start, stop).
func (sa *StringAccessor) Slice(start, stop int) *Series {
	return sa.mapString(func(s string) any {
		runes := []rune(s)
		n := len(runes)
		lo, hi := start, stop
		if lo < 0 {
			lo += n
		}
		if hi < 0 {
			hi += n
		}
		if lo < 0 {
			lo = 0
		}
		if hi > n {
			hi = n
		}
		if lo >= hi {
			return ""
		}
		return string(runes[lo:hi])
	})
}

// Time extracts the time-of-day as "HH:MM:SS" strings.
func (da *DatetimeAccessor) Time() *Series {
	return da.mapTime(func(t time.Time) any { return t.Format("15:04:05") })
}

// DayOfYear extracts the ordinal day (1-366).
func (da *DatetimeAccessor) DayOfYear() *Series {
	return da.mapTime(func(t time.Time) any { return t.YearDay() })
}

// Quarter extracts the calendar quarter (1-4).
func (da *DatetimeAccessor) Quarter() *Series {
	return da.mapTime(func(t time.Time) any { return (int(t.Month())-1)/3 + 1 })
}

// IsMonthStart reports whether each date is the first day of its month.
func (da *DatetimeAccessor) IsMonthStart() *Series {
	return da.mapTime(func(t time.Time) any { return t.Day() == 1 })
}

// IsMonthEnd reports whether each date is the last day of its month.
func (da *DatetimeAccessor) IsMonthEnd() *Series {
	return da.mapTime(func(t time.Time) any {
		return t.AddDate(0, 0, 1).Day() == 1
	})
}

// IsYearStart reports whether each date is January 1st.
func (da *DatetimeAccessor) IsYearStart() *Series {
	return da.mapTime(func(t time.Time) any { return t.YearDay() == 1 })
}

// IsYearEnd reports whether each date is December 31st.
func (da *DatetimeAccessor) IsYearEnd() *Series {
	return da.mapTime(func(t time.Time) any {
		return t.Month() == time.December && t.Day() == 31
	})
}

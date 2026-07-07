package pandas

import (
	"github.com/arturoeanton/go-pandas/dataframe"
	"github.com/arturoeanton/go-pandas/series"
)

// v0.9 time-series re-exports.
type (
	DatetimeOption  = series.DatetimeOption
	DatetimeOptions = series.DatetimeOptions
	Resampler       = dataframe.Resampler
)

// WithDatetimeFormat sets an explicit pandas-style datetime format
// ("%Y-%m-%d"; see docs/timeseries.md for the directive table).
func WithDatetimeFormat(format string) DatetimeOption {
	return series.WithDatetimeFormat(format)
}

// WithDatetimeErrors sets the ToDatetime error mode: "raise" (default)
// or "coerce" (invalid values become NA).
func WithDatetimeErrors(mode string) DatetimeOption {
	return series.WithDatetimeErrors(mode)
}

// WithDatetimeUnit interprets numeric values as unix timestamps in the
// given unit ("s", "ms", "us", "ns").
func WithDatetimeUnit(unit string) DatetimeOption {
	return series.WithDatetimeUnit(unit)
}

// WithDatetimeUTC converts parsed values to UTC (no timezone dtype).
func WithDatetimeUTC(v bool) DatetimeOption {
	return series.WithDatetimeUTC(v)
}

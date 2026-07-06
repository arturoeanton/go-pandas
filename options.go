package pandas

import "github.com/arturoeanton/go-pandas/internal/display"

// DisplayOptions controls how Series and DataFrames render as text.
type DisplayOptions struct {
	MaxRows   int
	MaxCols   int
	Width     int
	Precision int
}

// SetDisplayOptions updates the global display options (zero fields keep
// their current value).
func SetDisplayOptions(opts DisplayOptions) {
	display.Set(display.Options{
		MaxRows:   opts.MaxRows,
		MaxCols:   opts.MaxCols,
		Width:     opts.Width,
		Precision: opts.Precision,
	})
}

// GetDisplayOptions returns the current display options.
func GetDisplayOptions() DisplayOptions {
	o := display.Get()
	return DisplayOptions{
		MaxRows:   o.MaxRows,
		MaxCols:   o.MaxCols,
		Width:     o.Width,
		Precision: o.Precision,
	}
}

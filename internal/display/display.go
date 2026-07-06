// Package display holds the global display options used when rendering
// Series and DataFrames as text. It is the only global mutable state in
// go-pandas.
package display

import "sync"

// Options mirrors pandas display options.
type Options struct {
	MaxRows   int
	MaxCols   int
	Width     int
	Precision int
}

var (
	mu   sync.RWMutex
	opts = Options{MaxRows: 20, MaxCols: 20, Width: 120, Precision: 6}
)

// Set replaces the global display options. Zero fields keep their default.
func Set(o Options) {
	mu.Lock()
	defer mu.Unlock()
	if o.MaxRows > 0 {
		opts.MaxRows = o.MaxRows
	}
	if o.MaxCols > 0 {
		opts.MaxCols = o.MaxCols
	}
	if o.Width > 0 {
		opts.Width = o.Width
	}
	if o.Precision > 0 {
		opts.Precision = o.Precision
	}
}

// Get returns the current display options.
func Get() Options {
	mu.RLock()
	defer mu.RUnlock()
	return opts
}

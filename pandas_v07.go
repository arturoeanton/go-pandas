package pandas

import (
	pdio "github.com/arturoeanton/go-pandas/io"
	"github.com/arturoeanton/go-pandas/series"
)

// v0.7 categorical re-exports.
type (
	CategoricalAccessor = series.CategoricalAccessor
	CategoricalOption   = series.CategoricalOption
	CategoricalOptions  = series.CategoricalOptions
)

// NewCategoricalSeries builds a categorical series from untyped values.
// Without WithCategories the category list is the sorted distinct labels
// (pandas' astype("category") default).
func NewCategoricalSeries(name string, values []any, opts ...CategoricalOption) (*Series, error) {
	return series.NewCategoricalSeries(name, values, opts...)
}

// CategoricalSeries builds a categorical series from strings.
func CategoricalSeries(name string, values []string, opts ...CategoricalOption) (*Series, error) {
	return series.CategoricalSeries(name, values, opts...)
}

// WithCategories fixes the explicit category list and order; values
// outside the list are an error.
func WithCategories(values ...any) CategoricalOption { return series.WithCategories(values...) }

// WithOrdered marks the categorical as ordered, enabling Gt/Ge/Lt/Le.
func WithOrdered(v bool) CategoricalOption { return series.WithOrdered(v) }

// WithCategorical marks CSV columns to load with the categorical dtype,
// like read_csv(dtype={"col": "category"}). Categories are never
// inferred without this explicit opt-in.
func WithCategorical(columns ...string) CSVOption { return pdio.WithCategorical(columns...) }

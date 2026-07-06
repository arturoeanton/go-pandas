// Package io implements the low-level CSV/JSON/NDJSON readers and writers
// used by DataFrame IO. It parses into a neutral Table (column names +
// untyped rows) so it stays free of DataFrame dependencies.
package io

// Table is the neutral result of parsing a tabular source.
type Table struct {
	Columns []string
	Rows    [][]any
}

// CSVOptions configures CSV reading and writing.
type CSVOptions struct {
	Header     bool
	Comma      rune
	InferTypes bool
	NAValues   []string
	// KeepDefaultNA appends the default NA strings to custom NAValues
	// instead of replacing them, like read_csv(keep_default_na=True).
	KeepDefaultNA bool
	ParseDates    []string
	DateFormat    string
	TrimSpace     bool
	Comment       rune
	Limit         int
	// UseCols restricts parsing to the named columns, like
	// read_csv(usecols=[...]).
	UseCols []string
}

// DefaultCSVOptions returns the pandas-like defaults.
func DefaultCSVOptions() CSVOptions {
	return CSVOptions{
		Header:     true,
		Comma:      ',',
		InferTypes: true,
		NAValues:   []string{"", "NA", "NaN", "null", "NULL", "None"},
	}
}

// CSVOption mutates CSVOptions.
type CSVOption func(*CSVOptions)

// WithHeader toggles the header row.
func WithHeader(v bool) CSVOption { return func(o *CSVOptions) { o.Header = v } }

// WithComma sets the field delimiter.
func WithComma(r rune) CSVOption { return func(o *CSVOptions) { o.Comma = r } }

// WithInferTypes toggles numeric/bool type inference.
func WithInferTypes(v bool) CSVOption { return func(o *CSVOptions) { o.InferTypes = v } }

// WithNAValues replaces the set of strings parsed as missing.
func WithNAValues(values ...string) CSVOption {
	return func(o *CSVOptions) { o.NAValues = values }
}

// WithParseDates marks columns to parse as datetimes.
func WithParseDates(columns ...string) CSVOption {
	return func(o *CSVOptions) { o.ParseDates = columns }
}

// WithDateFormat sets an explicit Go time layout for ParseDates columns.
func WithDateFormat(format string) CSVOption {
	return func(o *CSVOptions) { o.DateFormat = format }
}

// WithTrimSpace trims whitespace around fields.
func WithTrimSpace(v bool) CSVOption { return func(o *CSVOptions) { o.TrimSpace = v } }

// WithComment sets a comment character; lines starting with it are skipped.
func WithComment(r rune) CSVOption { return func(o *CSVOptions) { o.Comment = r } }

// WithLimit caps the number of data rows read (0 means no limit).
func WithLimit(n int) CSVOption { return func(o *CSVOptions) { o.Limit = n } }

// WithNRows is an alias of WithLimit matching read_csv(nrows=n).
func WithNRows(n int) CSVOption { return WithLimit(n) }

// WithUseCols restricts parsing to the named columns.
func WithUseCols(columns ...string) CSVOption {
	return func(o *CSVOptions) { o.UseCols = columns }
}

// WithKeepDefaultNA appends the default NA strings to a custom
// WithNAValues list instead of replacing them.
func WithKeepDefaultNA(v bool) CSVOption {
	return func(o *CSVOptions) { o.KeepDefaultNA = v }
}

// JSONOptions configures JSON reading and writing.
type JSONOptions struct {
	// Orient is one of "records" (default) or "values".
	Orient string
	// Indent pretty-prints output when non-empty.
	Indent string
}

// DefaultJSONOptions returns the defaults (records orientation).
func DefaultJSONOptions() JSONOptions { return JSONOptions{Orient: "records"} }

// JSONOption mutates JSONOptions.
type JSONOption func(*JSONOptions)

// WithOrient sets the JSON orientation ("records" or "values").
func WithOrient(orient string) JSONOption {
	return func(o *JSONOptions) { o.Orient = orient }
}

// WithIndent pretty-prints written JSON with the given indent string.
func WithIndent(indent string) JSONOption {
	return func(o *JSONOptions) { o.Indent = indent }
}

package dataframe

import (
	stdio "io"
	"os"

	"github.com/arturoeanton/go-pandas/dtype"
	pdio "github.com/arturoeanton/go-pandas/io"
)

// CSVOptions and CSVOption re-export the io package types.
type (
	CSVOptions = pdio.CSVOptions
	CSVOption  = pdio.CSVOption
)

// ReadCSV reads a CSV file into a DataFrame (pd.read_csv).
func ReadCSV(path string, opts ...CSVOption) (*DataFrame, error) {
	table, err := pdio.ReadCSVTable(path, opts...)
	if err != nil {
		return nil, err
	}
	df, err := DataFrameFromRows(table.Columns, table.Rows)
	if err != nil {
		return nil, err
	}
	return applyCategorical(df, opts)
}

// ReadCSVReader reads CSV from any reader.
func ReadCSVReader(r stdio.Reader, opts ...CSVOption) (*DataFrame, error) {
	table, err := pdio.ReadCSVTableReader(r, opts...)
	if err != nil {
		return nil, err
	}
	df, err := DataFrameFromRows(table.Columns, table.Rows)
	if err != nil {
		return nil, err
	}
	return applyCategorical(df, opts)
}

// applyCategorical converts the WithCategorical columns to the
// categorical dtype after parsing (v0.7). Categories are the sorted
// distinct values, exactly like Astype(Category).
func applyCategorical(df *DataFrame, opts []CSVOption) (*DataFrame, error) {
	o := pdio.DefaultCSVOptions()
	for _, f := range opts {
		f(&o)
	}
	for _, name := range o.Categorical {
		s, err := df.Col(name)
		if err != nil {
			return nil, err
		}
		cat, err := s.Astype(dtype.Category)
		if err != nil {
			return nil, err
		}
		if df, err = df.Assign(name, cat); err != nil {
			return nil, err
		}
	}
	return df, nil
}

// ToCSV writes the frame to a CSV file (df.to_csv).
func (df *DataFrame) ToCSV(path string, opts ...CSVOption) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return df.WriteCSV(f, opts...)
}

// WriteCSV writes the frame as CSV to any writer.
func (df *DataFrame) WriteCSV(w stdio.Writer, opts ...CSVOption) error {
	table := &pdio.Table{Columns: df.Columns(), Rows: df.ToRows()}
	return pdio.WriteCSVTable(w, table, opts...)
}

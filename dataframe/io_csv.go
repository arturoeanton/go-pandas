package dataframe

import (
	stdio "io"
	"os"

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
	return DataFrameFromRows(table.Columns, table.Rows)
}

// ReadCSVReader reads CSV from any reader.
func ReadCSVReader(r stdio.Reader, opts ...CSVOption) (*DataFrame, error) {
	table, err := pdio.ReadCSVTableReader(r, opts...)
	if err != nil {
		return nil, err
	}
	return DataFrameFromRows(table.Columns, table.Rows)
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

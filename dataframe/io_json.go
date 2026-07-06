package dataframe

import (
	stdio "io"
	"os"

	pdio "github.com/arturoeanton/go-pandas/io"
)

// JSONOptions and JSONOption re-export the io package types.
type (
	JSONOptions = pdio.JSONOptions
	JSONOption  = pdio.JSONOption
)

// ReadJSON reads a JSON array of objects (pd.read_json, records orient).
func ReadJSON(path string, opts ...JSONOption) (*DataFrame, error) {
	table, err := pdio.ReadJSONTable(path, opts...)
	if err != nil {
		return nil, err
	}
	return DataFrameFromRows(table.Columns, table.Rows)
}

// ReadJSONReader reads JSON from any reader.
func ReadJSONReader(r stdio.Reader, opts ...JSONOption) (*DataFrame, error) {
	table, err := pdio.ReadJSONTableReader(r, opts...)
	if err != nil {
		return nil, err
	}
	return DataFrameFromRows(table.Columns, table.Rows)
}

// ReadNDJSON reads newline-delimited JSON (pd.read_json(lines=True)).
func ReadNDJSON(path string, opts ...JSONOption) (*DataFrame, error) {
	table, err := pdio.ReadNDJSONTable(path, opts...)
	if err != nil {
		return nil, err
	}
	return DataFrameFromRows(table.Columns, table.Rows)
}

// ReadNDJSONReader reads NDJSON from any reader.
func ReadNDJSONReader(r stdio.Reader, opts ...JSONOption) (*DataFrame, error) {
	table, err := pdio.ReadNDJSONTableReader(r, opts...)
	if err != nil {
		return nil, err
	}
	return DataFrameFromRows(table.Columns, table.Rows)
}

// ToJSON writes the frame as JSON to a file (df.to_json).
func (df *DataFrame) ToJSON(path string, opts ...JSONOption) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return df.WriteJSON(f, opts...)
}

// WriteJSON writes the frame as JSON to any writer.
func (df *DataFrame) WriteJSON(w stdio.Writer, opts ...JSONOption) error {
	table := &pdio.Table{Columns: df.Columns(), Rows: df.ToRows()}
	return pdio.WriteJSONTable(w, table, opts...)
}

// ToNDJSON writes the frame as newline-delimited JSON to a file.
func (df *DataFrame) ToNDJSON(path string, opts ...JSONOption) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return df.WriteNDJSON(f, opts...)
}

// WriteNDJSON writes the frame as NDJSON to any writer.
func (df *DataFrame) WriteNDJSON(w stdio.Writer, opts ...JSONOption) error {
	table := &pdio.Table{Columns: df.Columns(), Rows: df.ToRows()}
	return pdio.WriteNDJSONTable(w, table)
}

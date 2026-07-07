package io

import (
	"encoding/json"
	"fmt"
	stdio "io"
	"os"
	"sort"
	"strconv"

	"github.com/arturoeanton/go-pandas/errs"
)

// ReadJSONTable parses a JSON file (array of objects for "records", array
// of arrays for "values") into a neutral Table.
func ReadJSONTable(path string, opts ...JSONOption) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadJSONTableReader(f, opts...)
}

// ReadJSONTableReader parses JSON from a reader.
func ReadJSONTableReader(r stdio.Reader, opts ...JSONOption) (*Table, error) {
	o := DefaultJSONOptions()
	for _, f := range opts {
		f(&o)
	}
	switch o.Orient {
	case "records", "":
		var records []map[string]any
		if err := json.NewDecoder(r).Decode(&records); err != nil {
			return nil, fmt.Errorf("reading JSON records: %w", err)
		}
		return recordsToTable(records), nil
	case "values":
		var rows [][]any
		if err := json.NewDecoder(r).Decode(&rows); err != nil {
			return nil, fmt.Errorf("reading JSON values: %w", err)
		}
		table := &Table{Rows: normalizeRows(rows)}
		if len(rows) > 0 {
			for i := range rows[0] {
				table.Columns = append(table.Columns, fmt.Sprintf("column_%d", i))
			}
		}
		return table, nil
	case "split":
		var doc struct {
			Columns []string `json:"columns"`
			Index   []any    `json:"index"`
			Data    [][]any  `json:"data"`
		}
		if err := json.NewDecoder(r).Decode(&doc); err != nil {
			return nil, fmt.Errorf("reading JSON split: %w", err)
		}
		return &Table{Columns: doc.Columns, Rows: normalizeRows(doc.Data)}, nil
	case "columns":
		var doc map[string]map[string]any
		if err := json.NewDecoder(r).Decode(&doc); err != nil {
			return nil, fmt.Errorf("reading JSON columns: %w", err)
		}
		var columns []string
		for name := range doc {
			columns = append(columns, name)
		}
		sort.Strings(columns)
		// Row keys are shared across columns; sort them for determinism.
		keySet := map[string]bool{}
		for _, col := range doc {
			for k := range col {
				keySet[k] = true
			}
		}
		var keys []string
		for k := range keySet {
			keys = append(keys, k)
		}
		sortRowKeys(keys)
		table := &Table{Columns: columns}
		for _, k := range keys {
			row := make([]any, len(columns))
			for i, name := range columns {
				row[i] = normalizeJSONValue(doc[name][k])
			}
			table.Rows = append(table.Rows, row)
		}
		return table, nil
	}
	return nil, errs.NotImplemented("JSON orient " + o.Orient)
}

// sortRowKeys orders JSON row keys numerically when they all parse as
// integers (pandas writes RangeIndex labels as "0", "1", ..., "10" — a
// plain string sort would put "10" before "2").
func sortRowKeys(keys []string) {
	nums := make(map[string]int, len(keys))
	allInts := true
	for _, k := range keys {
		n, err := strconv.Atoi(k)
		if err != nil {
			allInts = false
			break
		}
		nums[k] = n
	}
	if allInts {
		sort.Slice(keys, func(a, b int) bool { return nums[keys[a]] < nums[keys[b]] })
		return
	}
	sort.Strings(keys)
}

// recordsToTable flattens JSON objects into a Table with sorted column
// names (JSON objects are unordered).
func recordsToTable(records []map[string]any) *Table {
	seen := map[string]bool{}
	var columns []string
	for _, rec := range records {
		for k := range rec {
			if !seen[k] {
				seen[k] = true
				columns = append(columns, k)
			}
		}
	}
	sort.Strings(columns)
	table := &Table{Columns: columns}
	for _, rec := range records {
		row := make([]any, len(columns))
		for i, c := range columns {
			row[i] = normalizeJSONValue(rec[c])
		}
		table.Rows = append(table.Rows, row)
	}
	return table
}

func normalizeRows(rows [][]any) [][]any {
	for _, row := range rows {
		for i, v := range row {
			row[i] = normalizeJSONValue(v)
		}
	}
	return rows
}

// normalizeJSONValue converts json.Number-free decoded values: floats that
// are integral become int so dtype inference matches pandas read_json.
func normalizeJSONValue(v any) any {
	if f, ok := v.(float64); ok {
		if f == float64(int64(f)) && f >= -1e15 && f <= 1e15 {
			return int(f)
		}
	}
	return v
}

// WriteJSONTable writes a Table as JSON in the requested orientation.
func WriteJSONTable(w stdio.Writer, table *Table, opts ...JSONOption) error {
	o := DefaultJSONOptions()
	for _, f := range opts {
		f(&o)
	}
	enc := json.NewEncoder(w)
	if o.Indent != "" {
		enc.SetIndent("", o.Indent)
	}
	switch o.Orient {
	case "records", "":
		records := make([]map[string]any, len(table.Rows))
		for i, row := range table.Rows {
			rec := make(map[string]any, len(table.Columns))
			for j, c := range table.Columns {
				rec[c] = row[j]
			}
			records[i] = rec
		}
		return enc.Encode(records)
	case "values":
		return enc.Encode(table.Rows)
	case "split":
		idx := make([]any, len(table.Rows))
		for i := range idx {
			idx[i] = i
		}
		return enc.Encode(map[string]any{
			"columns": table.Columns,
			"index":   idx,
			"data":    table.Rows,
		})
	case "columns":
		doc := make(map[string]map[string]any, len(table.Columns))
		for j, name := range table.Columns {
			col := make(map[string]any, len(table.Rows))
			for i, row := range table.Rows {
				col[fmt.Sprint(i)] = row[j]
			}
			doc[name] = col
		}
		return enc.Encode(doc)
	}
	return errs.NotImplemented("JSON orient " + o.Orient)
}

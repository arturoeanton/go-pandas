package io

import (
	"encoding/csv"
	"fmt"
	stdio "io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// ReadCSVTable parses a CSV file into a neutral Table.
func ReadCSVTable(path string, opts ...CSVOption) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadCSVTableReader(f, opts...)
}

// ReadCSVTableReader parses CSV from a reader into a neutral Table.
func ReadCSVTableReader(r stdio.Reader, opts ...CSVOption) (*Table, error) {
	o := DefaultCSVOptions()
	for _, f := range opts {
		f(&o)
	}
	cr := csv.NewReader(r)
	cr.Comma = o.Comma
	cr.Comment = o.Comment
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}
	table := &Table{}
	if len(records) == 0 {
		return table, nil
	}
	start := 0
	if o.Header {
		table.Columns = append([]string(nil), records[0]...)
		start = 1
	} else {
		for i := range records[0] {
			table.Columns = append(table.Columns, fmt.Sprintf("column_%d", i))
		}
	}
	dateCols := make(map[string]bool, len(o.ParseDates))
	for _, c := range o.ParseDates {
		dateCols[c] = true
	}
	naSet := make(map[string]bool, len(o.NAValues))
	for _, v := range o.NAValues {
		naSet[v] = true
	}
	if o.KeepDefaultNA {
		for _, v := range DefaultCSVOptions().NAValues {
			naSet[v] = true
		}
	}
	// UseCols restricts which physical columns are kept.
	keep := make([]int, 0, len(table.Columns))
	if len(o.UseCols) > 0 {
		want := make(map[string]bool, len(o.UseCols))
		for _, c := range o.UseCols {
			want[c] = true
		}
		var kept []string
		for i, name := range table.Columns {
			if want[name] {
				keep = append(keep, i)
				kept = append(kept, name)
			}
		}
		if len(kept) != len(o.UseCols) {
			return nil, fmt.Errorf("%w: usecols not found in header", errs.ErrColumnNotFound)
		}
		table.Columns = kept
	} else {
		for i := range table.Columns {
			keep = append(keep, i)
		}
	}
	for _, rec := range records[start:] {
		if o.Limit > 0 && len(table.Rows) >= o.Limit {
			break
		}
		row := make([]any, len(table.Columns))
		for j, src := range keep {
			var cell string
			if src < len(rec) {
				cell = rec[src]
			}
			if o.TrimSpace {
				cell = strings.TrimSpace(cell)
			}
			if naSet[cell] {
				row[j] = nil
				continue
			}
			switch {
			case dateCols[table.Columns[j]]:
				t, err := parseDate(cell, o.DateFormat)
				if err != nil {
					return nil, fmt.Errorf("column %q: %w", table.Columns[j], err)
				}
				row[j] = t
			case o.InferTypes:
				row[j] = inferCell(cell)
			default:
				row[j] = cell
			}
		}
		table.Rows = append(table.Rows, row)
	}
	return table, nil
}

func parseDate(cell, layout string) (time.Time, error) {
	if layout != "" {
		return time.Parse(layout, cell)
	}
	return dtype.ParseTime(cell)
}

// inferCell converts a CSV cell to int, float64, bool or keeps the string.
func inferCell(cell string) any {
	trimmed := strings.TrimSpace(cell)
	if trimmed == "" {
		return cell
	}
	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return int(i)
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}
	switch strings.ToLower(trimmed) {
	case "true":
		return true
	case "false":
		return false
	}
	return cell
}

// WriteCSVTable writes a neutral Table as CSV.
func WriteCSVTable(w stdio.Writer, table *Table, opts ...CSVOption) error {
	o := DefaultCSVOptions()
	for _, f := range opts {
		f(&o)
	}
	cw := csv.NewWriter(w)
	cw.Comma = o.Comma
	if o.Header {
		if err := cw.Write(table.Columns); err != nil {
			return err
		}
	}
	for _, row := range table.Rows {
		rec := make([]string, len(row))
		for i, v := range row {
			rec[i] = formatCell(v)
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func formatCell(v any) string {
	if dtype.IsNA(v) {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		s := strconv.FormatFloat(x, 'g', -1, 64)
		// Keep integral floats as "1500.0" like pandas, so the value
		// round-trips as a float instead of an int.
		if !strings.ContainsAny(s, ".eE") {
			s += ".0"
		}
		return s
	case time.Time:
		return x.Format(time.RFC3339)
	default:
		return fmt.Sprint(x)
	}
}

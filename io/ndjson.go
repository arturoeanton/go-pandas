package io

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdio "io"
	"os"
	"strings"
)

// ReadNDJSONTable parses newline-delimited JSON (one object per line).
func ReadNDJSONTable(path string, opts ...JSONOption) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadNDJSONTableReader(f, opts...)
}

// ReadNDJSONTableReader parses NDJSON from a reader.
func ReadNDJSONTableReader(r stdio.Reader, opts ...JSONOption) (*Table, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	var records []map[string]any
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(text), &rec); err != nil {
			return nil, fmt.Errorf("NDJSON line %d: %w", line, err)
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return recordsToTable(records), nil
}

// WriteNDJSONTable writes a Table as one JSON object per line.
func WriteNDJSONTable(w stdio.Writer, table *Table) error {
	enc := json.NewEncoder(w)
	for _, row := range table.Rows {
		rec := make(map[string]any, len(table.Columns))
		for j, c := range table.Columns {
			rec[c] = row[j]
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	return nil
}

# duckdb adapter (planned)

Registers DataFrames as DuckDB tables and turns SQL results back into
DataFrames.

Planned API (v0.4), guarded by the `duckdb` build tag so the core stays
dependency-free:

```go
//go:build duckdb

func RegisterDataFrame(db *sql.DB, name string, df *pd.DataFrame) error
func Query(db *sql.DB, query string) (*pd.DataFrame, error)
```

Not yet implemented; this directory only reserves the package path.

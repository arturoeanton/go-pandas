.PHONY: test build examples regen-goldens fmt vet race bench fuzz compat

race:
	go test ./... -race

bench:
	go test ./benchmarks/ -bench=. -benchmem

fuzz:
	go test ./fuzz/ -run=NONE -fuzz=FuzzReadCSV -fuzztime=30s

# Regenerate goldens from real pandas/NumPy and re-run the compat tests.
compat:
	python3 compat/python/run_compat_suite.py

# Print coverage numbers computed from the compatibility matrices.
compat-report:
	go run ./cmd/compat-report

test:
	go test ./...

build:
	go build ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

examples:
	go run ./examples/basic
	go run ./examples/pandas_compat
	go run ./examples/numpy_compat
	go run ./examples/groupby
	go run ./examples/merge
	go run ./examples/io_csv
	go run ./examples/ndarray
	go run ./examples/rolling

# Regenerates compat/goldens from real pandas/NumPy (requires python3
# with pandas and numpy installed).
regen-goldens:
	python3 compat/python/generate_pandas_goldens.py
	python3 compat/python/generate_numpy_goldens.py

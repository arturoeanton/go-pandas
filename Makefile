.PHONY: test build examples regen-goldens fmt vet

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

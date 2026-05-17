# Makefile for the Duniyani blockchain node

.PHONY: all test test-race bench clean

# Default target
all: test

# Run all tests, including integration tests.
# Use -v for verbose output.
test: 
	@echo "Running all tests..."
	@go test ./...

# Run all tests with the race detector enabled.
# This is essential for detecting concurrency issues.
test-race:
	@echo "Running all tests with race detector..."
	@go test -race ./...

# Run all benchmarks and report memory allocations.
# Use this to measure performance and track optimizations.
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Clean the module cache and remove test binaries.
clean:
	@echo "Cleaning up..."
	@go clean -testcache

# Command guide:
# make test      - Run all standard tests.
# make test-race - Run tests with the -race flag to detect data races.
# make bench     - Run all benchmarks to measure performance.
# make clean     - Clean up the test cache.

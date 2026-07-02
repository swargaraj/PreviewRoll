.PHONY: build run test clean

# Build the worker
build:
	go build -o bin/worker ./cmd/worker

# Run the worker
run: build
	./bin/worker

# Run tests
test:
	go test ./... -v

# Clean build artifacts
clean:
	rm -rf bin/

# Run go mod tidy
tidy:
	go mod tidy

# Run golangci-lint
lint:
	golangci-lint run

# Run go fmt
fmt:
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Run all checks
check: fmt vet lint test

# Build the kagi CLI
build:
    go build ./cmd/kagi

# Install the kagi CLI
install:
    go install ./cmd/kagi

# Build and install in one step
all: build install

# Run tests
test:
    go test ./...

# Run tests with coverage
test-coverage:
    go test -cover ./...

# Format code
fmt:
    go fmt ./...

# Run go vet
vet:
    go vet ./...

# Run all checks (fmt, vet, test)
check: fmt vet test

# Clean build artifacts
clean:
    rm -f kagi
    go clean

# Run the CLI with arguments (use: just run "your query here")
run *ARGS:
    go run ./cmd/kagi {{ARGS}}

# Show Go module dependencies
deps:
    go list -m all

# Tidy go.mod
tidy:
    go mod tidy

# Update dependencies
update:
    go get -u ./...
    go mod tidy

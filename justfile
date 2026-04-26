# Justfile for envoke
# Install just: https://github.com/casey/just

# Default recipe - show available commands
default:
    just --list

# Build with all backends included
build:
    go build -tags "1password,keychain,keeper,jumpcloud,aws" -o ee ./cmd/ee

# Build minimal (no backends)
build-minimal:
    go build -o ee ./cmd/ee

# Build with specific backends (example: just build-with keychain,keeper)
build-with *backends:
    go build -tags "{{backends}}" -o ee ./cmd/ee

# Run tests
test:
    go test ./...

# Run tests with all backends
test-all:
    go test -tags "1password,keychain,keeper,jumpcloud,aws" ./...

# Clean build artifacts
clean:
    rm -f ee
    go clean

# Install dependencies
deps:
    go mod download

# Format code
fmt:
    go fmt ./...

# Run linter
lint:
    golangci-lint run

# Install locally (with all backends)
install: build
    cp ee ~/.local/bin/ee 2>/dev/null || cp ee /usr/local/bin/ee

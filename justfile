# Justfile for envoke
# Install just: https://github.com/casey/just

# Default recipe - show available commands
default:
	just --list

# Build with all backends included
build:
	go build -tags "1password,keeper,jumpcloud,aws" -o ee ./cmd/ee

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
	go test -tags "1password,keeper,jumpcloud,aws" ./...

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

# macOS code signing (requires self-signed cert named "envoke-dev", see AGENTS.md)
sign-dev:
	codesign -s "envoke-dev" -f ./ee

# macOS code signing with Developer ID (requires Apple Developer account)
sign-release:
	codesign --deep --force --options runtime --timestamp --sign "Developer ID Application" ./ee
	xcrun notarytool submit ./ee --wait --keychain-profile "notary-profile"
	xcrun stapler staple ./ee

# macOS: build and sign with dev cert
develop: build sign-dev

# Install locally (with all backends)
install: build
	cp ee ~/.local/bin/ee 2>/dev/null || cp ee /usr/local/bin/ee

# Justfile for envoke
# Install just: https://github.com/casey/just
#
# Cross-platform build commands that work on Windows, macOS, and Linux.

# Use PowerShell on Windows for cross-platform compatibility
set windows-shell := ["powershell.exe", "-Command"]

# Default recipe - show available commands
default:
	just --list

# Build with all backends included
# Go automatically adds .exe extension on Windows
build:
	go build -tags "1password,keeper,jumpcloud,aws" ./cmd/ee

# Build minimal (no backends)
# Go automatically adds .exe extension on Windows
build-minimal:
	go build ./cmd/ee

# Build with specific backends (example: just build-with keychain,keeper)
# Go automatically adds .exe extension on Windows
build-with *backends:
	go build -tags "{{backends}}" ./cmd/ee

# Run tests
test:
	go test ./...

# Run tests with all backends
test-all:
	go test -tags "1password,keeper,jumpcloud,aws" ./...

# Clean build artifacts
clean:
	go clean
	-rm -fo ee.exe -ErrorAction SilentlyContinue 2>$null; if ($?) { $null }

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
[unix]
sign-dev:
	codesign -s "envoke-dev" -f ./ee

# macOS code signing with Developer ID (requires Apple Developer account)
[unix]
sign-release:
	codesign --deep --force --options runtime --timestamp --sign "Developer ID Application" ./ee
	xcrun notarytool submit ./ee --wait --keychain-profile "notary-profile"
	xcrun notarytool staple ./ee

# macOS: build and sign with dev cert
[unix]
develop: build sign-dev

# Install locally (with all backends)
[unix]
install: build
	cp ee ~/.local/bin/ee 2>/dev/null || cp ee /usr/local/bin/ee

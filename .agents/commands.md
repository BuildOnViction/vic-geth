# Commands Reference

Standard commands for building, testing, and maintaining this project.

## Build

- `go build ./...` - build all packages
<!-- Project-specific / Build -->

## Test

- `go test ./...` - run all tests
- `go test ./... -run <TestName>` - run a single test
<!-- Project-specific / Test -->
- `git submodule update --init --checkout tests/testdata` - sync the `tests/testdata` submodule to the revision pinned in the commit tree; run this before tests that use those fixtures

## Lint

- `gofmt -l .` - list files that are not formatted
- `gofmt -w .` - format all files in place
- `goimports -w .` - fix imports in place
- `go vet ./...` - run static analysis
<!-- Project-specific / Lint -->

## Dependencies

- `go mod tidy` - sync `go.mod`/`go.sum` with actual imports
- `go mod download` - download dependencies
<!-- Project-specific / Dependencies -->

## Project-specific

<!-- Project-specific -->

This project supports up to Go 1.18. When building or testing, set the environment variable `GOTOOLCHAIN=go1.18.10` to ensure a consistent result across development environments.

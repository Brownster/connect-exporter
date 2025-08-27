# Testing Overview

This project follows the standard Go testing workflow with additional helpers for Prometheus collectors. The main points are outlined below.

## Running Tests

Use the built in Go tooling to run the entire suite:

```bash
go test ./...
```

This command will execute all unit tests across the repository.

## MockConnector

Network calls are avoided during tests by using the `MockConnector` found in `connectors/mock.go`. Each test suite configures the mock with expected responses or errors. This ensures collectors run deterministically and without external dependencies.

## Test Suites and Metric Comparison

Tests are organized with `testify/suite` for clean setup and teardown. Metrics produced by an exporter are validated using `testutil.CollectAndCompare` which compares scraped metrics to an expected output string.

## Test Data

Place any canned API responses or command output under `collectors/testdata/`. Test suites can load these files and register them with the `MockConnector` for more complex scenarios.

## Further Reading

See [DEVELOPER.md](../DEVELOPER.md) for development guidelines and [development-with-mock-api.md](development-with-mock-api.md) for local development setup with the mock API server.

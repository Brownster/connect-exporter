# Developer Guide

This guide provides instructions for building, testing and extending **RSS Exporter**.

## Requirements

- Go 1.24 or newer
- GNU Make (optional for convenience)
- Docker (optional) if you prefer container builds

## Building

Clone the repository and run:

```bash
go build -o rss_exporter .
```

The exporter relies on the minimal `monitoring-maas` framework bundled in this
repository. Scrapers are implemented in the `collectors` package and scheduled
via `maas.ScheduledScraper`. HTTP access is provided through a pluggable
connector located in the `connectors` package. Tests use `MockHTTPConnector`
with canned RSS or Atom feeds stored in `collectors/testdata`.

Each unit test creates its own `MockHTTPConnector` instance and registers the
desired feed responses. This allows the collectors to run without network
access and ensures tests remain deterministic. To add a new scenario, place an
example feed file in `collectors/testdata` and load it in the test suite.

To build a container image:

```bash
docker build -t rss-exporter .
```

### Docker Compose

A `docker-compose.yml` file is included for quick local deployments. Adjust the
mounted configuration path as needed and run:

```bash
docker compose up -d
```

## Running Locally

Create a configuration file based on `config.example.yml` and start the exporter:

```bash
./rss_exporter -config.file=/path/to/config.yml
```

Metrics will be available from `http://<listen_address>:<listen_port>/metrics`.

## Tests

Run the full test suite with:

```bash
go test ./...
```

The tests include sample RSS and Atom feeds under `collectors/testdata` to verify parsing logic for different providers.

## Logging

The exporter uses [Logrus](https://github.com/sirupsen/logrus) for logging. Set `log_level` in the configuration file to `trace`, `debug`, `info`, or `warn` to control verbosity.

## Contributing

1. Fork the repository and create a feature branch.
2. Add tests for your changes.
3. Ensure `go test ./...` succeeds and that `go fmt` produces no diffs.
4. Submit a pull request describing the change.


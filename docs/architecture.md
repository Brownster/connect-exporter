# Architecture Overview

This document describes the internal structure of **RSS Exporter** and how the main components interact.

## Package layout

```
rss-exporter/
├── cmd/rss_exporter/   # Application entry point
│   └── main.go
├── collectors/         # Exporter logic and scrapers
│   ├── feed.go         # maas.ScheduledScraper implementation
│   ├── parsers.go      # Scraper implementations
│   ├── exporter.go     # Creates maas exporter with feed scrapers
│   └── testdata/       # Sample feed files
├── connectors/         # Maas compatible connectors
│   ├── http.go         # HTTP connector implementing maas.Connector
│   └── http_mock.go    # Test helper for mocks
└── internal/fetcher/   # Feed fetching helpers
    └── fetcher.go      # HTTP fetch with retries
```

All production code lives in the `collectors` package. Tests and sample feed files are kept alongside the implementation.

## Main flow

1. **Configuration** is loaded from YAML inside `NewRssExporter` using a `--config.file` flag.
2. `main.go` constructs a `maas.Exporter` via `NewRssExporter` which registers a `maas.ScheduledScraper` for each configured feed.
3. Each scraper periodically fetches its feed and returns metrics via the `maas` framework.
4. Feed items are parsed by a provider-specific scraper chosen by `ScraperForService` and converted to metrics with `maas.NewMetric`.
5. Prometheus metrics are exposed through the exporter when scraped by Prometheus.

## Adding new providers

Implement the `Scraper` interface with `ServiceInfo` and `IncidentKey`. Update `ScraperForService` to return the new scraper when the provider name is requested. Unit tests under `collectors` demonstrate expected behaviour for existing providers.


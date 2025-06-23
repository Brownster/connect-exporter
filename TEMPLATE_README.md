# Monitoring Exporter Template

This repository serves as a template for creating new Prometheus exporters using the Sabio monitoring framework.

## Template Usage

This template is based on the RSS exporter and includes:
- Complete monitoring-maas framework (local copy for development)
- Example collectors with multiple patterns
- Comprehensive test examples
- Production-ready structure

### Creating a New Exporter

1. **Clone this template repository**
2. **Run the setup script** (see `scripts/setup-new-exporter.sh`)
3. **Replace template placeholders** with your exporter details
4. **Implement your collectors** following the patterns in `collectors/`

### Template Structure

```
your-new-exporter/
├── cmd/
│   └── your_exporter/
│       └── main.go              # Main application entry point
├── collectors/
│   ├── exporter.go              # Main exporter constructor
│   ├── example_collector.go     # Example collector implementation
│   ├── example_collector_test.go # Example tests
│   └── testdata/               # Mock data for testing
├── connectors/                 # Connection handling
├── monitoring-maas/            # Framework (remove when moving to production)
├── config.example.yml          # Configuration template
├── Dockerfile                  # Container build
├── docker-compose.yml          # Local development
└── docs/                       # Documentation
```

## Migration to Production

When ready to deploy to company infrastructure:

1. **Remove local monitoring-maas folder**
2. **Update go.mod** to use `github.com/sabio-engineering-product/monitoring-maas`
3. **Update imports** to use the production maas package
4. **Follow company deployment procedures**

## Framework Features

The monitoring-maas framework provides:
- Automatic metric registration
- Scheduled scraping
- HTTP server for metrics endpoint
- Command-line flag handling
- Graceful shutdown
- Connector abstraction for different data sources

## Example Collectors

The template includes several collector patterns:

### HTTP API Collector
Based on the RSS feed collector pattern, suitable for REST APIs and web services.

### File-based Collector
For reading local files or mounted filesystems.

### Command Collector
For executing system commands and parsing output.

See `collectors/` directory for complete examples with tests.

## Development Workflow

1. **Local Development**: Use the included monitoring-maas folder
2. **Testing**: Use mock connectors with testdata
3. **Production**: Switch to company monitoring-maas repo

This approach ensures smooth development while maintaining production compatibility.
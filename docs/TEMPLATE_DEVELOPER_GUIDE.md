# Template Developer Guide: Creating Exporters with Monitoring-MaaS

This comprehensive guide shows how to create new Prometheus exporters using this template repository, which includes the monitoring-maas framework and proven patterns from the RSS exporter.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Template Architecture](#template-architecture)
3. [Collector Patterns](#collector-patterns)
4. [Testing Strategies](#testing-strategies)
5. [Production Migration](#production-migration)
6. [Advanced Patterns](#advanced-patterns)

## Quick Start

### 1. Setup New Exporter

```bash
# Clone this template repository
git clone <template-repo-url> my-new-exporter
cd my-new-exporter

# Run setup script
./scripts/setup-new-exporter.sh my-service sabio-engineering-product

# This creates:
# - cmd/my_service_exporter/main.go
# - collectors/my_service_exporter.go
# - collectors/my_service_collector.go
# - config.template.yml
```

### 2. Implement Your Collector

Edit the generated collector file and implement your `Scrape` method:

```go
func (s *MyServiceScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
    // Your data collection logic here
    response, err := c.Execute("your-command-or-url")
    if err != nil {
        return nil, err
    }
    
    // Parse response and create metrics
    metrics := []maas.Metric{
        maas.NewMetric("service_status", prometheus.GaugeValue, 1.0, []string{"service"}),
    }
    
    return metrics, nil
}
```

### 3. Test Your Collector

```bash
# Run tests
go test ./collectors/...

# Build and run
go build -o my_service_exporter ./cmd/my_service_exporter
./my_service_exporter --config.file=config.template.yml
```

## Template Architecture

### Core Components

```
your-exporter/
├── cmd/your_exporter/
│   └── main.go                 # Application entry point
├── collectors/
│   ├── your_exporter.go        # Main exporter constructor
│   ├── your_collector.go       # Individual collectors
│   ├── your_collector_test.go  # Tests
│   └── testdata/              # Mock data
├── connectors/                # Connection handling
├── monitoring-maas/           # Framework (local development)
├── config.yml                 # Configuration
└── docs/                      # Documentation
```

### Framework Flow

1. **main.go** creates connector and exporter
2. **exporter.go** loads config and registers collectors
3. **collectors** implement `Scraper` interface
4. **monitoring-maas** handles scheduling, HTTP server, metrics

## Collector Patterns

### Pattern 1: HTTP API Collector

Based on the RSS feed collector, perfect for REST APIs:

```go
func (s *APIScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
    response, err := c.Execute(s.config.URL)
    if err != nil {
        return []maas.Metric{
            maas.NewMetric("available", prometheus.GaugeValue, 0, []string{s.config.Name}),
        }, nil // Don't return error - report the down status
    }
    
    // Parse JSON response
    var data APIResponse
    if err := json.Unmarshal([]byte(response.(string)), &data); err != nil {
        return nil, err
    }
    
    return []maas.Metric{
        maas.NewMetric("available", prometheus.GaugeValue, 1, []string{s.config.Name}),
        maas.NewMetric("response_time", prometheus.GaugeValue, data.ResponseTime, []string{s.config.Name}),
    }, nil
}
```

### Pattern 2: Command Execution Collector

For SSH commands or local system monitoring:

```go
func (s *SystemScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
    // Execute command via SSH or locally
    output, err := c.Execute("ps aux | grep myservice | wc -l")
    if err != nil {
        return nil, err
    }
    
    // Parse command output
    processCount := parseFloat(strings.TrimSpace(output.(string)))
    
    return []maas.Metric{
        maas.NewMetric("processes_total", prometheus.GaugeValue, processCount, nil),
    }, nil
}
```

### Pattern 3: File Monitoring Collector

For log files, config files, or file system monitoring:

```go
func (s *FileScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
    // Get file stats
    output, err := c.Execute(fmt.Sprintf("stat -c '%%s %%Y' %s", s.filePath))
    if err != nil {
        return nil, err
    }
    
    parts := strings.Fields(output.(string))
    size := parseFloat(parts[0])
    modTime := parseFloat(parts[1])
    
    return []maas.Metric{
        maas.NewMetric("size_bytes", prometheus.GaugeValue, size, []string{s.filePath}),
        maas.NewMetric("modified_time", prometheus.GaugeValue, modTime, []string{s.filePath}),
    }, nil
}
```

### Pattern 4: SNMP Collector

For network devices and SNMP-enabled systems:

```go
func (s *SNMPScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
    // Execute SNMP walk
    response, err := c.Execute(snmp.Query{
        SNMPType: "walk",
        OIDs:     []string{s.baseOID},
    })
    if err != nil {
        return nil, err
    }
    
    results := response.(map[string]string)
    metrics := []maas.Metric{}
    
    for oid, value := range results {
        // Parse OID to extract interface/component ID
        interfaceID := extractInterfaceFromOID(oid)
        metricValue := parseFloat(value)
        
        metrics = append(metrics, maas.NewMetric(
            "interface_bytes_total",
            prometheus.CounterValue,
            metricValue,
            []string{interfaceID},
        ))
    }
    
    return metrics, nil
}
```

## Testing Strategies

### 1. Mock Connector Testing

The template includes comprehensive testing patterns:

```go
func (s *MyTestSuite) SetupTest() {
    // Create mock connector
    s.mockConnector = connectors.NewMockConnector()
    
    // Add predictable responses
    s.mockConnector.AddResponse("http://api.example.com/health", 
        `{"status": "ok", "uptime": 12345}`)
    
    // Add error responses
    s.mockConnector.AddError("http://broken.example.com/health", 
        "connection refused")
    
    // Create exporter with mock
    s.exporter, _ = NewMyExporter(s.mockConnector, testOptions...)
}
```

### 2. Test Data Management

Following the RSS exporter pattern:

```go
// Put complex responses in testdata files
func (s *MyTestSuite) TestComplexResponse() {
    // Load test data
    testData, _ := os.ReadFile("testdata/complex_response.json")
    s.mockConnector.AddResponse("http://api.example.com/complex", string(testData))
    
    // Test parsing
    s.exporter.Start()
    err := testutil.CollectAndCompare(s.exporter, expectedMetrics, "my_metric")
    s.NoError(err)
}
```

### 3. Metric Validation

```go
func (s *MyTestSuite) TestMetricValues() {
    s.exporter.Start()
    
    // Test exact metric values
    err := testutil.CollectAndCompare(s.exporter, strings.NewReader(`
        # HELP my_service_status Service status
        # TYPE my_service_status gauge
        my_service_status{service="api"} 1
        my_service_status{service="db"} 0
    `), "my_service_status")
    
    s.NoError(err)
}
```

## Production Migration

### Step 1: Remove Local monitoring-maas

```bash
# Remove local framework
rm -rf monitoring-maas/

# Update go.mod
go mod edit -require github.com/sabio-engineering-product/monitoring-maas@latest
go mod edit -require github.com/sabio-engineering-product/monitoring-connectors@latest
```

### Step 2: Update Imports

```go
// Change from:
import maas "github.com/your-org/your-exporter/monitoring-maas"

// To:
import maas "github.com/sabio-engineering-product/monitoring-maas"
import "github.com/sabio-engineering-product/monitoring-connectors/command"
import "github.com/sabio-engineering-product/monitoring-connectors/snmp"
```

### Step 3: Update Connectors

```go
// In main.go, use production connectors:
func main() {
    // For SSH connections
    connector := command.NewSSH()
    
    // For SNMP
    connector := snmp.NewSNMP()
    
    // For HTTP (if using monitoring-connectors)
    connector := http.NewHTTP()
    
    exporter, err := collectors.NewMyExporter(connector)
    // ...
}
```

## Advanced Patterns

### Multi-Service Configuration

```go
type Config struct {
    Services []ServiceConfig `yaml:"services"`
}

type ServiceConfig struct {
    Name      string            `yaml:"name"`
    Type      string            `yaml:"type"`      // "http", "ssh", "snmp"
    URL       string            `yaml:"url"`
    Interval  int               `yaml:"interval"`
    Headers   map[string]string `yaml:"headers"`   // For HTTP
    Commands  []string          `yaml:"commands"`  // For SSH
    OIDs      []string          `yaml:"oids"`      // For SNMP
}
```

### Dynamic Collector Registration

```go
func NewMyExporter(c maas.Connector, options ...func(*maas.Exporter)) (*maas.Exporter, error) {
    // ... load config ...
    
    scrapers := []*maas.ScheduledScraper{}
    
    for _, svc := range cfg.Services {
        switch svc.Type {
        case "http":
            scrapers = append(scrapers, NewHTTPCollector(app, svc))
        case "ssh":
            scrapers = append(scrapers, NewSSHCollector(app, svc))
        case "snmp":
            scrapers = append(scrapers, NewSNMPCollector(app, svc))
        default:
            scrapers = append(scrapers, NewGenericCollector(app, svc))
        }
    }
    
    options = append(options, maas.WithScheduledScrapers(scrapers...))
    return maas.NewExporter(app, c, options...)
}
```

### Error Handling Patterns

```go
func (s *MyScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
    metrics := []maas.Metric{}
    
    // Always try to report service availability
    response, err := c.Execute(s.config.URL)
    if err != nil {
        // Report service as down
        metrics = append(metrics, maas.NewMetric(
            "available", prometheus.GaugeValue, 0, []string{s.config.Name}))
        
        // Log the error but don't fail the scrape
        log.WithError(err).Warn("Failed to scrape service")
        return metrics, nil
    }
    
    // Report service as up
    metrics = append(metrics, maas.NewMetric(
        "available", prometheus.GaugeValue, 1, []string{s.config.Name}))
    
    // Parse response for additional metrics
    // If parsing fails, we still report availability
    if additionalMetrics, err := s.parseResponse(response); err == nil {
        metrics = append(metrics, additionalMetrics...)
    }
    
    return metrics, nil
}
```

### Custom Metric Types

```go
// Counter for continuously increasing values
maas.NewMetric("requests_total", prometheus.CounterValue, totalRequests, labels)

// Gauge for current state values
maas.NewMetric("queue_size", prometheus.GaugeValue, currentQueueSize, labels)

// Histogram for response times (requires special handling)
// Note: The framework primarily supports Gauge and Counter
```

## Best Practices

### 1. Configuration Management

- Use YAML for configuration following the RSS exporter pattern
- Support both file-based and environment variable configuration
- Provide sensible defaults
- Validate configuration at startup

### 2. Error Handling

- Never let one service failure break the entire exporter
- Always try to report service availability
- Log errors but continue scraping other services
- Use timeouts to prevent hanging

### 3. Metric Design

- Use consistent labeling across metrics
- Include service/component identification in labels
- Use descriptive metric names with units
- Register all metrics with help text

### 4. Testing

- Mock all external dependencies
- Test both success and failure scenarios
- Use testdata files for complex responses
- Validate metric values, not just presence

### 5. Performance

- Use appropriate scrape intervals
- Implement timeouts for all external calls
- Consider memory usage for high-cardinality metrics
- Monitor exporter performance itself

## Troubleshooting

### Common Issues

1. **Import Errors**: Ensure go.mod is correctly configured
2. **Metric Not Appearing**: Check metric registration in collector constructor
3. **Test Failures**: Verify mock connector responses match expected format
4. **Performance Issues**: Check scrape intervals and timeout values

### Debugging Tips

```bash
# Enable debug logging
./my_exporter --log.level=debug

# Check metrics endpoint
curl http://localhost:9090/metrics

# Run single test with verbose output
go test -v ./collectors -run TestSpecificTest
```

This template provides a solid foundation for creating robust, testable Prometheus exporters that follow Sabio engineering standards.
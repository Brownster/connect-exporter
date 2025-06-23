# Template Test Data

This directory contains example test data files that can be used as templates for your exporter tests.

## File Organization

```
testdata/
├── template/              # Template examples (this directory)
│   ├── README.md         # This file
│   ├── api_response.json # HTTP API response example
│   ├── command_output.txt # Command execution output example
│   ├── snmp_walk.txt     # SNMP walk output example
│   └── error_response.txt # Error response example
├── your_service_name/    # Your service-specific test data
│   ├── healthy_response.json
│   ├── error_response.json
│   └── ...
```

## Usage Patterns

### 1. HTTP API Testing

For HTTP-based collectors, create JSON files with realistic API responses:

```go
// In your test
mockConnector.AddResponse("https://api.example.com/health", 
    loadTestData("testdata/your_service/healthy_response.json"))
```

### 2. Command Execution Testing

For SSH/command-based collectors, create text files with command output:

```go
// In your test
mockConnector.AddResponse("ps aux | grep myservice", 
    loadTestData("testdata/your_service/process_list.txt"))
```

### 3. SNMP Testing

For SNMP collectors, create files with OID-value mappings:

```go
// In your test
mockConnector.AddResponse(snmp.Query{SNMPType: "walk", OIDs: []string{"1.3.6.1.2.1.1"}},
    loadTestData("testdata/your_service/snmp_system.txt"))
```

## Test Data Guidelines

### File Naming

- Use descriptive names: `healthy_response.json`, `error_condition.json`
- Include version numbers if APIs change: `v1_response.json`, `v2_response.json`
- Use service states: `service_up.json`, `service_down.json`, `service_degraded.json`

### Content Guidelines

1. **Realistic Data**: Use realistic values that your service would actually return
2. **Edge Cases**: Include edge cases like empty responses, extreme values
3. **Error Conditions**: Include various error responses
4. **Multiple States**: Cover all possible service states

### Example Test Structure

```go
func (s *MyTestSuite) SetupTest() {
    s.mockConnector = connectors.NewMockConnector()
    
    // Load test data files
    s.loadTestResponses(map[string]string{
        "https://api.example.com/health": "testdata/my_service/healthy.json",
        "https://api.example.com/metrics": "testdata/my_service/metrics.json",
        "system_command": "testdata/my_service/system_info.txt",
    })
    
    s.exporter = s.createExporter()
}

func (s *MyTestSuite) loadTestResponses(responses map[string]string) {
    for command, filename := range responses {
        data, err := os.ReadFile(filename)
        s.NoError(err)
        s.mockConnector.AddResponse(command, string(data))
    }
}
```

## Helper Functions

Create helper functions for common test patterns:

```go
// Helper to load test data
func loadTestData(filename string) string {
    data, err := os.ReadFile(filename)
    if err != nil {
        panic(fmt.Sprintf("Failed to load test data %s: %v", filename, err))
    }
    return string(data)
}

// Helper to create expected metrics
func expectedMetrics(metricName string, expectedData string) *strings.Reader {
    return strings.NewReader(expectedData)
}

// Helper for testing error conditions
func (s *MyTestSuite) TestErrorCondition() {
    s.mockConnector.AddError("https://api.example.com/health", "connection refused")
    
    s.exporter.Start()
    
    // Verify error metrics are created
    err := testutil.CollectAndCompare(s.exporter, expectedMetrics("my_service_available", `
        # HELP my_service_available Service availability
        # TYPE my_service_available gauge
        my_service_available{service="my_service"} 0
    `), "my_service_available")
    
    s.NoError(err)
}
```
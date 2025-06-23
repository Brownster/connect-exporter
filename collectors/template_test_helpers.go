package collectors

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	"github.com/stretchr/testify/suite"
)

// Template Test Helpers
// These functions provide common patterns for testing exporters

// BaseTestSuite provides common test setup for exporter tests
type BaseTestSuite struct {
	suite.Suite
	MockConnector *connectors.MockConnector
	Exporter      *maas.Exporter
	ServiceName   string
}

// SetupSuite runs once before all tests in the suite
func (s *BaseTestSuite) SetupSuite() {
	s.ServiceName = "test_service"
}

// SetupTest runs before each test method
func (s *BaseTestSuite) SetupTest() {
	s.MockConnector = connectors.NewMockConnector()
	// Override in specific test suites to create your exporter
}

// TearDownTest runs after each test method
func (s *BaseTestSuite) TearDownTest() {
	if s.Exporter != nil {
		// Stop the exporter gracefully
		// Note: The RSS exporter doesn't have a Stop method, but production ones might
	}
}

// TestDataHelpers

// LoadTestData loads test data from a file
func LoadTestData(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("Failed to load test data %s: %v", filename, err))
	}
	return string(data)
}

// LoadTestDataOr loads test data from a file, returns fallback if file doesn't exist
func LoadTestDataOr(filename, fallback string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fallback
	}
	return string(data)
}

// Mock Response Helpers

// AddResponseFromFile adds a mock response loaded from a file
func (s *BaseTestSuite) AddResponseFromFile(command, filename string) {
	data := LoadTestData(filename)
	s.MockConnector.AddResponse(command, data)
}

// AddResponseFromTemplate adds a mock response using a template file with substitutions
func (s *BaseTestSuite) AddResponseFromTemplate(command, templateFile string, substitutions map[string]string) {
	template := LoadTestData(templateFile)
	
	// Simple template substitution
	for key, value := range substitutions {
		template = strings.ReplaceAll(template, "{{"+key+"}}", value)
	}
	
	s.MockConnector.AddResponse(command, template)
}

// AddStandardResponses adds common responses for a service
func (s *BaseTestSuite) AddStandardResponses(serviceName string) {
	// Add healthy response
	s.AddResponseFromFile(
		fmt.Sprintf("https://api.%s.com/health", serviceName),
		fmt.Sprintf("testdata/%s/healthy_response.json", serviceName),
	)
	
	// Add metrics response
	s.AddResponseFromFile(
		fmt.Sprintf("https://api.%s.com/metrics", serviceName),
		fmt.Sprintf("testdata/%s/metrics_response.json", serviceName),
	)
}

// Metric Testing Helpers

// ExpectedMetrics creates a reader with expected Prometheus metrics
func ExpectedMetrics(metrics string) *strings.Reader {
	return strings.NewReader(strings.TrimSpace(metrics))
}

// AssertMetricEquals tests that a metric has the expected value
func (s *BaseTestSuite) AssertMetricEquals(metricName, expected string) {
	s.Exporter.Start()
	err := testutil.CollectAndCompare(s.Exporter, ExpectedMetrics(expected), metricName)
	s.NoError(err, "Metric %s did not match expected value", metricName)
}

// AssertMetricExists tests that a metric exists (regardless of value)
func (s *BaseTestSuite) AssertMetricExists(metricName string) {
	s.Exporter.Start()
	count, err := testutil.GatherAndCount(s.Exporter, metricName)
	s.NoError(err)
	s.Greater(count, 0, "Metric %s should exist", metricName)
}

// AssertMetricCount tests that a metric has a specific number of series
func (s *BaseTestSuite) AssertMetricCount(metricName string, expectedCount int) {
	s.Exporter.Start()
	count, err := testutil.GatherAndCount(s.Exporter, metricName)
	s.NoError(err)
	s.Equal(expectedCount, count, "Metric %s should have %d series", metricName, expectedCount)
}

// Error Testing Helpers

// TestServiceDown tests behavior when a service is unavailable
func (s *BaseTestSuite) TestServiceDown(serviceURL, expectedMetric string) {
	s.MockConnector.AddError(serviceURL, "connection refused")
	
	s.Exporter.Start()
	
	// Should report service as unavailable
	err := testutil.CollectAndCompare(s.Exporter, ExpectedMetrics(fmt.Sprintf(`
		# HELP %s_available Service availability
		# TYPE %s_available gauge
		%s_available{service="%s"} 0
	`, s.ServiceName, s.ServiceName, s.ServiceName, s.ServiceName)), expectedMetric)
	
	s.NoError(err)
}

// TestServiceTimeout tests behavior when a service times out
func (s *BaseTestSuite) TestServiceTimeout(serviceURL string) {
	s.MockConnector.AddResponseWithDelay(serviceURL, `{"status": "ok"}`, 30000) // 30 second delay
	
	// This should timeout and be handled gracefully
	s.Exporter.Start()
	// Add assertions for timeout behavior
}

// Common Test Patterns

// TestHealthyService tests a service in healthy state
func (s *BaseTestSuite) TestHealthyService() {
	s.AddStandardResponses(s.ServiceName)
	s.TestServiceReportsHealthy()
}

// TestServiceReportsHealthy verifies service reports as healthy
func (s *BaseTestSuite) TestServiceReportsHealthy() {
	s.AssertMetricEquals(fmt.Sprintf("%s_available", s.ServiceName), fmt.Sprintf(`
		# HELP %s_available Service availability
		# TYPE %s_available gauge
		%s_available{service="%s"} 1
	`, s.ServiceName, s.ServiceName, s.ServiceName, s.ServiceName))
}

// Configuration Testing Helpers

// TestWithConfig creates an exporter with specific configuration
func (s *BaseTestSuite) TestWithConfig(config interface{}) {
	// This would be implemented based on your specific exporter configuration
	// Example:
	// s.Exporter = NewMyExporter(s.MockConnector, WithConfig(config))
}

// Performance Testing Helpers

// BenchmarkScrape benchmarks a single scrape operation
func BenchmarkScrape(b *testing.B, scraper maas.Scraper, connector maas.Connector) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := scraper.Scrape(connector)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestDataGenerators

// GenerateJSONResponse creates a JSON response with specified fields
func GenerateJSONResponse(status string, fields map[string]interface{}) string {
	result := fmt.Sprintf(`{"status": "%s"`, status)
	
	for key, value := range fields {
		switch v := value.(type) {
		case string:
			result += fmt.Sprintf(`, "%s": "%s"`, key, v)
		case int:
			result += fmt.Sprintf(`, "%s": %d`, key, v)
		case float64:
			result += fmt.Sprintf(`, "%s": %f`, key, v)
		case bool:
			result += fmt.Sprintf(`, "%s": %t`, key, v)
		}
	}
	
	result += "}"
	return result
}

// GenerateCommandOutput creates realistic command output
func GenerateCommandOutput(command string, lines []string) string {
	return strings.Join(lines, "\n")
}

// GenerateSNMPWalk creates SNMP walk output
func GenerateSNMPWalk(baseOID string, values map[string]string) string {
	var lines []string
	for oid, value := range values {
		lines = append(lines, fmt.Sprintf("%s = %s", oid, value))
	}
	return strings.Join(lines, "\n")
}

// Example Usage in Tests:

// ExampleServiceTestSuite demonstrates how to use the base test suite
type ExampleServiceTestSuite struct {
	BaseTestSuite
}

func (s *ExampleServiceTestSuite) SetupTest() {
	s.BaseTestSuite.SetupTest()
	s.ServiceName = "example"
	
	// Create your specific exporter here
	// s.Exporter = NewExampleExporter(s.MockConnector, maas.TestExporterOptions()...)
}

func (s *ExampleServiceTestSuite) TestServiceHealth() {
	// Add mock response
	s.AddResponseFromFile("https://api.example.com/health", "testdata/example/healthy.json")
	
	// Test that service reports as healthy
	s.TestServiceReportsHealthy()
}

func (s *ExampleServiceTestSuite) TestServiceDown() {
	s.TestServiceDown("https://api.example.com/health", "example_available")
}

func TestExampleServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleServiceTestSuite))
}
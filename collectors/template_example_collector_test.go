package collectors

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	"github.com/stretchr/testify/suite"
)

// TEMPLATE TEST EXAMPLES
// These show different testing patterns for your collectors

// SystemCollectorTestSuite demonstrates testing command-based collectors
type SystemCollectorTestSuite struct {
	suite.Suite
	mockConnector *connectors.MockConnector
	exporter      *maas.Exporter
}

func (s *SystemCollectorTestSuite) SetupTest() {
	// Create mock connector with predefined responses
	s.mockConnector = connectors.NewMockConnector()
	
	// Mock system command responses
	s.mockConnector.AddResponse("uname -a && uptime && top -bn1 | grep 'Cpu(s)'", 
		`Linux testhost 5.4.0-42-generic #46-Ubuntu SMP Fri Jul 10 00:24:02 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux
 12:34:56 up 7 days,  2:18,  1 user,  load average: 0.08, 0.02, 0.01
%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.4 id,  0.3 wa,  0.0 hi,  0.0 si,  0.0 st`)

	// Create exporter with mock connector - following RSS exporter pattern
	app := maas.NewTestApp("system_exporter", "Test system exporter")
	scrapers := []*maas.ScheduledScraper{
		NewSystemCollector(app),
	}
	
	var err error
	s.exporter, err = maas.NewExporter(app, s.mockConnector, 
		maas.WithScheduledScrapers(scrapers...),
		maas.WithArgs([]string{"--web.listen-port=9100"}),
		maas.WithLabels(&maas.MockLabels{}),
	)
	s.NoError(err)
}

func (s *SystemCollectorTestSuite) TestSystemInfo() {
	s.exporter.Start()
	
	// Test the info metric
	err := testutil.CollectAndCompare(s.exporter, strings.NewReader(`
		# HELP system_info System information
		# TYPE system_info gauge
		system_info{hostname="testhost",version="5.4.0-42-generic"} 1
	`), "system_info")
	
	s.NoError(err)
}

func (s *SystemCollectorTestSuite) TestSystemUptime() {
	s.exporter.Start()
	
	// Test uptime metric (7 days = 604800 seconds)
	err := testutil.CollectAndCompare(s.exporter, strings.NewReader(`
		# HELP system_uptime_seconds System uptime in seconds
		# TYPE system_uptime_seconds gauge
		system_uptime_seconds 604800
	`), "system_uptime_seconds")
	
	s.NoError(err)
}

func TestSystemCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(SystemCollectorTestSuite))
}

// APICollectorTestSuite demonstrates testing HTTP API collectors
type APICollectorTestSuite struct {
	suite.Suite
	mockConnector *connectors.MockConnector
	exporter      *maas.Exporter
}

func (s *APICollectorTestSuite) SetupTest() {
	s.mockConnector = connectors.NewMockConnector()
	
	// Mock API responses
	s.mockConnector.AddResponse("https://api.example.com/health", 
		`{"status": "ok", "uptime": 12345, "version": "1.2.3"}`)
	
	// Mock slow API response
	s.mockConnector.AddResponseWithDelay("https://slow-api.example.com/health", 
		`{"status": "degraded"}`, 2000) // 2 second delay
	
	// Mock API error
	s.mockConnector.AddError("https://broken-api.example.com/health", 
		"connection refused")

	// Create exporter with API service configs
	app := maas.NewTestApp("api_exporter", "Test API exporter")
	services := []APIService{
		{Name: "fast-api", URL: "https://api.example.com/health", Method: "GET", Interval: 60},
		{Name: "slow-api", URL: "https://slow-api.example.com/health", Method: "GET", Interval: 60},
		{Name: "broken-api", URL: "https://broken-api.example.com/health", Method: "GET", Interval: 60},
	}
	
	scrapers := []*maas.ScheduledScraper{}
	for _, svc := range services {
		scrapers = append(scrapers, NewAPICollector(app, svc))
	}
	
	var err error
	s.exporter, err = maas.NewExporter(app, s.mockConnector,
		maas.WithScheduledScrapers(scrapers...),
		maas.WithArgs([]string{"--web.listen-port=9101"}),
		maas.WithLabels(&maas.MockLabels{}),
	)
	s.NoError(err)
}

func (s *APICollectorTestSuite) TestAPIAvailability() {
	s.exporter.Start()
	
	// Test that healthy API reports as available
	err := testutil.CollectAndCompare(s.exporter, strings.NewReader(`
		# HELP api_available API availability (1=up, 0=down)
		# TYPE api_available gauge
		api_available{endpoint="https://api.example.com/health"} 1
		api_available{endpoint="https://slow-api.example.com/health"} 1
		api_available{endpoint="https://broken-api.example.com/health"} 0
	`), "api_available")
	
	s.NoError(err)
}

func (s *APICollectorTestSuite) TestAPIResponseTime() {
	s.exporter.Start()
	
	// We can't easily test exact response times, but we can verify the metric exists
	// and has reasonable values
	metrics, err := testutil.GatherAndCount(s.exporter, "api_response_time_seconds")
	s.NoError(err)
	s.Equal(3, metrics) // Should have 3 services
}

func TestAPICollectorTestSuite(t *testing.T) {
	suite.Run(t, new(APICollectorTestSuite))
}

// FileCollectorTestSuite demonstrates testing file-based collectors
type FileCollectorTestSuite struct {
	suite.Suite
	mockConnector *connectors.MockConnector
	exporter      *maas.Exporter
}

func (s *FileCollectorTestSuite) SetupTest() {
	s.mockConnector = connectors.NewMockConnector()
	
	// Mock file stat responses
	s.mockConnector.AddResponse("stat -c '%s %Y' /var/log/app.log && wc -l /var/log/app.log",
		`1024 1640995200
42 /var/log/app.log`)
	
	s.mockConnector.AddResponse("stat -c '%s %Y' /etc/config.yml && wc -l /etc/config.yml",
		`512 1640995100
15 /etc/config.yml`)

	// Create exporter with file collectors
	app := maas.NewTestApp("file_exporter", "Test file exporter")
	scrapers := []*maas.ScheduledScraper{
		NewFileCollector(app, "/var/log/app.log"),
		NewFileCollector(app, "/etc/config.yml"),
	}
	
	var err error
	s.exporter, err = maas.NewExporter(app, s.mockConnector,
		maas.WithScheduledScrapers(scrapers...),
		maas.WithArgs([]string{"--web.listen-port=9102"}),
		maas.WithLabels(&maas.MockLabels{}),
	)
	s.NoError(err)
}

func (s *FileCollectorTestSuite) TestFileSize() {
	s.exporter.Start()
	
	err := testutil.CollectAndCompare(s.exporter, strings.NewReader(`
		# HELP file_size_bytes File size in bytes
		# TYPE file_size_bytes gauge
		file_size_bytes{path="/var/log/app.log"} 1024
		file_size_bytes{path="/etc/config.yml"} 512
	`), "file_size_bytes")
	
	s.NoError(err)
}

func (s *FileCollectorTestSuite) TestFileLines() {
	s.exporter.Start()
	
	err := testutil.CollectAndCompare(s.exporter, strings.NewReader(`
		# HELP file_lines_total Total lines in file
		# TYPE file_lines_total counter
		file_lines_total{path="/var/log/app.log"} 42
		file_lines_total{path="/etc/config.yml"} 15
	`), "file_lines_total")
	
	s.NoError(err)
}

func TestFileCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(FileCollectorTestSuite))
}

// TEMPLATE TEST PATTERNS:
//
// 1. Mock Connector Pattern:
//    - Use connectors.NewMockConnector() for predictable responses
//    - AddResponse() for successful calls
//    - AddError() for error conditions
//    - AddResponseWithDelay() for timing tests
//
// 2. Test Suite Pattern:
//    - Use testify/suite for organized tests
//    - SetupTest() creates fresh exporter for each test
//    - Multiple test methods for different scenarios
//
// 3. Metric Validation:
//    - testutil.CollectAndCompare() for exact metric matching
//    - testutil.GatherAndCount() for counting metrics
//    - Use Prometheus exposition format in expected strings
//
// 4. File-based Test Data:
//    - Put complex mock responses in testdata/ files
//    - Load with os.ReadFile() in tests
//    - Mirrors the RSS exporter testdata pattern
//
// 5. Edge Case Testing:
//    - Test error conditions (network failures, invalid responses)
//    - Test empty responses
//    - Test malformed data
//    - Test timeouts and delays
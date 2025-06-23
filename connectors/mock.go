package connectors

import (
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
)

// MockConnector implements maas.Connector for testing purposes.
// This provides predictable responses for unit tests following create_template_repo.md patterns.
type MockConnector struct {
	responses map[string]string
	errors    map[string]error
	delays    map[string]time.Duration
	callCount map[string]int
}

// NewMockConnector creates a new mock connector for testing.
func NewMockConnector() *MockConnector {
	return &MockConnector{
		responses: make(map[string]string),
		errors:    make(map[string]error),
		delays:    make(map[string]time.Duration),
		callCount: make(map[string]int),
	}
}

// Connect implements the maas.Connector interface (no-op for mock).
func (m *MockConnector) Connect() error {
	return nil
}

// Flags implements the maas.Connector interface (no flags for mock).
func (m *MockConnector) Flags(app *kingpin.Application) {
	// No flags needed for mock connector
}

// Execute returns predefined responses for testing.
func (m *MockConnector) Execute(query interface{}) (interface{}, error) {
	// Convert query to string key
	var key string
	switch q := query.(type) {
	case string:
		key = q
	case HTTPQuery:
		key = q.URL
	case TemplateHTTPQuery:
		key = q.URL
	default:
		key = fmt.Sprintf("%+v", query)
	}

	// Increment call count
	m.callCount[key]++

	// Simulate delay if configured
	if delay, exists := m.delays[key]; exists {
		time.Sleep(delay)
	}

	// Return error if configured
	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	// Return response if configured
	if response, exists := m.responses[key]; exists {
		return response, nil
	}

	// Default response for unknown queries
	return fmt.Sprintf("mock response for: %s", key), nil
}

// AddResponse adds a predefined response for a query.
func (m *MockConnector) AddResponse(query, response string) {
	m.responses[query] = response
}

// AddError adds a predefined error for a query.
func (m *MockConnector) AddError(query string, err error) {
	m.errors[query] = err
}

// AddErrorString adds a predefined error message for a query.
func (m *MockConnector) AddErrorString(query, errMsg string) {
	m.errors[query] = fmt.Errorf(errMsg)
}

// AddResponseWithDelay adds a response with a simulated delay.
func (m *MockConnector) AddResponseWithDelay(query, response string, delayMs int) {
	m.responses[query] = response
	m.delays[query] = time.Duration(delayMs) * time.Millisecond
}

// GetCallCount returns the number of times a query was executed.
func (m *MockConnector) GetCallCount(query string) int {
	return m.callCount[query]
}

// ClearResponses clears all configured responses.
func (m *MockConnector) ClearResponses() {
	m.responses = make(map[string]string)
	m.errors = make(map[string]error)
	m.delays = make(map[string]time.Duration)
	m.callCount = make(map[string]int)
}

// LoadResponsesFromMap loads multiple responses at once.
func (m *MockConnector) LoadResponsesFromMap(responses map[string]string) {
	for query, response := range responses {
		m.responses[query] = response
	}
}

// AddResponseFromFile adds a response loaded from a file (helper for tests).
func (m *MockConnector) AddResponseFromFile(query, content string) {
	// Remove any trailing whitespace that might cause test failures
	m.responses[query] = strings.TrimSpace(content)
}

// Template helper methods for common test scenarios

// AddHealthyAPIResponse adds a typical healthy API response.
func (m *MockConnector) AddHealthyAPIResponse(url string) {
	m.AddResponse(url, `{
		"status": "healthy",
		"timestamp": "2024-01-15T10:30:00Z",
		"uptime_seconds": 86400,
		"version": "1.0.0"
	}`)
}

// AddUnhealthyAPIResponse adds a typical unhealthy API response.
func (m *MockConnector) AddUnhealthyAPIResponse(url string) {
	m.AddResponse(url, `{
		"status": "error",
		"timestamp": "2024-01-15T10:30:00Z",
		"error": "Database connection failed"
	}`)
}

// AddSystemCommandResponse adds typical system command outputs.
func (m *MockConnector) AddSystemCommandResponse() {
	// Add common system commands
	m.AddResponse("uname -a", "Linux testhost 5.4.0-42-generic #46-Ubuntu SMP x86_64")
	m.AddResponse("uptime", " 10:30:15 up 7 days,  2:18,  3 users,  load average: 0.15, 0.12, 0.08")
	m.AddResponse("df -h", `Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        50G   22G   26G  46% /
/dev/sda2       100G   45G   50G  48% /var`)
}

// MockQuery represents a mock query for testing.
type MockQuery struct {
	Command string
	Args    []string
}

// String returns the string representation of a mock query.
func (q MockQuery) String() string {
	if len(q.Args) > 0 {
		return fmt.Sprintf("%s %s", q.Command, strings.Join(q.Args, " "))
	}
	return q.Command
}
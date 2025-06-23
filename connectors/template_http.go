package connectors

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
)

// TemplateHTTPConnector implements maas.Connector for HTTP-based monitoring.
// This is a template example for HTTP API monitoring, following create_template_repo.md patterns.
type TemplateHTTPConnector struct {
	client  *http.Client
	timeout time.Duration
	logger  *logrus.Entry
}

// NewTemplateHTTPConnector creates a new HTTP connector for template usage.
func NewTemplateHTTPConnector() *TemplateHTTPConnector {
	return &TemplateHTTPConnector{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
		logger:  logrus.WithField("component", "template_http_connector"),
	}
}

// Connect implements the maas.Connector interface.
// For HTTP connections, this is typically a no-op.
func (c *TemplateHTTPConnector) Connect() error {
	c.logger.Info("HTTP connector initialized")
	return nil
}

// Flags implements the maas.Connector interface.
// Add any HTTP-specific configuration flags here.
func (c *TemplateHTTPConnector) Flags(app *kingpin.Application) {
	app.Flag("http.timeout", "HTTP request timeout").
		Default("30s").
		DurationVar(&c.timeout)
		
	// TEMPLATE: Add other HTTP-specific flags as needed
	// app.Flag("http.user-agent", "HTTP User-Agent header").
	//     Default("TemplateExporter/1.0").
	//     StringVar(&c.userAgent)
}

// Execute performs an HTTP request.
// For template usage, the query parameter should be a URL string.
func (c *TemplateHTTPConnector) Execute(query interface{}) (interface{}, error) {
	// Handle different query types
	var url string
	
	switch q := query.(type) {
	case string:
		// Simple URL string
		url = q
	// TEMPLATE: Add support for structured queries as needed
	// case CustomQueryType:
	//     return c.executeCustom(q)
	default:
		return nil, fmt.Errorf("unsupported query type: %T", query)
	}

	// Perform simple GET request
	c.logger.WithField("url", url).Debug("Executing HTTP request")
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("User-Agent", "TemplateExporter/1.0")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// TemplateHTTPQuery represents an advanced HTTP query with headers, method, etc.
type TemplateHTTPQuery struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

// executeAdvanced handles advanced HTTP queries with custom headers, methods, etc.
func (c *TemplateHTTPConnector) executeAdvanced(query TemplateHTTPQuery) (interface{}, error) {
	method := query.Method
	if method == "" {
		method = "GET"
	}

	c.logger.WithFields(logrus.Fields{
		"url":    query.URL,
		"method": method,
	}).Debug("Executing advanced HTTP request")

	var body io.Reader
	if query.Body != "" {
		body = io.NopCloser(io.MultiReader(nil))
	}

	req, err := http.NewRequest(method, query.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set custom headers
	for key, value := range query.Headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "TemplateExporter/1.0")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(responseBody), nil
}
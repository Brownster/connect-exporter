package maas

import (
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Template Helper Functions
// These functions make it easier to use monitoring-maas in template exporters

// NewTestApp creates a kingpin application for testing
// This is useful in test suites where you need a basic app configuration
func NewTestApp(name, description string) *kingpin.Application {
	return kingpin.New(name, description).DefaultEnvars()
}

// QuickCollector creates a collector with sensible defaults for simple use cases
// This reduces boilerplate for basic collectors
func QuickCollector(name string, scraper Scraper, frequency, timeout string, metrics map[string]string) *ScheduledScraper {
	app := NewTestApp("quick", "Quick collector")
	
	// Parse frequency and timeout
	freq, _ := parseFrequency(frequency)
	tout, _ := parseTimeout(timeout)
	
	// Create scheduled scraper
	ss := NewScheduledScraper(name, scraper,
		WithSchedule(NewSchedule(
			WithFrequency(freq),
			WithTimeout(tout),
		)),
	)
	
	// Register metrics
	for metricName, help := range metrics {
		// Simple metrics with no labels by default
		ss.descriptions[metricName] = prometheus.NewDesc(
			prometheus.BuildFQName("", name, metricName),
			help,
			nil, // No variable labels
			nil, // No constant labels
		)
	}
	
	return ss
}

// Template patterns for common metric types

// NewGaugeMetric creates a gauge metric - for values that can go up and down
func NewGaugeMetric(name string, value float64, labels []string) Metric {
	return NewMetric(name, prometheus.GaugeValue, value, labels)
}

// NewCounterMetric creates a counter metric - for values that only increase
func NewCounterMetric(name string, value float64, labels []string) Metric {
	return NewMetric(name, prometheus.CounterValue, value, labels)
}

// NewInfoMetric creates an info metric - typically has value 1 with descriptive labels
func NewInfoMetric(name string, labels []string) Metric {
	return NewMetric(name, prometheus.GaugeValue, 1, labels)
}

// NewStatusMetric creates a status metric - 1 for up/ok, 0 for down/error
func NewStatusMetric(name string, isUp bool, labels []string) Metric {
	value := 0.0
	if isUp {
		value = 1.0
	}
	return NewMetric(name, prometheus.GaugeValue, value, labels)
}

// Template Scraper Interface Helpers

// SimpleScraper provides a basic implementation that templates can embed
type SimpleScraper struct {
	name        string
	description string
}

func NewSimpleScraper(name, description string) *SimpleScraper {
	return &SimpleScraper{
		name:        name,
		description: description,
	}
}

// GetName returns the scraper name - useful for logging and identification
func (s *SimpleScraper) GetName() string {
	return s.name
}

// GetDescription returns the scraper description
func (s *SimpleScraper) GetDescription() string {
	return s.description
}

// Template Configuration Helpers

// CommonFlags adds standard flags that most exporters need
func CommonFlags(app *kingpin.Application) {
	// These are commonly needed by exporters
	app.Flag("config.file", "Path to configuration file").Default("config.yml").String()
	app.Flag("dry-run", "Parse config and exit without starting").Bool()
	app.Flag("check-config", "Check configuration and exit").Bool()
}

// ServiceConfig represents a common service configuration pattern
type ServiceConfig struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	URL      string            `yaml:"url"`
	Interval int               `yaml:"interval"`
	Timeout  int               `yaml:"timeout"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty"`
	Enabled  bool              `yaml:"enabled"`
}

// GetInterval returns the scrape interval with a default
func (sc ServiceConfig) GetInterval() int {
	if sc.Interval <= 0 {
		return 300 // 5 minutes default
	}
	return sc.Interval
}

// GetTimeout returns the timeout with a default
func (sc ServiceConfig) GetTimeout() int {
	if sc.Timeout <= 0 {
		return 30 // 30 seconds default
	}
	return sc.Timeout
}

// IsEnabled returns whether the service is enabled (default true)
func (sc ServiceConfig) IsEnabled() bool {
	// Default to enabled if not specified
	return sc.Enabled != false
}

// Template Error Handling Patterns

// ScrapeError represents a scraping error with context
type ScrapeError struct {
	Service string
	URL     string
	Err     error
}

func (e ScrapeError) Error() string {
	return fmt.Sprintf("scrape failed for service %s at %s: %v", e.Service, e.URL, e.Err)
}

// NewScrapeError creates a new scrape error
func NewScrapeError(service, url string, err error) error {
	return ScrapeError{
		Service: service,
		URL:     url,
		Err:     err,
	}
}

// HandleScrapeError is a helper for consistent error handling in scrapers
// It returns metrics indicating the service is down and logs the error
func HandleScrapeError(serviceName string, err error, labels []string) []Metric {
	log.WithError(err).Warnf("Scrape failed for service %s", serviceName)
	
	return []Metric{
		NewStatusMetric("available", false, labels),
		NewGaugeMetric("last_scrape_error", 1, labels),
	}
}

// Template Testing Helpers

// MockLabels provides a simple label implementation for testing
type MockLabels struct{}

func (m *MockLabels) Flags(app *kingpin.Application) {}
func (m *MockLabels) Get() map[string]string       { return map[string]string{} }

// TestExporterOptions returns common options for test exporters
func TestExporterOptions() []func(*Exporter) {
	return []func(*Exporter){
		WithArgs([]string{"--web.listen-port=9100"}),
		WithLabels(&MockLabels{}),
	}
}

// Template Documentation Helpers

// MetricInfo holds information about a metric for documentation
type MetricInfo struct {
	Name        string
	Type        string
	Help        string
	Labels      []string
	Example     string
}

// DocumentedMetrics can be implemented by scrapers to provide metric documentation
type DocumentedMetrics interface {
	GetMetricInfo() []MetricInfo
}

// GenerateMarkdownDocs creates markdown documentation from metric info
func GenerateMarkdownDocs(scraperName string, metrics []MetricInfo) string {
	var doc strings.Builder
	
	doc.WriteString(fmt.Sprintf("# %s Metrics\n\n", scraperName))
	
	for _, metric := range metrics {
		doc.WriteString(fmt.Sprintf("## %s\n\n", metric.Name))
		doc.WriteString(fmt.Sprintf("**Type:** %s\n\n", metric.Type))
		doc.WriteString(fmt.Sprintf("**Help:** %s\n\n", metric.Help))
		
		if len(metric.Labels) > 0 {
			doc.WriteString("**Labels:**\n")
			for _, label := range metric.Labels {
				doc.WriteString(fmt.Sprintf("- `%s`\n", label))
			}
			doc.WriteString("\n")
		}
		
		if metric.Example != "" {
			doc.WriteString("**Example:**\n```\n")
			doc.WriteString(metric.Example)
			doc.WriteString("\n```\n\n")
		}
	}
	
	return doc.String()
}

// Internal helper functions

func parseFrequency(freq string) (time.Duration, error) {
	return time.ParseDuration(freq)
}

func parseTimeout(timeout string) (time.Duration, error) {
	return time.ParseDuration(timeout)
}
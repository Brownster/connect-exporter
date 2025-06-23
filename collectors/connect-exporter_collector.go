package collectors

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/alecthomas/kingpin/v2"
)

// NewExampleCollector creates a scheduled scraper for example metrics.
func NewExampleCollector(app *kingpin.Application, config ServiceConfig) *maas.ScheduledScraper {
	return maas.NewScheduledScraper("example", NewExampleScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(time.Duration(config.Interval)*time.Second),
			maas.WithTimeout(30*time.Second),
		)),
		// TEMPLATE: Register your metrics here
		maas.WithDescription(app, "status", "Service status (0=down, 1=up)", []string{"service", "endpoint"}),
		maas.WithDescription(app, "response_time_seconds", "Response time in seconds", []string{"service", "endpoint"}),
	)
}

// ExampleScraper holds state for scraping example metrics.
type ExampleScraper struct {
	config ServiceConfig
}

// NewExampleScraper creates a new scraper instance.
func NewExampleScraper(app *kingpin.Application, config ServiceConfig) *ExampleScraper {
	return &ExampleScraper{
		config: config,
	}
}

// Scrape performs the actual data collection.
func (s *ExampleScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// TEMPLATE: Implement your data collection logic here
	
	// Example: HTTP request to check service health
	response, err := c.Execute(s.config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", s.config.URL, err)
	}

	// TEMPLATE: Parse your response and extract metrics
	// This is a simple example - replace with your parsing logic
	responseStr := response.(string)
	
	metrics := []maas.Metric{}
	
	// Example status metric
	status := 1.0 // Assume success if no error
	if len(responseStr) == 0 {
		status = 0.0
	}
	
	metrics = append(metrics, maas.NewMetric(
		"status",
		prometheus.GaugeValue,
		status,
		[]string{s.config.Name, s.config.URL},
	))
	
	// Example response time metric (mock data)
	metrics = append(metrics, maas.NewMetric(
		"response_time_seconds",
		prometheus.GaugeValue,
		0.123, // TEMPLATE: Replace with actual response time
		[]string{s.config.Name, s.config.URL},
	))

	return metrics, nil
}

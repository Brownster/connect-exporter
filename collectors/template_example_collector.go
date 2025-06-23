package collectors

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/alecthomas/kingpin/v2"
)

// TEMPLATE COLLECTOR EXAMPLE
// This shows different collector patterns you can use in your exporter

// NewSystemCollector creates a scheduled scraper for system metrics.
// This example shows how to create a collector that runs commands via SSH/local execution.
func NewSystemCollector(app *kingpin.Application) *maas.ScheduledScraper {
	const defaultFrequency = 15 * time.Second
	const defaultTimeout = 5 * time.Second

	return maas.NewScheduledScraper("system", NewSystemScraper(app),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(defaultFrequency),
			maas.WithTimeout(defaultTimeout),
		)),
		// Register metrics - these create help text and --no-system.metric-name flags
		maas.WithDescription(app, "info", "System information", []string{"version", "hostname"}),
		maas.WithDescription(app, "uptime_seconds", "System uptime in seconds", nil),
		maas.WithDescription(app, "cpu_usage_percent", "CPU usage percentage", []string{"cpu"}),
	)
}

// SystemScraper demonstrates parsing structured command output
type SystemScraper struct {
	uptimeRegex *regexp.Regexp
	cpuRegex    *regexp.Regexp
}

func NewSystemScraper(app *kingpin.Application) *SystemScraper {
	return &SystemScraper{
		uptimeRegex: regexp.MustCompile(`up\s+(\d+)\s+days?`),
		cpuRegex:    regexp.MustCompile(`CPU(\d+):\s+(\d+\.?\d*)%`),
	}
}

func (s *SystemScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// TEMPLATE: This shows how to execute commands and parse output
	// For SSH: use command.NewSSH() connector
	// For local: use command.NewLocal() connector
	// For HTTP APIs: use your HTTP connector
	
	// Example 1: Get system info
	output, err := c.Execute("uname -a && uptime && top -bn1 | grep 'Cpu(s)'")
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	lines := strings.Split(output.(string), "\n")
	metrics := []maas.Metric{}

	// Parse system info
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			metrics = append(metrics, maas.NewMetric(
				"info",
				prometheus.GaugeValue,
				1, // Info metrics typically have value 1
				[]string{parts[2], parts[1]}, // version, hostname
			))
		}
	}

	// Parse uptime
	for _, line := range lines {
		if matches := s.uptimeRegex.FindStringSubmatch(line); len(matches) > 1 {
			days := parseFloat(matches[1])
			metrics = append(metrics, maas.NewMetric(
				"uptime_seconds",
				prometheus.GaugeValue,
				days*24*3600, // Convert days to seconds
				nil,
			))
		}
	}

	// Parse CPU usage (example pattern)
	for _, line := range lines {
		if matches := s.cpuRegex.FindAllStringSubmatch(line, -1); len(matches) > 0 {
			for _, match := range matches {
				if len(match) >= 3 {
					cpuId := match[1]
					usage := parseFloat(match[2])
					metrics = append(metrics, maas.NewMetric(
						"cpu_usage_percent",
						prometheus.GaugeValue,
						usage,
						[]string{cpuId},
					))
				}
			}
		}
	}

	return metrics, nil
}

// NewAPICollector shows how to create a collector for REST APIs
// Based on the RSS feed collector pattern but generic
func NewAPICollector(app *kingpin.Application, service APIService) *maas.ScheduledScraper {
	return maas.NewScheduledScraper("api", NewAPIScraper(app, service),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(time.Duration(service.Interval)*time.Second),
			maas.WithTimeout(30*time.Second),
		)),
		// API-specific metrics
		maas.WithDescription(app, "response_time_seconds", "API response time", []string{"endpoint", "method"}),
		maas.WithDescription(app, "status_code", "HTTP status code", []string{"endpoint", "method"}),
		maas.WithDescription(app, "available", "API availability (1=up, 0=down)", []string{"endpoint"}),
	)
}

type APIService struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Method   string `yaml:"method"`
	Interval int    `yaml:"interval"`
}

type APIScraper struct {
	service APIService
}

func NewAPIScraper(app *kingpin.Application, service APIService) *APIScraper {
	return &APIScraper{service: service}
}

func (s *APIScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// TEMPLATE: This shows HTTP API interaction pattern
	start := time.Now()
	
	// Execute HTTP request through connector
	response, err := c.Execute(s.service.URL)
	duration := time.Since(start).Seconds()
	
	metrics := []maas.Metric{}
	
	// Response time metric
	metrics = append(metrics, maas.NewMetric(
		"response_time_seconds",
		prometheus.GaugeValue,
		duration,
		[]string{s.service.URL, s.service.Method},
	))
	
	if err != nil {
		// API is down
		metrics = append(metrics, maas.NewMetric(
			"available",
			prometheus.GaugeValue,
			0,
			[]string{s.service.URL},
		))
		
		// Don't return error - we want to report the down status
		return metrics, nil
	}
	
	// API is up
	metrics = append(metrics, maas.NewMetric(
		"available",
		prometheus.GaugeValue,
		1,
		[]string{s.service.URL},
	))
	
	// TEMPLATE: Parse response for additional metrics
	// responseStr := response.(string)
	// ... parse JSON/XML/etc and extract metrics
	
	return metrics, nil
}

// NewFileCollector shows how to monitor file-based data sources
func NewFileCollector(app *kingpin.Application, filePath string) *maas.ScheduledScraper {
	return maas.NewScheduledScraper("file", NewFileScraper(app, filePath),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(60*time.Second),
			maas.WithTimeout(10*time.Second),
		)),
		maas.WithDescription(app, "size_bytes", "File size in bytes", []string{"path"}),
		maas.WithDescription(app, "modified_time", "File modification timestamp", []string{"path"}),
		maas.WithDescription(app, "lines_total", "Total lines in file", []string{"path"}),
	)
}

type FileScraper struct {
	filePath string
}

func NewFileScraper(app *kingpin.Application, filePath string) *FileScraper {
	return &FileScraper{filePath: filePath}
}

func (s *FileScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// TEMPLATE: File monitoring pattern
	// This could read local files or files via SSH/SFTP
	
	// Get file stats
	output, err := c.Execute(fmt.Sprintf("stat -c '%%s %%Y' %s && wc -l %s", s.filePath, s.filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", s.filePath, err)
	}
	
	lines := strings.Split(strings.TrimSpace(output.(string)), "\n")
	metrics := []maas.Metric{}
	
	// Parse stat output: size modification_time
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			size := parseFloat(parts[0])
			modTime := parseFloat(parts[1])
			
			metrics = append(metrics,
				maas.NewMetric("size_bytes", prometheus.GaugeValue, size, []string{s.filePath}),
				maas.NewMetric("modified_time", prometheus.GaugeValue, modTime, []string{s.filePath}),
			)
		}
	}
	
	// Parse wc output: line_count filename
	if len(lines) > 1 {
		parts := strings.Fields(lines[1])
		if len(parts) >= 1 {
			lineCount := parseFloat(parts[0])
			metrics = append(metrics, maas.NewMetric(
				"lines_total",
				prometheus.CounterValue,
				lineCount,
				[]string{s.filePath},
			))
		}
	}
	
	return metrics, nil
}

// Helper function for parsing strings to float64
func parseFloat(s string) float64 {
	// TEMPLATE: Add proper error handling for production use
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
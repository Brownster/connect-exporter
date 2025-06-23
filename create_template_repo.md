Developer Guide: Creating a Sabio Prometheus Exporter
1. Introduction

This guide provides a step-by-step process for creating a new Prometheus exporter that aligns with our internal standards and tooling. Our exporters are Go applications that poll a target system (via SSH, SNMP, etc.), collect metrics, and expose them on an HTTP endpoint for Prometheus to scrape.

The core of our ecosystem relies on two internal libraries:

    monitoring-maas: A framework that handles the web server, metric registration, scheduling, and command-line flag parsing.

    monitoring-connectors: A library providing standardized connectors to interact with target systems (e.g., via SSH commands or SNMP queries).

By following this guide, you will create an exporter that is consistent, testable, and easily integrated into our monitoring stack.
2. Project Structure

Your new exporter project should follow this standardized folder structure. The easiest way to start is to copy an existing exporter (like monitoring-linux-exporter) and modify its contents.
Generated code

      
your-new-exporter/
├── alerts/
│   └── *.yaml              # Prometheus alerting rules
├── cmd/
│   └── your_exporter/
│       └── main.go         # Main application entry point
├── collectors/
│   ├── exporter.go         # Main exporter constructor
│   ├── collector_A.go      # Logic for scraping one category of metrics
│   ├── collector_A_test.go # Tests for collector A
│   ├── collector_B.go
│   ├── collector_B_test.go
│   └── testdata/           # Mock data for testing
│       └── ...
├── dashboards/
│   └── *.json              # Grafana dashboard definitions
├── .github/                # GitHub Actions and config
├── .golangci.yml           # Go linter configuration
├── CODEOWNERS              # Code ownership for GitHub
├── Dockerfile              # Docker build file
├── go.mod                  # Go module definition
├── go.sum
└── README.md

    

IGNORE_WHEN_COPYING_START
Use code with caution.
IGNORE_WHEN_COPYING_END
3. The Core Logic: The /collectors Directory

This is where you will spend most of your development time. The logic is split into multiple files, each responsible for a group of related metrics.
3.1. The Main Exporter File (collectors/exporter.go)

This file acts as the central hub for your exporter. Its primary responsibility is to define a constructor function that initializes the maas.Exporter and registers all the individual metric collectors.

Key Responsibilities:

    Define a NewYourExporter function: This function takes a maas.Connector and returns a maas.Exporter.

    Initialize kingpin: This is our standard library for command-line flags. The maas library builds upon it.

    Register Collectors: Use maas.WithScheduledScrapers to add each of your collector initializers (e.g., NewCPUCollector(app)).

Example (collectors/exporter.go):
Generated go

      
package collectors

import (
	maas "github.com/sabio-engineering-product/monitoring-maas"
	"gopkg.in/alecthomas/kingpin.v2"
)

// NewYourExporter creates a new exporter for YourSystem.
func NewYourExporter(c maas.Connector, options ...func(*maas.Exporter)) (*maas.Exporter, error) {
	// Initialize kingpin application. Always give it a name and a description.
	app := kingpin.New("yourexporter", "Your Exporter description").DefaultEnvars()

	// Add all your collectors here.
	options = append(options, maas.WithScheduledScrapers(
		NewCPUCollector(app),
		NewMemoryCollector(app),
		NewLicenseCollector(app),
	))

    // Create the final exporter object.
	e, err := maas.NewExporter(app, c, options...)

	return e, err
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END
3.2. Individual Collector Files (e.g., collectors/cpu.go)

For each category of metrics (e.g., CPU, Memory, Calls), you will create a dedicated _collector.go file. Each file contains a Scraper struct and a constructor function.

A. The Constructor (New...Collector)

This function creates and configures a maas.ScheduledScraper.

    maas.NewScheduledScraper(): Takes a name (which becomes a prefix for all metrics from this collector, e.g., linux_cpu_...) and an instance of your Scraper struct.

    maas.WithSchedule(): Defines the scrape frequency and timeout. You can also disable a collector by default, requiring a flag to enable it (see restart.go in the linux exporter).

    maas.WithDescription(): This is crucial. For every metric your scraper will produce, you must register it here. This automatically creates a help text and a --no-metric.name flag to disable it.

        The parameters are: the kingpin app, the metric name (without prefix), the HELP text, and a slice of Prometheus label names.

Example (collectors/cpu.go):
Generated go

      
package collectors

import (
    "time"
	"github.com/prometheus/client_golang/prometheus"
	maas "github.com/sabio-engineering-product/monitoring-maas"
	"gopkg.in/alecthomas/kingpin.v2"
)

// NewCPUCollector creates a scheduled scraper for CPU metrics.
func NewCPUCollector(app *kingpin.Application) *maas.ScheduledScraper {
	const defaultFrequency = 15 * time.Second
	const defaultTimeout = 5 * time.Second

	return maas.NewScheduledScraper("cpu", NewCPUScraper(app),
		// Configure the schedule.
		maas.WithSchedule(maas.NewSchedule(
		    maas.WithFrequency(defaultFrequency), 
		    maas.WithTimeout(defaultTimeout),
		)),
		// Register the metric. The final name will be `yourexporter_cpu_usage_percent`.
		maas.WithDescription(app, "usage_percent", "CPU utilization", []string{"id"}),
	)
}

// ... Scraper struct and Scrape method ...

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END

B. The Scraper Struct and Scrape Method

This is where the data collection happens.

    Define a YourScraper struct: This struct holds any necessary state, like pre-compiled regular expressions or SNMP OIDs.

    Implement the Scrape method: func (s *YourScraper) Scrape(c maas.Connector) ([]maas.Metric, error)

        It receives the maas.Connector interface.

        Use the connector to execute a command (c.Execute("cat /proc/cpuinfo")) or an SNMP query. The connector handles the underlying communication.

        Parse the raw string or map returned by the connector.

        For each metric value you parse, create a new metric using maas.NewMetric().

        Return a slice of maas.Metric objects.

Example (collectors/cpu.go continued):
Generated go

      
// CPUScraper holds state for scraping CPU metrics.
type CPUScraper struct {
	baseoid string // Example for SNMP
}

// NewCPUScraper creates a new scraper instance.
func NewCPUScraper(app *kingpin.Application) *CPUScraper {
	return &CPUScraper{
		baseoid: ".1.3.6.1.4.1.5003.15.2.4.1.2.1",
	}
}

// Scrape performs the actual data collection.
func (s *CPUScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// 1. Use the connector to get raw data.
	// This example is from the AudioCodes exporter using SNMP.
	response, err := c.Execute(snmp.Query{
		SNMPType: "walk",
		OIDs:     []string{s.baseoid},
	})
	if err != nil {
		return nil, err
	}
	results := response.(map[string]string)
	
	metrics := make([]maas.Metric, 0, len(results))

	// 2. Parse the results.
	// ... (parsing logic as seen in the examples) ...
	
	// 3. Create metrics using maas.NewMetric.
	// The metric name "usage_percent" matches what was registered in NewCPUCollector.
	// The labels {"id"} also match the registration.
	for i, c := range cpus {
		metrics = append(metrics, maas.NewMetric(
		    "usage_percent",          // Metric name (suffix)
		    prometheus.GaugeValue,    // Metric type (GaugeValue or CounterValue)
		    c.usagePercent,           // The float64 value
		    []string{i},              // The label values, in the same order as defined in WithDescription
		))
	}

	return metrics, nil
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END
4. Writing Tests

Testing is mandatory. Our framework makes it easy to write hermetic tests by mocking the connector.

    Create a testdata directory: Inside /collectors/testdata, create mock output files. The name of the file should correspond to the command being run or the SNMP OID being queried. See existing exporters for examples.

    Use testify/suite: Structure your tests within a suite for better organization.

    Use a File Connector: In your SetupTest function, create a mock connector using command.NewFile("./testdata/...") or snmp.NewFile("./testdata/..."). This connector reads from your mock files instead of a real system.

    Use testutil.CollectAndCompare: This is the primary function for verifying metrics. It scrapes your exporter, compares the output to an expected string, and fails the test if they don't match.

Example (collectors/cpu_test.go):
Generated go

      
package collectors

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sabio-engineering-product/monitoring-connectors/snmp" // or command
	maas "github.com/sabio-engineering-product/monitoring-maas"
	"github.com/stretchr/testify/suite"
)

type CPUTestSuite struct {
	suite.Suite
	*snmp.File // or *command.File
	*maas.Exporter
}

func (s *CPUTestSuite) SetupTest() {
    // 1. Initialize the mock file connector.
	s.File = snmp.NewFile("./testdata/7.4/") 
	
	// 2. Create an exporter instance using the mock connector.
	e, err := NewAudioCodesSBCExporter(s.File, maas.WithArgs([]string{"--web.listen-port=9100"}), maas.WithLabels(&maas.MockLabels{}))
	s.NoError(err)
	s.Exporter = e
}

func (s *CPUTestSuite) TestUsagePercent() {
	s.Exporter.Start()

    // 3. Define the expected Prometheus output.
	err := testutil.CollectAndCompare(s.Exporter, strings.NewReader(`
		# HELP audiocodes_cpu_usage_percent CPU utilization
		# TYPE audiocodes_cpu_usage_percent gauge
		audiocodes_cpu_usage_percent{id="0"} 1
		audiocodes_cpu_usage_percent{id="1"} 0
	`), "audiocodes_cpu_usage_percent") // Metric name to filter for

	s.NoError(err)
}

func TestCPUTestSuite(t *testing.T) {
	suite.Run(t, new(CPUTestSuite))
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END
5. Main Application and Dockerfile

    cmd/your_exporter/main.go: This file is the final entrypoint. It's responsible for creating the real connector (command.NewSSH() or snmp.NewSNMP()) and passing it to your exporter constructor.

    Dockerfile: Our Dockerfiles are simple. They use a minimal base image (busybox-glibc) and copy the compiled Go binary into it. Ensure your CMD points to your binary.

6. Alerts and Dashboards

As part of the deliverable for a new exporter, you must also provide basic alerting rules and a Grafana dashboard.

    Alerts (/alerts/*.yaml): Define Prometheus alerting rules for your new metrics. Focus on critical conditions like high usage, error rates, or service availability. Use existing alert files as a template.

    Dashboards (/dashboards/*.json): Create at least one overview dashboard for Grafana. Export the JSON model from a working Grafana instance and place it here. Ensure you use template variables (e.g., $fqdn, $location) so the dashboard can be used for any instance of your exporter.

7. Workflow Summary

    Bootstrap: Copy an existing exporter project to a new directory. Rename files and package declarations accordingly.

    Define Collectors: In /collectors, create a collector_...go file for each metric group.

    Implement New...Collector: In each collector, define the constructor, register metrics with maas.WithDescription, and set a schedule with maas.WithSchedule.

    Implement Scrape: Add the logic to fetch, parse, and convert data into maas.Metric objects.

    Write Tests: Create mock data in /collectors/testdata. Write a _test.go file for each collector, using the File connector and testutil.CollectAndCompare.

    Wire It Up: Add your new collectors to the maas.WithScheduledScrapers call in collectors/exporter.go.

    Update main.go: Ensure cmd/your_exporter/main.go calls your NewYourExporter function.

    Add Alerts & Dashboards: Create corresponding monitoring assets.

    Documentation: Update the README.md and CODEOWNERS file.

Of course. Here is a basic repository template, structured as a set of files, that you can provide to your developers. It includes a simple, commented collector, a corresponding test, and all the necessary boilerplate to get started quickly.
New Exporter Template Repository

This repository serves as a template for creating new Prometheus exporters. It follows the standard structure and practices used at Sabio.
File Structure and Contents

Below are the contents for each file in the template.

monitoring-new-exporter-template/README.md
Generated markdown

      
# New Exporter

This is a Prometheus exporter for [System Name].

## Overview

This exporter collects metrics such as:
- System uptime and version information.
- Other key performance indicators.

It connects to the target system via [SSH/SNMP/API] and exposes metrics on `/metrics` for Prometheus to scrape.

## Development

To get started, follow the [Developer Guide for Creating a Sabio Prometheus Exporter](link_to_your_guide.html).

### Building

```bash
go build -o new_exporter ./cmd/new_exporter

    

IGNORE_WHEN_COPYING_START
Use code with caution. Markdown
IGNORE_WHEN_COPYING_END
Testing
Generated bash

      
go test -v ./...

    

IGNORE_WHEN_COPYING_START
Use code with caution. Bash
IGNORE_WHEN_COPYING_END
Generated code

      
---
**`monitoring-new-exporter-template/CODEOWNERS`**

    

IGNORE_WHEN_COPYING_START
Use code with caution.
IGNORE_WHEN_COPYING_END

/.github/ @sabio-engineering-product/dev-ops
/.conform.yaml @sabio-engineering-product/dev-ops
/.golangci.yml @sabio-engineering-product/dev-ops
/CODEOWNERS @sabio-engineering-product/dev-ops
Generated code

      
---
**`monitoring-new-exporter-template/.golangci.yml`**
```yaml
# Standard company Go linter configuration.
# (Copy contents from an existing project)

    

IGNORE_WHEN_COPYING_START
Use code with caution.
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/go.mod
Generated mod

      
module github.com/sabio-engineering-product/monitoring-new-exporter-template

go 1.19

require (
	github.com/getsentry/sentry-go v0.13.0
	github.com/prometheus/client_golang v1.13.0
	github.com/sabio-engineering-product/monitoring-connectors v0.0.0-20230217141956-fb537bda1706
	github.com/sabio-engineering-product/monitoring-maas v0.0.0-20230217132311-f35be6c949cb
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cast v1.5.0
	github.com/stretchr/testify v1.8.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)

// Add 'replace' directives here for local development if needed
// replace github.com/sabio-engineering-product/monitoring-maas => ../monitoring-maas

    

IGNORE_WHEN_COPYING_START
Use code with caution. Mod
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/go.sum
Generated code

      
# A minimal go.sum for the template. `go mod tidy` will populate this fully.
github.com/getsentry/sentry-go v0.13.0 h1:20dgTiUSfxRB/EhMPtxcL9ZEbM1ZdR+W/7f7NWD+xWo=
github.com/getsentry/sentry-go v0.13.0/go.mod h1:EOsfu5ZdvKPfeHYV6pTVQnsjfp30+XA7//UooKNumH0=
github.com/prometheus/client_golang v1.13.0 h1:b71QUfeo5M8gq2+evJdTPfZhYMAU0uKPkyPJ7TPsloU=
github.com/prometheus/client_golang v1.13.0/go.mod h1:vTeo+zgvILHsnnj/39Ou/1fPN5nJFOEMgftOUOmlvYQ=
github.com/sabio-engineering-product/monitoring-connectors v0.0.0-20230217141956-fb537bda1706 h1:cFm4NiBhh0Qx5JxyhTo4ji6/TFX6wNEPZRT+JxNuL3s=
github.com/sabio-engineering-product/monitoring-maas v0.0.0-20230217132311-f35be6c949cb h1:0k5A046OukGCkpxvOfAoEbQ6evjbDNbYZmdt3RQvfLE=
github.com/sirupsen/logrus v1.9.0 h1:trlNQbNUG3OdDrDil03MCb1H2o9nJ1x4/5LYw7byDE0=
github.com/sirupsen/logrus v1.9.0/go.mod h1:naHLuLoDiP4jHNo9R0sCBMtWGeIprob74mVsIT4qYEQ=
gopkg.in/alecthomas/kingpin.v2 v2.2.6 h1:jMFz6MfLP0/4fUyZle81rXUoxOBFi19VUFKVDOQfozc=
gopkg.in/alecthomas/kingpin.v2 v2.2.6/go.mod h1:FMv+mEhP44yOT+4EoQTLFTRgOQ1FBLkstjWtayDeSgw=

    

IGNORE_WHEN_COPYING_START
Use code with caution.
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/Dockerfile
Generated dockerfile

      
FROM registry-maas.maas.services.sabio.co.uk/docker/busybox-glibc:1.0.0
COPY new_exporter /
CMD ["/new_exporter"]

    

IGNORE_WHEN_COPYING_START
Use code with caution. Dockerfile
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/cmd/new_exporter/main.go
Generated go

      
package main

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sabio-engineering-product/monitoring-connectors/command"
	"github.com/sabio-engineering-product/monitoring-new-exporter-template/collectors"
	log "github.com/sirupsen/logrus"
)

const sentryTimeout = 5 * time.Second

func main() {
	// For a real exporter, choose the appropriate connector, e.g., command.NewSSH() or snmp.NewSNMP()
	e, err := collectors.NewSystemExporter(command.NewSSH())

	if err != nil {
		log.Fatal(err)
	}

	e.Start()
	e.Serve()

	sentry.Flush(sentryTimeout)
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/collectors/exporter.go
Generated go

      
package collectors

import (
	maas "github.com/sabio-engineering-product/monitoring-maas"
	"gopkg.in/alecthomas/kingpin.v2"
)

// NewSystemExporter is the constructor for the entire exporter. It registers all individual collectors.
func NewSystemExporter(c maas.Connector, options ...func(*maas.Exporter)) (*maas.Exporter, error) {
	// Initialize kingpin, which handles command-line flags.
	app := kingpin.New("newexporter", "A Prometheus exporter for a new system.").DefaultEnvars()

	// Register all collectors that this exporter will use.
	options = append(options, maas.WithScheduledScrapers(
		NewSystemCollector(app),
		// Add other collectors here, e.g., NewCPUCollector(app),
	))

	// Create the final exporter instance.
	e, err := maas.NewExporter(app, c, options...)

	return e, err
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/collectors/system.go
Generated go

      
package collectors

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	maas "github.com/sabio-engineering-product/monitoring-maas"
	"github.com/spf13/cast"
	"gopkg.in/alecthomas/kingpin.v2"
)

// NewSystemCollector defines the "system" metric collector.
// It registers the metrics this collector provides and sets its schedule.
func NewSystemCollector(app *kingpin.Application) *maas.ScheduledScraper {
	const defaultFrequency = 15 * time.Second
	const defaultTimeout = 5 * time.Second

	return maas.NewScheduledScraper("system", NewSystemScraper(app),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(defaultFrequency),
			maas.WithTimeout(defaultTimeout),
		)),
		// IMPORTANT: Every metric must be registered here.
		// This creates the HELP string and a --no-system.metric-name flag.
		maas.WithDescription(app, "info", "System information", []string{"version", "serial"}),
		maas.WithDescription(app, "uptime_seconds", "System uptime in seconds", nil),
	)
}

// SystemScraper holds state for this scraper, like compiled regexes.
type SystemScraper struct {
	parser *regexp.Regexp
}

// NewSystemScraper is the constructor for the scraper struct.
func NewSystemScraper(app *kingpin.Application) *SystemScraper {
	return &SystemScraper{
		// Use regex to parse structured text data.
		parser: regexp.MustCompile(`(?m)^(\w+):(.+)$`),
	}
}

// Scrape is the core method where data is collected and converted to metrics.
func (s *SystemScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// 1. Execute a command on the target using the connector.
	out, err := c.Execute("get_system_stats")
	if err != nil {
		return nil, err
	}

	// 2. Parse the output.
	matches := s.parser.FindAllStringSubmatch(out.(string), -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("failed to parse system stats from output: %s", out.(string))
	}

	parsedStats := make(map[string]string)
	for _, match := range matches {
		parsedStats[match[1]] = strings.TrimSpace(match[2])
	}

	// 3. Create metrics using maas.NewMetric.
	metrics := make([]maas.Metric, 0, 2)

	// Uptime metric (no labels)
	if uptimeStr, ok := parsedStats["uptime_seconds"]; ok {
		metrics = append(metrics, maas.NewMetric(
			"uptime_seconds",
			prometheus.GaugeValue,
			cast.ToFloat64(uptimeStr),
			nil, // No labels for this metric
		))
	}

	// Info metric (with labels)
	if version, ok := parsedStats["version"]; ok {
		serial := parsedStats["serial"] // Assume serial is always present if version is
		metrics = append(metrics, maas.NewMetric(
			"info",
			prometheus.GaugeValue,
			1, // Info metrics are typically a gauge with a value of 1
			[]string{version, serial}, // Label values must match the order in WithDescription
		))
	}

	return metrics, nil
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/collectors/system_test.go
Generated go

      
package collectors

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sabio-engineering-product/monitoring-connectors/command"
	maas "github.com/sabio-engineering-product/monitoring-maas"
	"github.com/stretchr/testify/suite"
)

type SystemTestSuite struct {
	suite.Suite
	*command.File
	*maas.Exporter
}

// SetupTest initializes the test suite.
func (s *SystemTestSuite) SetupTest() {
	// Use a mock file connector that reads from local testdata.
	s.File = command.NewFile("./testdata/")
	// Create an exporter instance with our mock connector.
	e, err := NewSystemExporter(s.File, maas.WithArgs([]string{"--web.listen-port=9100"}), maas.WithLabels(&maas.MockLabels{}))
	s.NoError(err)
	s.Exporter = e
}

// TestSystemInfo verifies the 'info' metric.
func (s *SystemTestSuite) TestSystemInfo() {
	s.Exporter.Start()
	err := testutil.CollectAndCompare(s.Exporter, strings.NewReader(`
		# HELP newexporter_system_info System information
		# TYPE newexporter_system_info gauge
		newexporter_system_info{serial="ABC-123",version="1.2.3"} 1
	`), "newexporter_system_info")
	s.NoError(err)
}

// TestUptimeSeconds verifies the 'uptime_seconds' metric.
func (s *SystemTestSuite) TestUptimeSeconds() {
	s.Exporter.Start()
	err := testutil.CollectAndCompare(s.Exporter, strings.NewReader(`
		# HELP newexporter_system_uptime_seconds System uptime in seconds
		# TYPE newexporter_system_uptime_seconds gauge
		newexporter_system_uptime_seconds 98765
	`), "newexporter_system_uptime_seconds")
	s.NoError(err)
}

// TestSystemTestSuite runs the suite.
func TestSystemTestSuite(t *testing.T) {
	suite.Run(t, new(SystemTestSuite))
}

    

IGNORE_WHEN_COPYING_START
Use code with caution. Go
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/collectors/testdata/get_system_stats
Generated code

      
version: 1.2.3
serial: ABC-123
uptime_seconds: 98765

    

IGNORE_WHEN_COPYING_START
Use code with caution.
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/alerts/system.yaml
Generated yaml

      
groups:
  - name: System Health
    rules:
      - alert: SystemJustRebooted
        expr: newexporter_system_uptime_seconds < 300
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "System {{ $labels.fqdn }} has just rebooted"
          description: "The system {{ $labels.fqdn }} has an uptime of less than 5 minutes."
          dashboard: "d/new_exporter_overview?var-fqdn={{ $labels.fqdn }}"

    

IGNORE_WHEN_COPYING_START
Use code with caution. Yaml
IGNORE_WHEN_COPYING_END

monitoring-new-exporter-template/dashboards/new_exporter_overview.json
Generated json

      
{
  "__inputs": [],
  "__requires": [],
  "annotations": {
    "list": []
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "id": null,
  "links": [],
  "panels": [
    {
      "datasource": null,
      "fieldConfig": {
        "defaults": {
          "unit": "s"
        },
        "overrides": []
      },
      "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
      "id": 2,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {
          "calcs": ["lastNotNull"],
          "fields": "",
          "values": false
        },
        "textMode": "auto"
      },
      "pluginVersion": "8.3.3",
      "targets": [
        {
          "exemplar": true,
          "expr": "newexporter_system_uptime_seconds{fqdn=\"$fqdn\"}",
          "refId": "A"
        }
      ],
      "title": "Uptime",
      "type": "stat"
    }
  ],
  "schemaVersion": 34,
  "style": "dark",
  "tags": ["new-exporter"],
  "templating": {
    "list": [
      {
        "current": {},
        "datasource": null,
        "definition": "label_values(up{job=\"newexporter\"}, fqdn)",
        "hide": 0,
        "includeAll": false,
        "label": "FQDN",
        "multi": false,
        "name": "fqdn",
        "options": [],
        "query": "label_values(up{job=\"newexporter\"}, fqdn)",
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
        "sort": 0,
        "type": "query"
      }
    ]
  },
  "time": { "from": "now-1h", "to": "now" },
  "timepicker": {},
  "timezone": "",
  "title": "New Exporter Overview",
  "uid": "new_exporter_overview",
  "version": 1
}

    

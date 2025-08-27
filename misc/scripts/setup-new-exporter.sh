#!/bin/bash

# Setup script for creating a new exporter from this template
# Usage: ./scripts/setup-new-exporter.sh <new-exporter-name> <github-org>

set -e

if [ $# -ne 2 ]; then
    echo "Usage: $0 <new-exporter-name> <github-org>"
    echo "Example: $0 my-service-exporter sabio-engineering-product"
    exit 1
fi

NEW_EXPORTER_NAME="$1"
GITHUB_ORG="$2"

echo "Setting up new exporter: $NEW_EXPORTER_NAME"
echo "GitHub organization: $GITHUB_ORG"

# Function to replace template placeholders in files
replace_in_file() {
    local file="$1"
    if [ -f "$file" ]; then
        # Replace template placeholders
        sed -i "s/TEMPLATE_EXPORTER_NAME/$NEW_EXPORTER_NAME/g" "$file"
        sed -i "s/TEMPLATE_GITHUB_ORG/$GITHUB_ORG/g" "$file"
        sed -i "s/TEMPLATE_ORG/$GITHUB_ORG/g" "$file"
        sed -i "s/TEMPLATE_REPO_NAME/$NEW_EXPORTER_NAME/g" "$file"
        sed -i "s/monitoring-rss-exporter/$NEW_EXPORTER_NAME/g" "$file"
        sed -i "s/rss_exporter/${NEW_EXPORTER_NAME}_exporter/g" "$file"
        sed -i "s/RssExporter/${NEW_EXPORTER_NAME^}Exporter/g" "$file"
        sed -i "s/template_exporter/${NEW_EXPORTER_NAME}_exporter/g" "$file"
        sed -i "s/TemplateExporter/${NEW_EXPORTER_NAME^}Exporter/g" "$file"
        sed -i "s/new_exporter/${NEW_EXPORTER_NAME}_exporter/g" "$file"
        
        # Fix function naming issues (remove hyphens)
        local clean_name=$(echo "$NEW_EXPORTER_NAME" | sed 's/-//g')
        sed -i "s/NewConnect-exporterExporter/New${clean_name^}Exporter/g" "$file"
        sed -i "s/Newconnect-exporterExporter/New${clean_name^}Exporter/g" "$file"
        
        # Fix import paths to use local monitoring-maas
        sed -i "s|github.com/TEMPLATE_GITHUB_ORG/TEMPLATE_EXPORTER_NAME/monitoring-maas|github.com/$GITHUB_ORG/$NEW_EXPORTER_NAME/monitoring-maas|g" "$file"
        sed -i "s|github.com/TEMPLATE_ORG/TEMPLATE_REPO_NAME/monitoring-maas|github.com/$GITHUB_ORG/$NEW_EXPORTER_NAME/monitoring-maas|g" "$file"
        
        # Fix connector imports to use local connectors instead of monitoring-connectors  
        sed -i "s|github.com/sabio-engineering-product/monitoring-connectors/command|github.com/$GITHUB_ORG/$NEW_EXPORTER_NAME/connectors|g" "$file"
        sed -i "s|command.NewFile|connectors.NewMockConnector|g" "$file"
        sed -i "s|command.File|connectors.MockConnector|g" "$file"
        
        echo "Updated: $file"
    fi
}

# Create template files for new exporter
echo "Creating template files..."

# Create new main.go template
cat > "cmd/template_exporter/main.go" << 'EOF'
package main

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/TEMPLATE_GITHUB_ORG/TEMPLATE_EXPORTER_NAME/collectors"
	"github.com/TEMPLATE_GITHUB_ORG/TEMPLATE_EXPORTER_NAME/connectors"
	log "github.com/sirupsen/logrus"
)

const sentryTimeout = 5 * time.Second

func main() {
	// TEMPLATE: Choose the appropriate connector for your data source
	// Examples:
	// - For local/SSH commands: Use command.NewSSH() (production) or connectors.NewMockConnector() (testing)
	// - For HTTP APIs: connectors.NewTemplateHTTPConnector()
	// - For SNMP: Use snmp.NewSNMP() (production)
	
	// For this template example, we'll use HTTP connector
	connector := connectors.NewTemplateHTTPConnector()

	// Create the exporter using our template constructor
	exporter, err := collectors.NewTEMPLATE_EXPORTER_NAMEExporter(connector)
	if err != nil {
		log.Fatal(err)
	}

	// Start all scheduled scrapers
	exporter.Start()
	
	// Start HTTP server for metrics endpoint
	exporter.Serve()

	// Ensure Sentry events are sent before shutdown
	sentry.Flush(sentryTimeout)
}
EOF

# Create template exporter.go
cat > "collectors/template_exporter.go" << 'EOF'
package collectors

import (
	"os"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	maas "github.com/TEMPLATE_GITHUB_ORG/TEMPLATE_EXPORTER_NAME/monitoring-maas"
	"gopkg.in/yaml.v3"
)

// Config defines exporter settings loaded from YAML configuration.
type Config struct {
	// TEMPLATE: Define your configuration structure
	Services []ServiceConfig `yaml:"services"`
}

type ServiceConfig struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
	// TEMPLATE: Add your service-specific config fields
}

// NewTEMPLATE_EXPORTER_NAMEExporter constructs a maas exporter with your collectors.
func NewTEMPLATE_EXPORTER_NAMEExporter(c maas.Connector, options ...func(*maas.Exporter)) (*maas.Exporter, error) {
	app := kingpin.New("TEMPLATE_EXPORTER_NAME_exporter", "TEMPLATE: Description of your exporter.").DefaultEnvars()
	app.Flag("config.file", "Configuration file path.").Default("config.yml").String()

	// Read configuration file
	configPath := "config.yml"
	// Look for --config.file in args to override default
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--config.file=") {
			configPath = strings.TrimPrefix(arg, "--config.file=")
		} else if arg == "--config.file" && i+1 < len(os.Args[1:]) {
			configPath = os.Args[i+2]
		}
	}

	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
		return nil, err
	}

	// Create collectors based on config
	scrapers := []*maas.ScheduledScraper{}
	for _, svc := range cfg.Services {
		if svc.Interval <= 0 {
			svc.Interval = 300 // Default 5 minutes
		}
		// TEMPLATE: Create your specific collectors here
		scrapers = append(scrapers, NewExampleCollector(app, svc))
	}

	// Create the exporter with collectors
	options = append(options, maas.WithScheduledScrapers(scrapers...))
	return maas.NewExporter(app, c, options...)
}
EOF

# Create template collector
cat > "collectors/template_collector.go" << 'EOF'
package collectors

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	maas "github.com/TEMPLATE_GITHUB_ORG/TEMPLATE_EXPORTER_NAME/monitoring-maas"
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
EOF

# Update existing template files with new names
echo "Updating existing template files..."

# Update main application
replace_in_file "cmd/new_exporter/main.go"

# Update collectors
replace_in_file "collectors/template_exporter.go"
replace_in_file "collectors/system.go"
replace_in_file "collectors/cpu.go"
replace_in_file "collectors/memory.go"
replace_in_file "collectors/system_test.go"
replace_in_file "collectors/template_example_collector.go"
replace_in_file "collectors/template_example_collector_test.go"
replace_in_file "collectors/template_test_helpers.go"

# Update go.mod
replace_in_file "go.mod"

# Rename files and directories
echo "Renaming template files..."

# Rename main application directory and file
mkdir -p "cmd/${NEW_EXPORTER_NAME}_exporter"
mv "cmd/new_exporter/main.go" "cmd/${NEW_EXPORTER_NAME}_exporter/main.go"
rmdir "cmd/new_exporter" 2>/dev/null || true

# Rename collector files
mv "collectors/template_exporter.go" "collectors/${NEW_EXPORTER_NAME}_exporter.go"
if [ -f "collectors/system.go" ]; then
    mv "collectors/system.go" "collectors/${NEW_EXPORTER_NAME}_system.go"
fi
if [ -f "collectors/cpu.go" ]; then
    mv "collectors/cpu.go" "collectors/${NEW_EXPORTER_NAME}_cpu.go"
fi
if [ -f "collectors/memory.go" ]; then
    mv "collectors/memory.go" "collectors/${NEW_EXPORTER_NAME}_memory.go"
fi

# Rename test files
if [ -f "collectors/system_test.go" ]; then
    mv "collectors/system_test.go" "collectors/${NEW_EXPORTER_NAME}_system_test.go"
fi

# Rename template_collector.go to be more specific
if [ -f "collectors/template_collector.go" ]; then
    mv "collectors/template_collector.go" "collectors/${NEW_EXPORTER_NAME}_collector.go"
fi

# Clean up any remaining template files
echo "Cleaning up template files..."
if [ -f "cmd/template_exporter/main.go" ]; then
    rm -f "cmd/template_exporter/main.go"
    rmdir "cmd/template_exporter" 2>/dev/null || true
fi

# Fix any remaining template placeholders in collector files
echo "Fixing remaining template placeholders..."
for file in collectors/*.go; do
    if [ -f "$file" ]; then
        replace_in_file "$file"
    fi
done

# Fix CPU test file specifically
if [ -f "collectors/cpu_test.go" ]; then
    replace_in_file "collectors/cpu_test.go"
fi

# Final fixes for function naming (remove hyphens from function names)
echo "Applying final function name fixes..."
clean_name=$(echo "$NEW_EXPORTER_NAME" | sed 's/-//g')
if [ -f "collectors/${NEW_EXPORTER_NAME}_exporter.go" ]; then
    sed -i "s/NewConnectexporterExporter/New${clean_name^}Exporter/g" "collectors/${NEW_EXPORTER_NAME}_exporter.go"
    sed -i "s/connect-exporter_exporter/connect-exporter/g" "collectors/${NEW_EXPORTER_NAME}_exporter.go"
    sed -i "s/TEMPLATE: Description of your exporter./Prometheus exporter for ${NEW_EXPORTER_NAME} services./g" "collectors/${NEW_EXPORTER_NAME}_exporter.go"
fi

if [ -f "cmd/${NEW_EXPORTER_NAME}_exporter/main.go" ]; then
    sed -i "s/NewConnectexporterExporter/New${clean_name^}Exporter/g" "cmd/${NEW_EXPORTER_NAME}_exporter/main.go"
fi

# Create template config
cat > "config.template.yml" << EOF
listen_address: 127.0.0.1
listen_port: 9090
log_level: info

services:
  - name: example-service
    url: https://api.example.com/health
    interval: 60
  # TEMPLATE: Add your service configurations here
EOF

# Copy template config to working config
echo "Creating working configuration file..."
cp config.template.yml config.yml

# Run go mod tidy to clean up dependencies
echo "Cleaning up Go module dependencies..."
go mod tidy 2>/dev/null || echo "Warning: go mod tidy had issues, but continuing..."

echo ""
echo "✅ Template setup complete!"
echo ""
echo "📁 Generated Files:"
echo "   - cmd/${NEW_EXPORTER_NAME}_exporter/main.go         # Main application"
echo "   - collectors/${NEW_EXPORTER_NAME}_exporter.go       # Main exporter constructor"
echo "   - collectors/${NEW_EXPORTER_NAME}_collector.go      # Example collector"
echo "   - config.yml                                         # Working configuration"
echo "   - config.template.yml                               # Configuration template"
echo ""
echo "🚀 Next Steps:"
echo "1. Implement your data collection logic in the Scrape methods"
echo "2. Customize config.yml with your service endpoints"
echo "3. Add tests following the pattern in collectors/*_test.go"
echo "4. Update README.md with your exporter details"
echo ""
echo "🔧 Build & Test:"
echo "   go build -o ${NEW_EXPORTER_NAME} ./cmd/${NEW_EXPORTER_NAME}_exporter"
echo "   ./${NEW_EXPORTER_NAME} --config.file=config.yml"
echo ""
echo "📝 Note: There may be some compilation issues to resolve:"
echo "   - HTTPQuery type definition in connectors/"
echo "   - Duplicate MockLabels in monitoring-maas/"
echo "   These are template artifacts that need implementation."
echo ""
echo "🏭 For production deployment:"
echo "   - Remove the monitoring-maas/ folder"
echo "   - Update imports to use github.com/sabio-engineering-product/monitoring-maas"
echo ""
echo "Happy coding! 🚀"
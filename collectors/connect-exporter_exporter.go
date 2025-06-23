package collectors

import (
	"os"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
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

// NewConnectExporter constructs a maas exporter with your collectors.
func NewConnectExporter(c maas.Connector, options ...func(*maas.Exporter)) (*maas.Exporter, error) {
	app := kingpin.New("connect-exporter", "Prometheus exporter for connect services.").DefaultEnvars()
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

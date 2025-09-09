package collectors

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
)

// NewAWSConnectExporter constructs a maas exporter with AWS Connect collectors
func NewAWSConnectExporter(options ...func(*maas.Exporter)) (*maas.Exporter, error) {
	app := kingpin.New("aws-connect-exporter", "Prometheus exporter for AWS Connect services").DefaultEnvars()

	// Create a configurable AWS Connect connector
	awsConnector, err := connectors.NewConfigurableAWSConnectConnector(app)
	if err != nil {
		return nil, err
	}

	// Create collectors with default configuration - they will get the real config during Connect()
	defaultConfig := connectors.AWSConnectConfig{
		Collectors: connectors.CollectorsConfig{
			AgentRealtimeStatus: connectors.AgentRealtimeStatusConfig{Enabled: true, Interval: 60},
			GenericRealtime:     connectors.GenericRealtimeConfig{Enabled: true, Interval: 30},
			GenericHistorical:   connectors.GenericHistoricalConfig{Enabled: true, Interval: 300},
		},
	}

	scrapers := []*maas.ScheduledScraper{
		NewAgentRealtimeStatusCollector(app, defaultConfig),
		NewGenericRealtimeCollector(app, defaultConfig),
		NewGenericHistoricalCollector(app, defaultConfig),
	}

	// Create the exporter with the connector and collectors
	options = append(options, maas.WithScheduledScrapers(scrapers...))
	return maas.NewExporter(app, awsConnector, options...)
}

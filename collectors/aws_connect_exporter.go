package collectors

import (
	"time"

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
			RealtimeQueue: connectors.RealtimeQueueConfig{
				Enabled:         true,
				Interval:        30,
				MetricsCacheTTL: 15 * time.Second,
				QueueCacheTTL:   5 * time.Minute,
			},
			AgentStatus:         connectors.AgentStatusConfig{Enabled: true, Interval: 60},
			AgentRealtimeStatus: connectors.AgentRealtimeStatusConfig{Enabled: true, Interval: 60},
			HistoricalQueue:     connectors.HistoricalQueueConfig{Enabled: true, Interval: 300},
		},
	}

	scrapers := []*maas.ScheduledScraper{
		NewRealtimeQueueCollector(app, defaultConfig),
		NewAgentStatusCollector(app, defaultConfig),
		NewAgentRealtimeStatusCollector(app, defaultConfig),
		NewHistoricalQueueCollector(app, defaultConfig),
	}

	// Create the exporter with the connector and collectors
	options = append(options, maas.WithScheduledScrapers(scrapers...))
	return maas.NewExporter(app, awsConnector, options...)
}

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
				Enabled:              false, // Replaced by generic_realtime
				Interval:             30,
				MetricsCacheTTL:      15 * time.Second,
				QueueCacheTTL:        5 * time.Minute,
				QueueRefreshInterval: 5 * time.Minute,
				QueuesIncludeRegex:   "",
			},
			AgentStatus:         connectors.AgentStatusConfig{Enabled: false, Interval: 60}, // Removed - not operationally useful
			AgentRealtimeStatus: connectors.AgentRealtimeStatusConfig{Enabled: true, Interval: 60},
			HistoricalQueue:     connectors.HistoricalQueueConfig{Enabled: false, Interval: 300}, // Replaced by generic_historical
			HistoricalCalls:     connectors.HistoricalCallsConfig{Enabled: false, Interval: 300}, // Replaced by generic_historical
			HistoricalAHT:       connectors.HistoricalAHTConfig{Enabled: false, Interval: 300},   // Replaced by generic_historical
			HistoricalACW:       connectors.HistoricalACWConfig{Enabled: false, Interval: 300},   // Replaced by generic_historical
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

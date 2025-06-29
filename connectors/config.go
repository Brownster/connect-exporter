// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file defines the configuration structures for the AWS Connect exporter.
// These structs are used to parse the YAML configuration file and are central
// to the exporter's operation. By externalizing the configuration, this file
// allows for a clear separation of configuration from the core connector logic,
// making the codebase more modular and easier to maintain.
package connectors

import "time"

// Queue represents queue metadata. This struct is used to hold essential
// information about an AWS Connect queue, such as its ID, name, and type.
// It is populated by the ListQueues API call and used by various collectors.
type Queue struct {
	ID   string
	Name string
	Type string
}

// AWSConnectConfig is the top-level configuration structure for the exporter.
// It encapsulates all AWS-specific settings and collector configurations.
type AWSConnectConfig struct {
	AWS        AWSConfig        `yaml:"aws"`
	Collectors CollectorsConfig `yaml:"collectors"`
}

// AWSConfig contains AWS-specific settings required to connect to the
// correct AWS Connect instance.
type AWSConfig struct {
	Region      string `yaml:"region"`
	InstanceARN string `yaml:"instance_arn"`
}

// CollectorsConfig groups all collector-specific configurations. This allows
// for easy enabling, disabling, and tuning of individual metric collectors.
type CollectorsConfig struct {
	RealtimeQueue       RealtimeQueueConfig       `yaml:"realtime_queue"`
	AgentStatus         AgentStatusConfig         `yaml:"agent_status"`
	AgentRealtimeStatus AgentRealtimeStatusConfig `yaml:"agent_realtime_status"`
	HistoricalQueue     HistoricalQueueConfig     `yaml:"historical_queue"`
	HistoricalCalls     HistoricalCallsConfig     `yaml:"historical_calls"`
	HistoricalAHT       HistoricalAHTConfig       `yaml:"historical_aht"`
	HistoricalACW       HistoricalACWConfig       `yaml:"historical_acw"`
	GenericRealtime     GenericRealtimeConfig     `yaml:"generic_realtime"`
	GenericHistorical   GenericHistoricalConfig   `yaml:"generic_historical"`
}

// RealtimeQueueConfig configures the realtime_queue collector, which gathers
// live metrics about queue performance.
type RealtimeQueueConfig struct {
	Enabled              bool          `yaml:"enabled"`
	Interval             int           `yaml:"interval"`
	MetricsCacheTTL      time.Duration `yaml:"metrics_cache_ttl"`
	QueueCacheTTL        time.Duration `yaml:"queue_cache_ttl"`
	QueueRefreshInterval time.Duration `yaml:"queue_refresh_interval"`
	QueuesIncludeRegex   string        `yaml:"queues_include_regex"`
}

// AgentStatusConfig configures the agent_status collector, which provides
// information about the different agent statuses available in the instance.
type AgentStatusConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// AgentRealtimeStatusConfig configures the agent_realtime_status collector,
// which provides a snapshot of the current status of each agent.
type AgentRealtimeStatusConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalQueueConfig configures the historical_queue collector, which
// retrieves historical metrics for queues.
type HistoricalQueueConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalCallsConfig configures the historical_calls collector, which
// gathers metrics about call volumes and handling.
type HistoricalCallsConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalAHTConfig configures the historical_aht collector for tracking
// Average Handle Time.
type HistoricalAHTConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalACWConfig configures the historical_acw collector for tracking
// Average After Contact Work Time.
type HistoricalACWConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// FilterConfig defines a key-value filter for metric queries, allowing for
// more specific data retrieval from the AWS Connect API.
type FilterConfig struct {
	Key    string   `yaml:"key"`
	Values []string `yaml:"values"`
}

// MetricDefinition maps a specific AWS Connect metric to a Prometheus metric,
// including any additional labels that should be applied.
type MetricDefinition struct {
	AWSName        string            `yaml:"aws_name"`
	PrometheusName string            `yaml:"prometheus_name"`
	Labels         map[string]string `yaml:"labels"`
}

// GenericMetricConfig defines a flexible, user-configurable metric query.
// This allows for the collection of a wide range of metrics without code changes.
type GenericMetricConfig struct {
	Name      string             `yaml:"name"`
	Metrics   []MetricDefinition `yaml:"metrics"`
	Groupings []string           `yaml:"groupings"`
	Filters   []FilterConfig     `yaml:"filters"`
}

// GenericRealtimeConfig configures the generic_realtime collector, which uses
// a list of GenericMetricConfig to fetch live data.
type GenericRealtimeConfig struct {
	Enabled    bool                  `yaml:"enabled"`
	Interval   int                   `yaml:"interval"`
	Collectors []GenericMetricConfig `yaml:"collectors"`
}

// HistoricalMetricDefinition extends MetricDefinition to include metric-specific
// filters for historical data queries.
type HistoricalMetricDefinition struct {
	AWSName        string            `yaml:"aws_name"`
	PrometheusName string            `yaml:"prometheus_name"`
	Labels         map[string]string `yaml:"labels"`
	MetricFilters  []MetricFilter    `yaml:"metric_filters"`
}

// MetricFilter defines AWS Connect metric filters, such as filtering by call
// direction (INBOUND/OUTBOUND).
type MetricFilter struct {
	Key    string   `yaml:"key"`
	Values []string `yaml:"values"`
}

// GenericHistoricalMetricConfig defines a user-configurable historical metric query.
type GenericHistoricalMetricConfig struct {
	Name      string                       `yaml:"name"`
	Metrics   []HistoricalMetricDefinition `yaml:"metrics"`
	Groupings []string                     `yaml:"groupings"`
	Filters   []FilterConfig               `yaml:"filters"`
}

// GenericHistoricalConfig configures the generic_historical collector, which
// fetches historical data based on a list of GenericHistoricalMetricConfig.
type GenericHistoricalConfig struct {
	Enabled    bool                            `yaml:"enabled"`
	Interval   int                             `yaml:"interval"`
	Collectors []GenericHistoricalMetricConfig `yaml:"collectors"`
}

package connectors

import "time"

// Queue represents queue metadata
// (moved from aws_connect_connector.go for clarity)
// NOTE: not configuration-related but kept with config types for centralized definitions

type Queue struct {
	ID   string
	Name string
	Type string
}

// AWSConnectConfig defines the configuration structure for AWS Connect exporter
// extracted from aws_connect_connector.go

type AWSConnectConfig struct {
	AWS        AWSConfig        `yaml:"aws"`
	Collectors CollectorsConfig `yaml:"collectors"`
}

// AWSConfig contains AWS-specific configuration

type AWSConfig struct {
	Region      string `yaml:"region"`
	InstanceARN string `yaml:"instance_arn"`
}

// CollectorsConfig contains collector-specific configuration

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

// RealtimeQueueConfig configures the realtime queue collector

type RealtimeQueueConfig struct {
	Enabled              bool          `yaml:"enabled"`
	Interval             int           `yaml:"interval"`
	MetricsCacheTTL      time.Duration `yaml:"metrics_cache_ttl"`
	QueueCacheTTL        time.Duration `yaml:"queue_cache_ttl"`
	QueueRefreshInterval time.Duration `yaml:"queue_refresh_interval"`
	QueuesIncludeRegex   string        `yaml:"queues_include_regex"`
}

// AgentStatusConfig configures the agent status collector

type AgentStatusConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// AgentRealtimeStatusConfig configures the agent realtime status collector

type AgentRealtimeStatusConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalQueueConfig configures the historical queue collector

type HistoricalQueueConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalCallsConfig configures the historical calls collector

type HistoricalCallsConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalAHTConfig configures the historical average handle time collector

type HistoricalAHTConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// HistoricalACWConfig configures the historical after contact work time collector

type HistoricalACWConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

// FilterConfig defines a key/value filter for metric queries

type FilterConfig struct {
	Key    string   `yaml:"key"`
	Values []string `yaml:"values"`
}

// MetricDefinition maps an AWS metric to a Prometheus metric with optional labels

type MetricDefinition struct {
	AWSName        string            `yaml:"aws_name"`
	PrometheusName string            `yaml:"prometheus_name"`
	Labels         map[string]string `yaml:"labels"`
}

// GenericMetricConfig defines a single generic metric query

type GenericMetricConfig struct {
	Name      string             `yaml:"name"`
	Metrics   []MetricDefinition `yaml:"metrics"`
	Groupings []string           `yaml:"groupings"`
	Filters   []FilterConfig     `yaml:"filters"`
}

// GenericRealtimeConfig configures the generic realtime collector

type GenericRealtimeConfig struct {
	Enabled    bool                  `yaml:"enabled"`
	Interval   int                   `yaml:"interval"`
	Collectors []GenericMetricConfig `yaml:"collectors"`
}

// HistoricalMetricDefinition maps an AWS historical metric to a Prometheus metric with optional labels and filters

type HistoricalMetricDefinition struct {
	AWSName        string            `yaml:"aws_name"`
	PrometheusName string            `yaml:"prometheus_name"`
	Labels         map[string]string `yaml:"labels"`
	MetricFilters  []MetricFilter    `yaml:"metric_filters"`
}

// MetricFilter defines AWS Connect metric filters (e.g., for direction filtering)

type MetricFilter struct {
	Key    string   `yaml:"key"`
	Values []string `yaml:"values"`
}

// GenericHistoricalMetricConfig defines a single generic historical metric query

type GenericHistoricalMetricConfig struct {
	Name      string                       `yaml:"name"`
	Metrics   []HistoricalMetricDefinition `yaml:"metrics"`
	Groupings []string                     `yaml:"groupings"`
	Filters   []FilterConfig               `yaml:"filters"`
}

// GenericHistoricalConfig configures the generic historical collector

type GenericHistoricalConfig struct {
	Enabled    bool                            `yaml:"enabled"`
	Interval   int                             `yaml:"interval"`
	Collectors []GenericHistoricalMetricConfig `yaml:"collectors"`
}

package connectors

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// AWSConnectConnector implements the maas.Connector interface for AWS Connect
type AWSConnectConnector struct {
	client      *connect.Client
	region      string
	instanceARN string
	// Development mode uses mock API
	developmentMode bool
	mockEndpoint    string
	// Configuration flags
	regionFlag      *string
	instanceARNFlag *string
	configFileFlag  *string
	// Indicates if the connector has been configured
	configured bool
	// Store the full configuration for collectors to access
	config AWSConnectConfig
	// Queue manager for shared queue discovery
	queueManager *QueueManager
	// Agent manager for shared agent discovery
	agentManager *AgentManager
	// Routing profile -> queues mapping with caching
	rpQueueManager *RoutingProfileQueueManager
}

// Queue represents queue metadata
type Queue struct {
	ID   string
	Name string
	Type string
}

// AWSConnectConfig defines the configuration structure for AWS Connect exporter
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

// NewConfigurableAWSConnectConnector creates a new AWS Connect connector with kingpin flags
func NewConfigurableAWSConnectConnector(app *kingpin.Application) (*AWSConnectConnector, error) {
	connector := &AWSConnectConnector{
		configured: false,
	}

	// Define CLI flags for AWS Connect
	connector.instanceARNFlag = app.Flag("instance-arn", "AWS Connect Instance ARN").String()
	connector.regionFlag = app.Flag("region", "AWS region for AWS Connect").Default("us-east-1").String()
	connector.configFileFlag = app.Flag("config.file", "Configuration file path").Default("config.yml").String()

	return connector, nil
}

// NewAWSConnectConnector creates a new AWS Connect connector
func NewAWSConnectConnector(region, instanceARN string) (*AWSConnectConnector, error) {
	connector := &AWSConnectConnector{
		region:      region,
		instanceARN: instanceARN,
	}

	// Check if we're in development mode (use mock API)
	if isDevelopmentMode() {
		connector.developmentMode = true
		connector.mockEndpoint = getMockEndpoint()
		logrus.WithFields(logrus.Fields{
			"mock_endpoint": connector.mockEndpoint,
			"instance_arn":  instanceARN,
		}).Info("Running in development mode with mock AWS Connect API")
	}

	// Initialize AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Connect client
	if connector.developmentMode {
		// Configure client to use mock endpoint
		connector.client = connect.NewFromConfig(cfg, func(o *connect.Options) {
			o.BaseEndpoint = aws.String(connector.mockEndpoint)
		})
	} else {
		connector.client = connect.NewFromConfig(cfg)
	}

	return connector, nil
}

// Execute implements the maas.Connector interface
func (c *AWSConnectConnector) Execute(command interface{}) (interface{}, error) {
	switch cmd := command.(type) {
	case string:
		// Handle string commands for backward compatibility
		switch cmd {
		case "list_queues":
			return c.ListQueues(context.Background())
		default:
			return nil, fmt.Errorf("unsupported command: %s", cmd)
		}
	case *connect.GetCurrentMetricDataInput:
		return c.GetCurrentMetricData(context.Background(), cmd)
	case *connect.GetMetricDataV2Input:
		return c.GetMetricDataV2(context.Background(), cmd)
	case *connect.ListAgentStatusesInput:
		return c.ListAgentStatuses(context.Background(), cmd)
	case *connect.GetCurrentUserDataInput:
		return c.GetCurrentUserData(context.Background(), cmd)
	case *connect.ListUsersInput:
		return c.ListUsers(context.Background(), cmd)
	default:
		return nil, fmt.Errorf("unsupported command type: %T", command)
	}
}

// Connect implements the maas.Connector interface - validates AWS connection
func (c *AWSConnectConnector) Connect() error {
	// Configure the connector if not already done
	if !c.configured {
		err := c.configure()
		if err != nil {
			return err
		}
	}

	// In development mode, skip connection validation since mock API might not be running
	if c.developmentMode {
		logrus.Info("Development mode: skipping connection validation")
	} else {
		// Test connection by trying to list queues with a minimal request
		ctx := context.Background()
		instanceID := c.GetInstanceID()
		_, err := c.client.ListQueues(ctx, &connect.ListQueuesInput{
			InstanceId: &instanceID,
			MaxResults: aws.Int32(1), // Just test with 1 item
		})
		if err != nil {
			return err
		}
	}

	if c.queueManager == nil {
		qm, err := NewQueueManager(c, c.config.Collectors.RealtimeQueue.QueueRefreshInterval, c.config.Collectors.RealtimeQueue.QueuesIncludeRegex)
		if err != nil {
			return err
		}
		c.queueManager = qm
		qm.Start()
	}

	if c.agentManager == nil {
		am := NewAgentManager(c, c.config.Collectors.RealtimeQueue.QueueRefreshInterval)
		c.agentManager = am
		am.Start()
	}

	if c.rpQueueManager == nil {
		c.rpQueueManager = NewRoutingProfileQueueManager(c, c.config.Collectors.RealtimeQueue.QueueCacheTTL)
	}
	return nil
}

// Flags implements the maas.Connector interface
func (c *AWSConnectConnector) Flags(app *kingpin.Application) {
	// AWS Connect specific flags could be added here if needed
	// For now, we rely on config file and CLI flags in the exporter
}

// ListQueues retrieves all queues for the instance
func (c *AWSConnectConnector) ListQueues(ctx context.Context) ([]Queue, error) {
	var queues []Queue
	var nextToken *string

	for {
		instanceID := c.GetInstanceID()
		output, err := c.client.ListQueues(ctx, &connect.ListQueuesInput{
			InstanceId: &instanceID,
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing queues: %w", err)
		}

		for _, q := range output.QueueSummaryList {
			queues = append(queues, Queue{
				ID:   aws.ToString(q.Id),
				Name: aws.ToString(q.Name),
				Type: string(q.QueueType),
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	logrus.WithFields(logrus.Fields{
		"queue_count":  len(queues),
		"instance_arn": c.instanceARN,
		"region":       c.region,
	}).Debug("Successfully listed queues")

	return queues, nil
}

// GetCurrentMetricData retrieves current metrics for specified filters
func (c *AWSConnectConnector) GetCurrentMetricData(ctx context.Context, input *connect.GetCurrentMetricDataInput) (*connect.GetCurrentMetricDataOutput, error) {
	output, err := c.client.GetCurrentMetricData(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting current metric data: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"metric_count": len(output.MetricResults),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully retrieved current metric data")

	return output, nil
}

// GetMetricDataV2 retrieves historical metrics using the V2 API
func (c *AWSConnectConnector) GetMetricDataV2(ctx context.Context, input *connect.GetMetricDataV2Input) (*connect.GetMetricDataV2Output, error) {
	output, err := c.client.GetMetricDataV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting metric data v2: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"metric_count": len(output.MetricResults),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully retrieved metric data v2")

	return output, nil
}

// GetCurrentUserData retrieves current user data including agent status
func (c *AWSConnectConnector) GetCurrentUserData(ctx context.Context, input *connect.GetCurrentUserDataInput) (*connect.GetCurrentUserDataOutput, error) {
	output, err := c.client.GetCurrentUserData(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting current user data: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_count":   len(output.UserDataList),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully retrieved current user data")

	return output, nil
}

// ListAgentStatuses retrieves agent status information
func (c *AWSConnectConnector) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	output, err := c.client.ListAgentStatuses(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error listing agent statuses: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"status_count": len(output.AgentStatusSummaryList),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully listed agent statuses")

	return output, nil
}

// ListUsers retrieves user information
func (c *AWSConnectConnector) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	output, err := c.client.ListUsers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_count":   len(output.UserSummaryList),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully listed users")

	return output, nil
}

// ListRoutingProfileQueues retrieves queue IDs associated with a routing profile
func (c *AWSConnectConnector) ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error) {
	output, err := c.client.ListRoutingProfileQueues(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error listing routing profile queues: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"queue_count":  len(output.RoutingProfileQueueConfigSummaryList),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully listed routing profile queues")

	return output, nil
}

// Helper functions for development mode detection

// isDevelopmentMode checks if we should use the mock API
func isDevelopmentMode() bool {
	// Check for development environment variables
	env := strings.ToLower(os.Getenv("ENV"))
	if env == "development" || env == "dev" {
		return true
	}

	nodeEnv := strings.ToLower(os.Getenv("NODE_ENV"))
	if nodeEnv == "development" || nodeEnv == "dev" {
		return true
	}

	// Check if mock endpoint is explicitly configured
	mockEndpoint := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	return mockEndpoint != ""
}

// getMockEndpoint returns the mock API endpoint
func getMockEndpoint() string {
	endpoint := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:3000"
	}
	return endpoint
}

// configure handles the configuration parsing after flags have been parsed
func (c *AWSConnectConnector) configure() error {
	// Load configuration from file
	var cfg AWSConnectConfig
	configPath := *c.configFileFlag

	if _, err := os.Stat(configPath); err == nil {
		yamlFile, err := os.ReadFile(configPath)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to read config file %s, using defaults", configPath)
		} else {
			if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
				logrus.WithError(err).Warnf("Failed to parse config file %s, using defaults", configPath)
			}
		}
	}

	// Override config with CLI flags if provided
	if *c.instanceARNFlag != "" {
		cfg.AWS.InstanceARN = *c.instanceARNFlag
	}
	if *c.regionFlag != "" {
		cfg.AWS.Region = *c.regionFlag
	}

	// Set defaults if not configured
	if cfg.AWS.Region == "" {
		cfg.AWS.Region = "us-east-1"
	}
	if cfg.Collectors.RealtimeQueue.Interval <= 0 {
		cfg.Collectors.RealtimeQueue.Interval = 60 // Default 1 minute
	}
	if cfg.Collectors.RealtimeQueue.MetricsCacheTTL <= 0 {
		cfg.Collectors.RealtimeQueue.MetricsCacheTTL = 15 * time.Second // Default 15s
	}
	if cfg.Collectors.RealtimeQueue.QueueCacheTTL <= 0 {
		cfg.Collectors.RealtimeQueue.QueueCacheTTL = 5 * time.Minute // Default 5m
	}
	if cfg.Collectors.RealtimeQueue.QueueRefreshInterval <= 0 {
		cfg.Collectors.RealtimeQueue.QueueRefreshInterval = 5 * time.Minute
	}
	if cfg.Collectors.AgentStatus.Interval <= 0 {
		cfg.Collectors.AgentStatus.Interval = 300 // Default 5 minutes
	}
	if cfg.Collectors.AgentRealtimeStatus.Interval <= 0 {
		cfg.Collectors.AgentRealtimeStatus.Interval = 60 // Default 1 minute
	}
	if cfg.Collectors.HistoricalQueue.Interval <= 0 {
		cfg.Collectors.HistoricalQueue.Interval = 300 // Default 5 minutes
	}
	if cfg.Collectors.HistoricalCalls.Interval <= 0 {
		cfg.Collectors.HistoricalCalls.Interval = 300 // Default 5 minutes
	}
	if cfg.Collectors.HistoricalAHT.Interval <= 0 {
		cfg.Collectors.HistoricalAHT.Interval = 300 // Default 5 minutes
	}
	if cfg.Collectors.HistoricalACW.Interval <= 0 {
		cfg.Collectors.HistoricalACW.Interval = 300 // Default 5 minutes
	}
	if cfg.Collectors.GenericRealtime.Interval <= 0 {
		cfg.Collectors.GenericRealtime.Interval = 30 // Default 30 seconds
	}
	if cfg.Collectors.GenericHistorical.Interval <= 0 {
		cfg.Collectors.GenericHistorical.Interval = 300 // Default 5 minutes
	}

	// Validate required configuration
	if cfg.AWS.InstanceARN == "" {
		return fmt.Errorf("AWS Connect Instance ARN is required (use --instance-arn flag or config file)")
	}

	// Set connector values
	c.region = cfg.AWS.Region
	c.instanceARN = cfg.AWS.InstanceARN
	// Store the full configuration for collectors to access
	c.config = cfg

	logrus.WithFields(logrus.Fields{
		"instance_arn": cfg.AWS.InstanceARN,
		"region":       cfg.AWS.Region,
		"config_file":  configPath,
	}).Info("Starting AWS Connect exporter")

	// Check if we're in development mode (use mock API)
	if isDevelopmentMode() {
		c.developmentMode = true
		c.mockEndpoint = getMockEndpoint()
		logrus.WithFields(logrus.Fields{
			"mock_endpoint": c.mockEndpoint,
			"instance_arn":  c.instanceARN,
		}).Info("Running in development mode with mock AWS Connect API")
	}

	// Initialize AWS config
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(c.region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Connect client
	if c.developmentMode {
		// Configure client to use mock endpoint with HTTP (not HTTPS)
		c.client = connect.NewFromConfig(awsCfg, func(o *connect.Options) {
			o.BaseEndpoint = aws.String(c.mockEndpoint)
			// Force HTTP for development mode to work with mock API
			o.EndpointOptions.UseFIPSEndpoint = aws.FIPSEndpointStateDisabled
			o.EndpointOptions.UseDualStackEndpoint = aws.DualStackEndpointStateDisabled
		})
	} else {
		c.client = connect.NewFromConfig(awsCfg)
	}

	// Log which collectors are enabled based on configuration
	if cfg.Collectors.RealtimeQueue.Enabled {
		logrus.WithField("interval", cfg.Collectors.RealtimeQueue.Interval).Info("Enabled realtime queue collector")
	}
	if cfg.Collectors.AgentStatus.Enabled {
		logrus.WithField("interval", cfg.Collectors.AgentStatus.Interval).Info("Enabled agent status collector")
	}
	if cfg.Collectors.AgentRealtimeStatus.Enabled {
		logrus.WithField("interval", cfg.Collectors.AgentRealtimeStatus.Interval).Info("Enabled agent realtime status collector")
	}
	if cfg.Collectors.HistoricalQueue.Enabled {
		logrus.WithField("interval", cfg.Collectors.HistoricalQueue.Interval).Info("Enabled historical queue collector")
	}
	if cfg.Collectors.HistoricalCalls.Enabled {
		logrus.WithField("interval", cfg.Collectors.HistoricalCalls.Interval).Info("Enabled historical calls collector")
	}
	if cfg.Collectors.HistoricalAHT.Enabled {
		logrus.WithField("interval", cfg.Collectors.HistoricalAHT.Interval).Info("Enabled historical AHT collector")
	}
	if cfg.Collectors.HistoricalACW.Enabled {
		logrus.WithField("interval", cfg.Collectors.HistoricalACW.Interval).Info("Enabled historical ACW collector")
	}
	if cfg.Collectors.GenericRealtime.Enabled {
		logrus.WithField("interval", cfg.Collectors.GenericRealtime.Interval).Info("Enabled generic realtime collector")
	}
	if cfg.Collectors.GenericHistorical.Enabled {
		logrus.WithField("interval", cfg.Collectors.GenericHistorical.Interval).Info("Enabled generic historical collector")
	}

	c.configured = true
	return nil
}

// GetConfig returns the full configuration for collectors to access
func (c *AWSConnectConnector) GetConfig() AWSConnectConfig {
	return c.config
}

// GetQueueManager exposes the shared QueueManager
func (c *AWSConnectConnector) GetQueueManager() *QueueManager {
	return c.queueManager
}

// GetAgentManager exposes the shared AgentManager
func (c *AWSConnectConnector) GetAgentManager() *AgentManager {
	return c.agentManager
}

// GetRoutingProfileQueueManager exposes the routing profile queue manager
func (c *AWSConnectConnector) GetRoutingProfileQueueManager() *RoutingProfileQueueManager {
	return c.rpQueueManager
}

// GetInstanceID extracts the instance ID from the ARN
func (c *AWSConnectConnector) GetInstanceID() string {
	// Extract instance ID from ARN
	// ARN format: arn:aws:connect:region:account:instance/instance-id
	parts := strings.Split(c.instanceARN, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return c.instanceARN
}

// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file implements the maas.Connector interface for AWS Connect. It is
// responsible for managing the connection to the AWS Connect API, handling
// configuration, and providing a unified interface for the metric collectors.
// The connector is designed to be modular, with its functionality broken down
// into smaller, more focused files.
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

// AWSConnectConnector implements the maas.Connector interface for AWS Connect.
// It holds the AWS Connect client, configuration, and managers for queues and agents.
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

// NewConfigurableAWSConnectConnector creates a new AWS Connect connector and
// defines the command-line flags used to configure it.
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

// NewAWSConnectConnector creates a new AWS Connect connector with the specified
// region and instance ARN. This is used for testing and scenarios where the
// configuration is provided programmatically.
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

// Execute implements the maas.Connector interface. It acts as a dispatcher,
// forwarding commands to the appropriate API wrapper function.
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

// Connect implements the maas.Connector interface. It configures the connector,
// validates the connection to AWS, and initializes the queue and agent managers.
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

// Flags implements the maas.Connector interface. It is a placeholder for any
// future AWS Connect-specific flags that might be needed.
func (c *AWSConnectConnector) Flags(app *kingpin.Application) {
	// AWS Connect specific flags could be added here if needed
	// For now, we rely on config file and CLI flags in the exporter
}

// configure handles the parsing of the configuration file and command-line flags.
// It also sets default values for any missing configuration options.
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

// GetConfig returns the full configuration for collectors to access.
func (c *AWSConnectConnector) GetConfig() AWSConnectConfig {
	return c.config
}

// GetQueueManager exposes the shared QueueManager.
func (c *AWSConnectConnector) GetQueueManager() *QueueManager {
	return c.queueManager
}

// GetAgentManager exposes the shared AgentManager.
func (c *AWSConnectConnector) GetAgentManager() *AgentManager {
	return c.agentManager
}

// GetRoutingProfileQueueManager exposes the routing profile queue manager.
func (c *AWSConnectConnector) GetRoutingProfileQueueManager() *RoutingProfileQueueManager {
	return c.rpQueueManager
}

// GetInstanceID extracts the instance ID from the ARN.
func (c *AWSConnectConnector) GetInstanceID() string {
	// Extract instance ID from ARN
	// ARN format: arn:aws:connect:region:account:instance/instance-id
	parts := strings.Split(c.instanceARN, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return c.instanceARN
}

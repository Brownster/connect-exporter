package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/sirupsen/logrus"
)

const (
	// Version of the agent status collector
	agentStatusCollectorVersion = "1.0.0"

	// AWS Connect API limits
	maxAgentsPerRequest = 100 // Maximum number of agents processed per batch
)

// NewAgentStatusCollector creates a scheduled scraper for agent status metrics
func NewAgentStatusCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.AgentStatus.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("agent_status", NewAgentStatusScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		// Register metrics with descriptions
		maas.WithDescription(app, "agent_status_count", "Number of agents in each status", []string{"status_id", "status_name", "status_type", "region"}),
		maas.WithDescription(app, "agent_status_custom", "Indicator for custom agent statuses (1=custom, 0=built-in)", []string{"status_id", "status_name", "status_type", "region"}),
	)
}

// AgentStatusScraper collects agent status metrics from AWS Connect
type AgentStatusScraper struct {
	config connectors.AWSConnectConfig
}

// NewAgentStatusScraper creates a new scraper instance
func NewAgentStatusScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *AgentStatusScraper {
	return &AgentStatusScraper{
		config: config,
	}
}

// Scrape collects agent status metrics and returns them as a slice of maas.Metric
func (s *AgentStatusScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// Ensure the connector is of type AWSConnectConnector
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: expected AWSConnectConnector, got %T", c)
	}

	// Log the start of the metrics collection process
	logrus.WithFields(logrus.Fields{
		"instance_arn":       s.config.AWS.InstanceARN,
		"region":             s.config.AWS.Region,
		"collector_version":  agentStatusCollectorVersion,
	}).Debug("Starting agent status metrics collection")

	// Initialize a slice to hold the collected metrics
	metrics := make([]maas.Metric, 0)

	var nextToken *string
	totalAgentStatuses := 0

	for {
		// Create the input for ListAgentStatuses API call
		input := &connect.ListAgentStatusesInput{
			InstanceId: &s.config.AWS.InstanceARN,
			MaxResults: aws.Int32(maxAgentsPerRequest),
			NextToken:  nextToken,
		}

		// Fetch agent statuses
		data, err := conn.ListAgentStatuses(context.Background(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list agent statuses: %w", err)
		}

		// Process the agent statuses
		for _, agentStatus := range data.AgentStatusSummaryList {
			if agentStatus.Id == nil || agentStatus.Name == nil {
				logrus.WithField("collector_version", agentStatusCollectorVersion).Debug("Skipping agent status with missing information")
				continue
			}

			labels := []string{
				*agentStatus.Id,
				*agentStatus.Name,
				string(agentStatus.Type),
				s.config.AWS.Region,
			}

			// Create metric for agent status count (always 1 per status type)
			metrics = append(metrics, maas.NewMetric(
				"agent_status_count",
				prometheus.GaugeValue,
				1, // Count of this status type
				labels,
			))

			// Note: DisplayOrder field is not available in current AWS SDK version
			// If needed, this could be retrieved via a separate DescribeAgentStatus call

			// Add custom status indicator
			customValue := 0.0
			if agentStatus.Type == "CUSTOM" {
				customValue = 1.0
			}
			metrics = append(metrics, maas.NewMetric(
				"agent_status_custom",
				prometheus.GaugeValue,
				customValue,
				labels,
			))

			logrus.WithFields(logrus.Fields{
				"status_id":          *agentStatus.Id,
				"status_name":        *agentStatus.Name,
				"status_type":        string(agentStatus.Type),
				"collector_version":  agentStatusCollectorVersion,
			}).Debug("Collected agent status metrics")

			totalAgentStatuses++
		}

		// Check if there are more pages
		if data.NextToken == nil {
			break
		}
		nextToken = data.NextToken

		logrus.WithFields(logrus.Fields{
			"page_statuses":     len(data.AgentStatusSummaryList),
			"total_statuses":    totalAgentStatuses,
			"has_more_pages":    data.NextToken != nil,
			"collector_version": agentStatusCollectorVersion,
		}).Debug("Processed agent status page")
	}

	logrus.WithFields(logrus.Fields{
		"total_metrics":      len(metrics),
		"total_statuses":     totalAgentStatuses,
		"collector_version":  agentStatusCollectorVersion,
	}).Debug("Completed agent status metrics collection")

	return metrics, nil
}
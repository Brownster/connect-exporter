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
		maas.WithDescription(app, "connect_agent_statuses_total", "Number of agents in each status", []string{"status_id", "status_name", "status_type", "region"}),
		// REMOVED: agent_status_custom - Not operationally valuable (just indicates custom vs built-in)
	)
}

// AgentStatusScraper collects agent status metrics from AWS Connect
type AgentStatusScraper struct {
	// No longer store config - get it from connector during scrape
}

// NewAgentStatusScraper creates a new scraper instance
func NewAgentStatusScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *AgentStatusScraper {
	return &AgentStatusScraper{}
}

// Scrape collects agent status metrics and returns them as a slice of maas.Metric
func (s *AgentStatusScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// Ensure the connector is of type AWSConnectConnector
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: expected AWSConnectConnector, got %T", c)
	}

	// Get configuration from connector
	config := conn.GetConfig()

	// Log the start of the metrics collection process
	logrus.WithFields(logrus.Fields{
		"instance_arn":      config.AWS.InstanceARN,
		"region":            config.AWS.Region,
		"collector_version": agentStatusCollectorVersion,
	}).Debug("Starting agent status metrics collection")

	// Initialize a slice to hold the collected metrics
	metrics := make([]maas.Metric, 0)

	var nextToken *string
	totalAgentStatuses := 0

	for {
		// Create the input for ListAgentStatuses API call
		instanceID := conn.GetInstanceID()
		input := &connect.ListAgentStatusesInput{
			InstanceId: &instanceID,
			MaxResults: aws.Int32(maxAgentsPerRequest),
			NextToken:  nextToken,
		}

		// Fetch agent statuses
		data, err := conn.ListAgentStatuses(context.Background(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list agent statuses: %w", err)
		}

		// Process the agent statuses
		statusCounts := make(map[string]struct {
			name  string
			stype string
			count float64
		})

		for _, agentStatus := range data.AgentStatusSummaryList {
			if agentStatus.Id == nil || agentStatus.Name == nil {
				logrus.WithField("collector_version", agentStatusCollectorVersion).Debug("Skipping agent status with missing information")
				continue
			}

			id := *agentStatus.Id
			info, ok := statusCounts[id]
			if !ok {
				info = struct {
					name  string
					stype string
					count float64
				}{name: *agentStatus.Name, stype: string(agentStatus.Type), count: 0}
			}
			info.count++
			statusCounts[id] = info
		}

		for id, info := range statusCounts {
			labels := []string{id, info.name, info.stype, config.AWS.Region}
			metrics = append(metrics, maas.NewMetric(
				"connect_agent_statuses_total",
				prometheus.GaugeValue,
				info.count,
				labels,
			))

			logrus.WithFields(logrus.Fields{
				"status_id":         id,
				"status_name":       info.name,
				"status_type":       info.stype,
				"agent_count":       info.count,
				"collector_version": agentStatusCollectorVersion,
			}).Debug("Collected agent status metrics")

			totalAgentStatuses += int(info.count)
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
		"total_metrics":     len(metrics),
		"total_statuses":    totalAgentStatuses,
		"collector_version": agentStatusCollectorVersion,
	}).Debug("Completed agent status metrics collection")

	return metrics, nil
}

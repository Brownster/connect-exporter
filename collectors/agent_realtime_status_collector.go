package collectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/sirupsen/logrus"
)

const agentRealtimeStatusCollectorVersion = "1.0.0"

// NewAgentRealtimeStatusCollector creates a scheduled scraper for agent realtime status metrics
func NewAgentRealtimeStatusCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.AgentRealtimeStatus.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("agent_realtime_status", NewAgentRealtimeStatusScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "agent_realtime_status_agents", "Number of agents in each realtime status", []string{"status_id", "status_name", "status_type", "region"}),
	)
}

// AgentRealtimeStatusScraper collects realtime agent status metrics
type AgentRealtimeStatusScraper struct{}

func NewAgentRealtimeStatusScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *AgentRealtimeStatusScraper {
	return &AgentRealtimeStatusScraper{}
}

// Scrape collects agent realtime status metrics by counting agents in each status
func (s *AgentRealtimeStatusScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: %T", c)
	}

	cfg := conn.GetConfig()
	instanceID := conn.GetInstanceID()

	// AWS requires at least one filter parameter. Try using Queues filter instead since we can get those
	queues, err := conn.ListQueues(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list queues for filter: %w", err)
	}
	
	if len(queues) == 0 {
		return []maas.Metric{}, nil
	}
	
	// Use the first queue ID as a filter - this should return all agents assigned to any queue
	queueIDs := make([]string, 0, len(queues))
	for _, queue := range queues {
		queueIDs = append(queueIDs, queue.ID)
	}
	
	// Retrieve current user data using queue filter
	input := &connect.GetCurrentUserDataInput{
		InstanceId: &instanceID,
		Filters:    &types.UserDataFilters{
			Queues: queueIDs, // Use queue IDs to get agents assigned to these queues
		},
		MaxResults: aws.Int32(100),
	}

	data, err := conn.GetCurrentUserData(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user data: %w", err)
	}

	// Count agents by status ARN
	counts := make(map[string]int)
	statusInfo := make(map[string]types.AgentStatusReference)

	for _, user := range data.UserDataList {
		if user.Status != nil && user.Status.StatusArn != nil {
			arn := aws.ToString(user.Status.StatusArn)
			counts[arn]++
			if _, exists := statusInfo[arn]; !exists {
				statusInfo[arn] = *user.Status
			}
		}
	}

	metrics := make([]maas.Metric, 0, len(counts))
	for arn, count := range counts {
		info := statusInfo[arn]
		name := aws.ToString(info.StatusName)
		statusID := extractStatusID(arn)
		statusType := "unknown"
		// We don't have type info here; attempt to map using ListAgentStatuses
		// for enrichment
		nameType, ok := getStatusDetails(arn, conn)
		if ok {
			name = aws.ToString(nameType.Name)
			statusType = string(nameType.Type)
		}

		labels := []string{statusID, name, statusType, cfg.AWS.Region}
		metrics = append(metrics, maas.NewMetric("agent_realtime_status_agents", prometheus.GaugeValue, float64(count), labels))
	}

	return metrics, nil
}

// helper to extract status ID from ARN
func extractStatusID(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}

// getStatusDetails fetches status info from ListAgentStatuses if not already known
func getStatusDetails(arn string, conn *connectors.AWSConnectConnector) (types.AgentStatusSummary, bool) {
	instanceID := conn.GetInstanceID()
	input := &connect.ListAgentStatusesInput{
		InstanceId: &instanceID,
		MaxResults: aws.Int32(1000),
	}
	data, err := conn.ListAgentStatuses(context.Background(), input)
	if err != nil {
		logrus.WithError(err).Warn("failed to list agent statuses for enrichment")
		return types.AgentStatusSummary{}, false
	}
	for _, st := range data.AgentStatusSummaryList {
		if st.Arn != nil && aws.ToString(st.Arn) == arn {
			return st, true
		}
	}
	return types.AgentStatusSummary{}, false
}

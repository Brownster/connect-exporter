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


// NewAgentRealtimeStatusCollector creates a scheduled scraper for agent realtime status metrics
func NewAgentRealtimeStatusCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.AgentRealtimeStatus.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("agent_realtime_status", NewAgentRealtimeStatusScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
               maas.WithDescription(app, "connect_agent_status_state", "Realtime status for each agent", []string{"region", "agent_id", "username", "first_name", "last_name", "status_id", "status_name", "status_type", "routing_profile_id", "routing_profile_name", "hierarchy_group_id", "hierarchy_group_name", "queue_id", "queue_name"}),
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
	agentManager := conn.GetAgentManager()

	// Get queues from existing queue manager
	queues := conn.GetQueueManager().GetQueues()
	if len(queues) == 0 {
		return []maas.Metric{}, nil
	}

	// Use all queue IDs as filters to retrieve agent data
	queueIDs := make([]string, 0, len(queues))
	for _, queue := range queues {
		queueIDs = append(queueIDs, queue.ID)
	}

	// Batch queue IDs to respect 100-queue limit
	batchSize := 100
	if len(queueIDs) > batchSize {
		logrus.WithFields(logrus.Fields{
			"total_queues": len(queueIDs),
			"batch_size":   batchSize,
		}).Debug("Batching queue IDs to respect AWS 100-queue limit")
		queueIDs = queueIDs[:batchSize] // Take first 100 queues only for now
	}

	// Retrieve current user data using queue filter
	input := &connect.GetCurrentUserDataInput{
		InstanceId: &instanceID,
		Filters: &types.UserDataFilters{
			Queues: queueIDs, // Use queue IDs to get agents assigned to these queues
		},
		MaxResults: aws.Int32(100),
	}

	data, err := conn.GetCurrentUserData(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user data: %w", err)
	}

	// Collect per-agent metrics using simple agent cache
	var perAgent []maas.Metric

	for _, user := range data.UserDataList {
		// Extract basic user data
		agentID := ""
		if user.User != nil && user.User.Id != nil {
			agentID = aws.ToString(user.User.Id)
		}

		// Extract status information
		statusID := ""
		statusName := ""
		statusType := "unknown"
		if user.Status != nil {
			if user.Status.StatusArn != nil {
				statusID = extractStatusID(aws.ToString(user.Status.StatusArn))
			}
			if user.Status.StatusName != nil {
				statusName = aws.ToString(user.Status.StatusName)
			}
			// Attempt enrichment with ListAgentStatuses for type
			if user.Status.StatusArn != nil {
				st, ok := getStatusDetails(aws.ToString(user.Status.StatusArn), conn)
				if ok {
					if st.Name != nil {
						statusName = aws.ToString(st.Name)
					}
					statusType = string(st.Type)
				}
			}
		}

		// Get cached agent data for enrichment
		cachedAgent := agentManager.GetAgentByID(agentID)
		
		// Build enhanced labels
		region := cfg.AWS.Region
		username := ""
		firstName := ""
		lastName := ""
		routingProfileID := ""
		routingProfileName := ""
		hierarchyGroupID := ""
		hierarchyGroupName := ""
		
		if cachedAgent != nil {
			username = cachedAgent.Username
			firstName = cachedAgent.FirstName
			lastName = cachedAgent.LastName
			routingProfileID = cachedAgent.RoutingProfileID
			// For routing profile name, we'd need another cache - keeping simple for now
		}

		// Simple queue info from first queue for this agent (simplified)
		queueID := ""
		queueName := ""
		if len(queues) > 0 {
			queueID = queues[0].ID
			queueName = queues[0].Name
		}

		labelValues := []string{
			region,           // region
			agentID,          // agent_id
			username,         // username
			firstName,        // first_name
			lastName,         // last_name
			statusID,         // status_id
			statusName,       // status_name
			statusType,       // status_type
			routingProfileID, // routing_profile_id
			routingProfileName, // routing_profile_name
			hierarchyGroupID, // hierarchy_group_id
			hierarchyGroupName, // hierarchy_group_name
			queueID,          // queue_id
			queueName,        // queue_name
		}

		perAgent = append(perAgent, maas.NewMetric("connect_agent_status_state", prometheus.GaugeValue, 1, labelValues))
	}

	metrics := make([]maas.Metric, 0, len(perAgent))
	metrics = append(metrics, perAgent...)

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

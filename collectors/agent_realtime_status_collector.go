// Package collectors provides the metric collectors for the AWS Connect exporter.
//
// This file defines the agent_realtime_status collector, which is responsible
// for gathering live data about the status of each agent in the AWS Connect
// instance. This information is crucial for real-time monitoring of agent
// activity and availability.
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

// NewAgentRealtimeStatusCollector creates a new scheduled scraper for collecting
// agent realtime status metrics. It configures the scraper with a schedule and
// a metric description.
func NewAgentRealtimeStatusCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.AgentRealtimeStatus.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("agent_realtime_status", NewAgentRealtimeStatusScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "connect_agent_status_state", "Realtime status for each agent", []string{"agent_id", "username", "first_name", "last_name", "routing_profile_id", "routing_profile_name", "queue_id", "queue_name", "status_id", "status_name", "status_type", "region"}),
	)
}

// AgentRealtimeStatusScraper is the scraper implementation for agent realtime status metrics.
type AgentRealtimeStatusScraper struct{}

// NewAgentRealtimeStatusScraper creates a new AgentRealtimeStatusScraper.
func NewAgentRealtimeStatusScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *AgentRealtimeStatusScraper {
	return &AgentRealtimeStatusScraper{}
}

// Scrape collects agent realtime status metrics by fetching the current user data
// from the AWS Connect API and transforming it into Prometheus metrics.
func (s *AgentRealtimeStatusScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: %T", c)
	}

	cfg := conn.GetConfig()
	instanceID := conn.GetInstanceID()

	// AWS requires at least one filter parameter. Use discovered queues
	queues := conn.GetQueueManager().GetQueues()
	if len(queues) == 0 {
		return []maas.Metric{}, nil
	}

	// Use all queue IDs as filters to retrieve agent data
	// for agents assigned to any queue
	queueIDs := make([]string, 0, len(queues))
	for _, queue := range queues {
		queueIDs = append(queueIDs, queue.ID)
	}

	logrus.WithFields(logrus.Fields{
		"total_queues": len(queueIDs),
		"instance_id":  instanceID,
	}).Debug("Starting agent realtime status collection")

	// AWS Connect GetCurrentUserData API has a 100 queue limit
	// Process queues in batches to respect this limit
	const maxQueuesPerRequest = 100
	allUserData := make([]types.UserData, 0)

	for i := 0; i < len(queueIDs); i += maxQueuesPerRequest {
		end := i + maxQueuesPerRequest
		if end > len(queueIDs) {
			end = len(queueIDs)
		}

		batchQueueIDs := queueIDs[i:end]
		
		logrus.WithFields(logrus.Fields{
			"batch_start": i,
			"batch_end":   end,
			"batch_size":  len(batchQueueIDs),
		}).Debug("Processing agent queue batch")

		// Retrieve current user data for this batch of queues
		input := &connect.GetCurrentUserDataInput{
			InstanceId: &instanceID,
			Filters: &types.UserDataFilters{
				Queues: batchQueueIDs,
			},
			MaxResults: aws.Int32(100),
		}

		data, err := conn.GetCurrentUserData(context.Background(), input)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"batch_start": i,
				"batch_end":   end,
				"batch_size":  len(batchQueueIDs),
			}).Error("Failed to get current user data for batch, continuing with next batch")
			continue // Continue with other batches even if one fails
		}

		// Collect users from this batch
		if data.UserDataList != nil {
			allUserData = append(allUserData, data.UserDataList...)
		}
	}

	logrus.WithFields(logrus.Fields{
		"total_users":  len(allUserData),
		"total_queues": len(queueIDs),
	}).Debug("Completed agent realtime status batching")

	// Get agent metadata from shared AgentManager
	agentMap := conn.GetAgentManager().GetAgentMap()

	// Get routing profile metadata from shared RoutingProfileManager
	routingProfileMap := conn.GetRoutingProfileManager().GetRoutingProfileMap()

	// Collect per-agent metrics
	var perAgent []maas.Metric

	rpManager := conn.GetRoutingProfileQueueManager()

	for _, user := range allUserData {

		agentID := ""
		if user.User != nil && user.User.Id != nil {
			agentID = aws.ToString(user.User.Id)
		}
		username := ""
		firstName := ""
		lastName := ""
		if agent, exists := agentMap[agentID]; exists {
			username = agent.Username
			firstName = agent.FirstName
			lastName = agent.LastName
		}
		rpID := ""
		rpName := ""
		if user.RoutingProfile != nil && user.RoutingProfile.Id != nil {
			rpID = aws.ToString(user.RoutingProfile.Id)
			// Look up routing profile name from routing profile map
			if rp, exists := routingProfileMap[rpID]; exists {
				rpName = rp.Name
			}
		}
		agentQueues := []connectors.Queue{{ID: "", Name: "", Type: ""}}
		if rpID != "" && rpManager != nil {
			queues, err := rpManager.GetQueuesForRoutingProfile(rpID)
			if err != nil {
				logrus.WithError(err).Debug("failed to list routing profile queues")
			} else if len(queues) > 0 {
				agentQueues = queues
			}
		}

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

		for _, queue := range agentQueues {
			labels := []string{agentID, username, firstName, lastName, rpID, rpName, queue.ID, queue.Name, statusID, statusName, statusType, cfg.AWS.Region}
			perAgent = append(perAgent, maas.NewMetric("connect_agent_status_state", prometheus.GaugeValue, 1, labels))
		}
	}

	metrics := make([]maas.Metric, 0, len(perAgent))
	metrics = append(metrics, perAgent...)

	return metrics, nil
}

// extractStatusID is a helper function to extract the status ID from an ARN.
func extractStatusID(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}

// getStatusDetails fetches additional status information, such as the status type,
// by calling the ListAgentStatuses API.
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

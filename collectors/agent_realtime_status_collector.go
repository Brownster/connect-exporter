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
		maas.WithDescription(app, "connect_agent_status_state", "Realtime status for each agent", []string{"agent_id", "username", "routing_profile_id", "queue_id", "status_id", "status_name", "status_type", "region"}),
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

	// Get agent metadata from shared AgentManager
	agentMap := conn.GetAgentManager().GetAgentMap()

	// Collect per-agent metrics
	var perAgent []maas.Metric

	rpManager := conn.GetRoutingProfileQueueManager()

	for _, user := range data.UserDataList {

		agentID := ""
		if user.User != nil && user.User.Id != nil {
			agentID = aws.ToString(user.User.Id)
		}
		username := ""
		if agent, exists := agentMap[agentID]; exists {
			username = agent.Username
		}
		rpID := ""
		if user.RoutingProfile != nil && user.RoutingProfile.Id != nil {
			rpID = aws.ToString(user.RoutingProfile.Id)
		}
		agentQueueIDs := []string{""}
		if rpID != "" && rpManager != nil {
			ids, err := rpManager.GetQueuesForRoutingProfile(rpID)
			if err != nil {
				logrus.WithError(err).Debug("failed to list routing profile queues")
			} else if len(ids) > 0 {
				agentQueueIDs = ids
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

		for _, qid := range agentQueueIDs {
			labels := []string{agentID, username, rpID, qid, statusID, statusName, statusType, cfg.AWS.Region}
			perAgent = append(perAgent, maas.NewMetric("connect_agent_status_state", prometheus.GaugeValue, 1, labels))
		}
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

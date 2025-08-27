package collectors

import (
	"context"
	"fmt"
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

const agentStateCollectorVersion = "1.0.0"

// NewAgentStateCollector creates a scheduled scraper for aggregated agent state metrics
func NewAgentStateCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.AgentRealtimeStatus.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("agent_state", NewAgentStateScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "agents", "The number of agents in a specific state, grouped by queue.", []string{"queue", "state"}),
	)
}

// AgentStateScraper collects aggregated agent state metrics
type AgentStateScraper struct{}

func NewAgentStateScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *AgentStateScraper {
	return &AgentStateScraper{}
}

func normalizeState(metric string) string {
	switch metric {
	case "AGENTS_AVAILABLE":
		return "Available"
	case "AGENTS_STAFFED":
		return "Staffed"
	case "AGENTS_ON_CONTACT":
		return "OnContact"
	case "AGENTS_AFTER_CONTACT_WORK":
		return "AfterContactWork"
	case "AGENTS_NON_PRODUCTIVE":
		return "NonProductive"
	case "AGENTS_ERROR":
		return "Error"
	default:
		return metric
	}
}

// Scrape collects and aggregates agent state counts by queue
func (s *AgentStateScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: %T", c)
	}

	queues := conn.GetQueueManager().GetQueues()
	if len(queues) == 0 {
		return []maas.Metric{}, nil
	}

	queueIDs := make([]string, len(queues))
	queueNames := make(map[string]string)
	for i, q := range queues {
		queueIDs[i] = q.ID
		queueNames[q.ID] = q.Name
	}

	instanceID := conn.GetInstanceID()
	input := &connect.GetCurrentMetricDataInput{
		InstanceId: &instanceID,
		Filters: &types.Filters{
			Queues:   queueIDs,
			Channels: []types.Channel{types.ChannelVoice, types.ChannelChat, types.ChannelTask},
		},
		Groupings: []types.Grouping{types.GroupingQueue},
		CurrentMetrics: []types.CurrentMetric{
			{Name: types.CurrentMetricNameAgentsAvailable, Unit: types.UnitCount},
			{Name: types.CurrentMetricNameAgentsStaffed, Unit: types.UnitCount},
			{Name: types.CurrentMetricNameAgentsOnContact, Unit: types.UnitCount},
			{Name: types.CurrentMetricNameAgentsAfterContactWork, Unit: types.UnitCount},
			{Name: types.CurrentMetricNameAgentsNonProductive, Unit: types.UnitCount},
			{Name: types.CurrentMetricNameAgentsError, Unit: types.UnitCount},
		},
	}

	data, err := conn.GetCurrentMetricData(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent state metrics: %w", err)
	}

	metrics := make([]maas.Metric, 0)
	for _, result := range data.MetricResults {
		if result.Dimensions == nil || result.Dimensions.Queue == nil || result.Dimensions.Queue.Id == nil {
			continue
		}
		qid := aws.ToString(result.Dimensions.Queue.Id)
		qname := queueNames[qid]
		for _, coll := range result.Collections {
			if coll.Metric == nil || coll.Value == nil {
				continue
			}
			state := normalizeState(string(coll.Metric.Name))
			metrics = append(metrics, maas.NewMetric(
				"agents",
				prometheus.GaugeValue,
				*coll.Value,
				[]string{qname, state},
			))
		}
	}

	logrus.WithFields(logrus.Fields{
		"collector_version": agentStateCollectorVersion,
		"metric_count":      len(metrics),
	}).Debug("Collected aggregated agent state metrics")

	return metrics, nil
}

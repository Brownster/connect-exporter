package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/avast/retry-go/v4"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/sirupsen/logrus"
)

const (
	// Version of the collector, following semantic versioning
	realtimeQueueCollectorVersion = "1.0.0"

	// AWS Connect API limits and configuration
	maxQueuesPerRequest = 100 // Maximum number of queues that can be processed in a single API request
	metricNamespace     = "aws_connect"

	// AWS API retry configuration
	defaultRetryAttempts = 3               // Maximum number of retry attempts for failed API calls
	defaultRetryDelay    = 1 * time.Second // Initial delay between retry attempts
	defaultMaxRetryDelay = 5 * time.Second // Maximum delay between retry attempts
)

// NewRealtimeQueueCollector creates a scheduled scraper for realtime queue metrics
func NewRealtimeQueueCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.RealtimeQueue.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("realtime_queue", NewRealtimeQueueScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		// Register metrics with descriptions
		maas.WithDescription(app, "agent_state_agents", "Number of agents in each state", []string{"queue_id", "queue_name", "queue_type", "channel", "state", "region"}),
		maas.WithDescription(app, "slots_available", "Number of contact slots available", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
		maas.WithDescription(app, "slots_active", "Number of contact slots currently active", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
		maas.WithDescription(app, "contacts_in_queue", "Number of contacts currently waiting in queue", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
		maas.WithDescription(app, "oldest_contact_age_seconds", "Age in seconds of the oldest contact in queue", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
	)
}

// RealtimeQueueScraper collects real-time metrics from AWS Connect queues
type RealtimeQueueScraper struct {
	// No longer store config - get it from connector during scrape
}

// NewRealtimeQueueScraper creates a new scraper instance
func NewRealtimeQueueScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *RealtimeQueueScraper {
	return &RealtimeQueueScraper{}
}

// Scrape performs the actual data collection
func (s *RealtimeQueueScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	// Ensure the connector is of type AWSConnectConnector
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: expected AWSConnectConnector, got %T", c)
	}

	// Get configuration from connector
	config := conn.GetConfig()

	logrus.WithFields(logrus.Fields{
		"instance_arn":      config.AWS.InstanceARN,
		"region":            config.AWS.Region,
		"collector_version": realtimeQueueCollectorVersion,
	}).Debug("Starting realtime queue metrics collection")

	// Get all queues for the instance
	queues, err := conn.ListQueues(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	if len(queues) == 0 {
		logrus.WithField("instance_arn", config.AWS.InstanceARN).Warn("No queues found for instance")
		return []maas.Metric{}, nil
	}

	logrus.WithFields(logrus.Fields{
		"queue_count":       len(queues),
		"collector_version": realtimeQueueCollectorVersion,
	}).Debug("Found queues for metrics collection")

	// Collect metrics from all queues
	allMetrics := make([]maas.Metric, 0)

	// Define the current metrics we want to collect
	currentMetrics := []types.CurrentMetric{
		{
			Name: types.CurrentMetricNameAgentsNonProductive,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameSlotsAvailable,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameSlotsActive,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameContactsInQueue,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameOldestContactAge,
			Unit: types.UnitSeconds,
		},
		{
			Name: types.CurrentMetricNameAgentsAvailable,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameAgentsStaffed,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameAgentsOnContact,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameAgentsAfterContactWork,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameAgentsError,
			Unit: types.UnitCount,
		},
	}

	// Process queues in batches to respect API limits
	for i := 0; i < len(queues); i += maxQueuesPerRequest {
		end := i + maxQueuesPerRequest
		if end > len(queues) {
			end = len(queues)
		}

		batchQueues := queues[i:end]
		batchMetrics, err := s.processQueueBatch(conn, batchQueues, currentMetrics, config, i, end)
		if err != nil {
			// Log error but continue with other batches
			logrus.WithFields(logrus.Fields{
				"batch_start":       i,
				"batch_end":         end,
				"error":             err,
				"collector_version": realtimeQueueCollectorVersion,
			}).Error("Failed to process queue batch, continuing with next batch")
			continue
		}

		allMetrics = append(allMetrics, batchMetrics...)
	}

	logrus.WithFields(logrus.Fields{
		"total_metrics":     len(allMetrics),
		"queue_count":       len(queues),
		"collector_version": realtimeQueueCollectorVersion,
	}).Debug("Completed realtime queue metrics collection")

	return allMetrics, nil
}

// processQueueBatch handles metrics collection for a batch of queues
func (s *RealtimeQueueScraper) processQueueBatch(conn *connectors.AWSConnectConnector, batchQueues []connectors.Queue, currentMetrics []types.CurrentMetric, config connectors.AWSConnectConfig, batchStart, batchEnd int) ([]maas.Metric, error) {
	metrics := make([]maas.Metric, 0)
	queueIDs := make([]string, len(batchQueues))
	queueMap := make(map[string]connectors.Queue)

	for j, queue := range batchQueues {
		queueIDs[j] = queue.ID
		queueMap[queue.ID] = queue
	}

	logrus.WithFields(logrus.Fields{
		"batch_size":        len(batchQueues),
		"batch_start":       batchStart,
		"batch_end":         batchEnd,
		"collector_version": realtimeQueueCollectorVersion,
	}).Debug("Processing queue batch")

	instanceID := conn.GetInstanceID()
	input := &connect.GetCurrentMetricDataInput{
		InstanceId: &instanceID,
		Filters: &types.Filters{
			Queues:   queueIDs,
			Channels: []types.Channel{types.ChannelVoice, types.ChannelChat, types.ChannelTask},
		},
		Groupings:      []types.Grouping{types.GroupingQueue, types.GroupingChannel},
		CurrentMetrics: currentMetrics,
	}

	var data *connect.GetCurrentMetricDataOutput
	err := retry.Do(
		func() error {
			var err error
			data, err = conn.GetCurrentMetricData(context.Background(), input)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error":             err,
					"collector_version": realtimeQueueCollectorVersion,
				}).Warn("Failed to get metrics for queue batch, retrying...")
				return err
			}
			return nil
		},
		retry.Attempts(defaultRetryAttempts),
		retry.Delay(defaultRetryDelay),
		retry.MaxDelay(defaultMaxRetryDelay),
		retry.OnRetry(func(n uint, err error) {
			logrus.WithFields(logrus.Fields{
				"attempt":           n + 1,
				"error":             err,
				"collector_version": realtimeQueueCollectorVersion,
			}).Debug("Retrying queue batch metrics collection")
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get metrics after retries: %w", err)
	}

	// Process the metric results
	for _, result := range data.MetricResults {
		if result.Collections == nil {
			continue
		}

		queueID := ""
		if result.Dimensions != nil && result.Dimensions.Queue != nil && result.Dimensions.Queue.Id != nil {
			queueID = *result.Dimensions.Queue.Id
		}

		queue, exists := queueMap[queueID]
		if !exists {
			logrus.WithFields(logrus.Fields{
				"queue_id":          queueID,
				"collector_version": realtimeQueueCollectorVersion,
			}).Debug("Queue not found in map, skipping")
			continue
		}

		channel := ""
		if result.Dimensions != nil && result.Dimensions.Channel != "" {
			channel = string(result.Dimensions.Channel)
		}

		// Base labels for this queue
		baseLabels := []string{
			queue.ID,
			queue.Name,
			queue.Type,
			channel,
		}

		// Process each metric collection
		for _, collection := range result.Collections {
			if collection.Metric == nil || collection.Value == nil {
				continue
			}

			metricName := string(collection.Metric.Name)
			metricValue := *collection.Value

			// Map AWS Connect metric names to our Prometheus metric names
			var (
				prometheusMetricName string
				stateLabel           string
			)
			switch collection.Metric.Name {
			case types.CurrentMetricNameAgentsNonProductive:
				prometheusMetricName = "agent_state_agents"
				stateLabel = "NonProductive"
			case types.CurrentMetricNameSlotsAvailable:
				prometheusMetricName = "slots_available"
			case types.CurrentMetricNameSlotsActive:
				prometheusMetricName = "slots_active"
			case types.CurrentMetricNameContactsInQueue:
				prometheusMetricName = "contacts_in_queue"
			case types.CurrentMetricNameOldestContactAge:
				prometheusMetricName = "oldest_contact_age_seconds"
			case types.CurrentMetricNameAgentsAvailable:
				prometheusMetricName = "agent_state_agents"
				stateLabel = "Available"
			case types.CurrentMetricNameAgentsStaffed:
				prometheusMetricName = "agent_state_agents"
				stateLabel = "Staffed"
			case types.CurrentMetricNameAgentsOnContact:
				prometheusMetricName = "agent_state_agents"
				stateLabel = "OnContact"
			case types.CurrentMetricNameAgentsAfterContactWork:
				prometheusMetricName = "agent_state_agents"
				stateLabel = "AfterContactWork"
			case types.CurrentMetricNameAgentsError:
				prometheusMetricName = "agent_state_agents"
				stateLabel = "Error"
			default:
				logrus.WithFields(logrus.Fields{
					"metric_name":       metricName,
					"collector_version": realtimeQueueCollectorVersion,
				}).Debug("Unknown metric name, skipping")
				continue
			}

			// Construct labels for this metric
			labels := baseLabels
			if prometheusMetricName == "agent_state_agents" {
				labels = append(labels, stateLabel, config.AWS.Region)
			} else {
				labels = append(labels, config.AWS.Region)
			}

			// Create the metric
			metrics = append(metrics, maas.NewMetric(
				prometheusMetricName,
				prometheus.GaugeValue,
				metricValue,
				labels,
			))

			logrus.WithFields(logrus.Fields{
				"metric_name":       prometheusMetricName,
				"metric_value":      metricValue,
				"queue_name":        queue.Name,
				"collector_version": realtimeQueueCollectorVersion,
			}).Debug("Collected queue metric")
		}
	}

	return metrics, nil
}

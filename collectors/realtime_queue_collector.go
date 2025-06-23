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
	defaultRetryAttempts = 3              // Maximum number of retry attempts for failed API calls
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
		maas.WithDescription(app, "contacts_in_queue", "Number of contacts currently waiting in queue", []string{"queue_id", "queue_name", "queue_type", "region"}),
		maas.WithDescription(app, "oldest_contact_age", "Age in seconds of the oldest contact in queue", []string{"queue_id", "queue_name", "queue_type", "region"}),
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
		"instance_arn":       config.AWS.InstanceARN,
		"region":             config.AWS.Region,
		"collector_version":  realtimeQueueCollectorVersion,
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
		"queue_count":        len(queues),
		"collector_version":  realtimeQueueCollectorVersion,
	}).Debug("Found queues for metrics collection")

	// Collect metrics from all queues
	allMetrics := make([]maas.Metric, 0)

	// Define the current metrics we want to collect
	currentMetrics := []types.CurrentMetric{
		{
			Name: types.CurrentMetricNameContactsInQueue,
			Unit: types.UnitCount,
		},
		{
			Name: types.CurrentMetricNameOldestContactAge,
			Unit: types.UnitSeconds,
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

	input := &connect.GetCurrentMetricDataInput{
		InstanceId:     &config.AWS.InstanceARN,
		Filters:        &types.Filters{Queues: queueIDs},
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

		// Create labels for this queue
		labels := []string{
			queue.ID,
			queue.Name,
			queue.Type,
			config.AWS.Region,
		}

		// Process each metric collection
		for _, collection := range result.Collections {
			if collection.Metric == nil || collection.Value == nil {
				continue
			}

			metricName := string(collection.Metric.Name)
			metricValue := *collection.Value

			// Map AWS Connect metric names to our Prometheus metric names
			var prometheusMetricName string
			switch collection.Metric.Name {
			case types.CurrentMetricNameContactsInQueue:
				prometheusMetricName = "contacts_in_queue"
			case types.CurrentMetricNameOldestContactAge:
				prometheusMetricName = "oldest_contact_age"
			default:
				logrus.WithFields(logrus.Fields{
					"metric_name":       metricName,
					"collector_version": realtimeQueueCollectorVersion,
				}).Debug("Unknown metric name, skipping")
				continue
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
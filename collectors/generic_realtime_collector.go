package collectors

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
	// AWS Connect API limits and configuration for generic collector
	genericMaxQueuesPerRequest = 100 // Maximum number of queues that can be processed in a single API request

	// AWS API retry configuration for generic collector
	genericDefaultRetryAttempts = 3               // Maximum number of retry attempts for failed API calls
	genericDefaultRetryDelay    = 1 * time.Second // Initial delay between retry attempts
	genericDefaultMaxRetryDelay = 5 * time.Second // Maximum delay between retry attempts
)

// NewGenericRealtimeCollector creates a scheduled scraper for generic realtime metrics
func NewGenericRealtimeCollector(app *kingpin.Application, cfg connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(cfg.Collectors.GenericRealtime.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper(
		"generic_realtime",
		NewGenericRealtimeScraper(),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "generic_metric", "Generic realtime metric", []string{"collector", "metric_name", "queue_id", "queue_name", "queue_type", "channel", "region", "state"}),
	)
}


// GenericRealtimeScraper gathers realtime metrics defined in configuration
type GenericRealtimeScraper struct {
	metricsCache struct {
		metrics   []maas.Metric
		timestamp time.Time
	}
	metricsTTL time.Duration
	mu         sync.Mutex
}

// NewGenericRealtimeScraper constructs the scraper
func NewGenericRealtimeScraper() *GenericRealtimeScraper {
	return &GenericRealtimeScraper{
		metricsTTL: 15 * time.Second, // Default cache TTL
	}
}

// Scrape implements the Scraper interface
func (s *GenericRealtimeScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: expected AWSConnectConnector, got %T", c)
	}

	cfg := conn.GetConfig()
	rt := cfg.Collectors.GenericRealtime
	if !rt.Enabled {
		return []maas.Metric{}, nil
	}

	// Check cache first
	now := time.Now()
	s.mu.Lock()
	if now.Sub(s.metricsCache.timestamp) < s.metricsTTL && len(s.metricsCache.metrics) > 0 {
		cached := s.metricsCache.metrics
		s.mu.Unlock()
		logrus.Debug("Returning cached generic realtime metrics")
		return cached, nil
	}
	s.mu.Unlock()

	queues := conn.GetQueueManager().GetQueues()
	if len(queues) == 0 {
		logrus.WithField("instance_arn", cfg.AWS.InstanceARN).Warn("No queues found for generic realtime collector")
		return []maas.Metric{}, nil
	}

	allMetrics := make([]maas.Metric, 0)

	// Process each collector configuration
	for _, col := range rt.Collectors {
		colMetrics, err := s.processCollectorConfig(conn, queues, col, cfg)
		if err != nil {
			logrus.WithError(err).WithField("collector", col.Name).Error("Failed to process collector config")
			continue
		}
		allMetrics = append(allMetrics, colMetrics...)
	}

	// Cache the results
	s.mu.Lock()
	s.metricsCache.metrics = allMetrics
	s.metricsCache.timestamp = time.Now()
	s.mu.Unlock()

	return allMetrics, nil
}

// processCollectorConfig handles metrics collection for a single collector configuration
func (s *GenericRealtimeScraper) processCollectorConfig(conn *connectors.AWSConnectConnector, queues []connectors.Queue, col connectors.GenericMetricConfig, cfg connectors.AWSConnectConfig) ([]maas.Metric, error) {
	allMetrics := make([]maas.Metric, 0)

	// Convert metric definitions to AWS types
	currentMetrics := make([]types.CurrentMetric, len(col.Metrics))
	metricMap := make(map[string]connectors.MetricDefinition)
	for i, m := range col.Metrics {
		currentMetrics[i] = types.CurrentMetric{
			Name: types.CurrentMetricName(m.AWSName),
			Unit: types.UnitCount, // Default to count, could be configurable
		}
		metricMap[m.AWSName] = m
	}

	// Convert groupings to AWS types
	groupings := make([]types.Grouping, len(col.Groupings))
	for i, g := range col.Groupings {
		groupings[i] = types.Grouping(g)
	}

	// Process queues in batches to respect API limits
	for i := 0; i < len(queues); i += genericMaxQueuesPerRequest {
		end := i + genericMaxQueuesPerRequest
		if end > len(queues) {
			end = len(queues)
		}

		batchQueues := queues[i:end]
		batchMetrics, err := s.processQueueBatch(conn, batchQueues, currentMetrics, groupings, metricMap, col, cfg)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"collector":   col.Name,
				"batch_start": i,
				"batch_end":   end,
			}).Error("Failed to process queue batch")
			continue
		}

		allMetrics = append(allMetrics, batchMetrics...)
	}

	return allMetrics, nil
}

// processQueueBatch handles metrics collection for a batch of queues
func (s *GenericRealtimeScraper) processQueueBatch(conn *connectors.AWSConnectConnector, batchQueues []connectors.Queue, currentMetrics []types.CurrentMetric, groupings []types.Grouping, metricMap map[string]connectors.MetricDefinition, col connectors.GenericMetricConfig, cfg connectors.AWSConnectConfig) ([]maas.Metric, error) {
	metrics := make([]maas.Metric, 0)
	queueIDs := make([]string, len(batchQueues))
	queueMap := make(map[string]connectors.Queue)

	for j, queue := range batchQueues {
		queueIDs[j] = queue.ID
		queueMap[queue.ID] = queue
	}

	instanceID := conn.GetInstanceID()
	input := &connect.GetCurrentMetricDataInput{
		InstanceId: &instanceID,
		Filters: &types.Filters{
			Queues:   queueIDs,
			Channels: []types.Channel{types.ChannelVoice, types.ChannelChat, types.ChannelTask},
		},
		Groupings:      groupings,
		CurrentMetrics: currentMetrics,
	}

	// Apply additional filters from configuration
	for _, f := range col.Filters {
		switch strings.ToUpper(f.Key) {
		case "CHANNELS":
			channels := make([]types.Channel, len(f.Values))
			for i, v := range f.Values {
				channels[i] = types.Channel(strings.ToUpper(v))
			}
			input.Filters.Channels = channels
		}
	}

	var data *connect.GetCurrentMetricDataOutput
	err := retry.Do(
		func() error {
			var err error
			data, err = conn.GetCurrentMetricData(context.Background(), input)
			if err != nil {
				logrus.WithError(err).WithField("collector", col.Name).Warn("Failed to get current metric data, retrying...")
				return err
			}
			return nil
		},
		retry.Attempts(genericDefaultRetryAttempts),
		retry.Delay(genericDefaultRetryDelay),
		retry.MaxDelay(genericDefaultMaxRetryDelay),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get current metric data after retries: %w", err)
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
			logrus.WithField("queue_id", queueID).Debug("Queue not found in map, skipping")
			continue
		}

		channel := ""
		if result.Dimensions != nil && result.Dimensions.Channel != "" {
			channel = string(result.Dimensions.Channel)
		}

		// Process each metric collection
		for _, collection := range result.Collections {
			if collection.Metric == nil || collection.Value == nil {
				continue
			}

			awsMetricName := string(collection.Metric.Name)
			def, exists := metricMap[awsMetricName]
			if !exists {
				logrus.WithField("metric_name", awsMetricName).Debug("Metric not in configuration, skipping")
				continue
			}

			// Build base labels
			labels := []string{
				col.Name,       // collector name
				def.PrometheusName, // metric name
				queue.ID,
				queue.Name,
				queue.Type,
				channel,
				cfg.AWS.Region,
			}

			// Add state label if configured
			if state, hasState := def.Labels["state"]; hasState {
				labels = append(labels, state)
			} else {
				labels = append(labels, "")
			}

			// Create the metric
			metrics = append(metrics, maas.NewMetric(
				"generic_metric",
				prometheus.GaugeValue,
				*collection.Value,
				labels,
			))

			logrus.WithFields(logrus.Fields{
				"collector":     col.Name,
				"metric_name":   def.PrometheusName,
				"metric_value":  *collection.Value,
				"queue_name":    queue.Name,
			}).Debug("Collected generic realtime metric")
		}
	}

	return metrics, nil
}

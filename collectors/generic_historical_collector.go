// Package collectors provides the metric collectors for the AWS Connect exporter.
//
// This file defines the generic_historical collector, which is responsible for
// gathering historical metrics from AWS Connect. This collector is highly
// configurable, allowing users to define which metrics to collect, how to group
// them, and what filters to apply. This flexibility is crucial for adapting the
// exporter to different monitoring needs without requiring code changes.
package collectors

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

const (
	// genericHistoricalMaxQueuesPerRequest is the maximum number of queues that can be processed in a single API request.
	genericHistoricalMaxQueuesPerRequest = 100
)

// NewGenericHistoricalCollector creates a new scheduled scraper for collecting
// generic historical metrics. It configures the scraper with a schedule and a
// metric description.
func NewGenericHistoricalCollector(app *kingpin.Application, cfg connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(cfg.Collectors.GenericHistorical.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper(
		"generic_historical",
		NewGenericHistoricalScraper(),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "generic_historical_metric", "Generic historical metric", []string{"collector", "metric_name", "queue_id", "queue_name", "queue_type", "channel", "region", "direction"}),
	)
}

// GenericHistoricalScraper is the scraper implementation for generic historical metrics.
// It includes a cache to avoid redundant API calls.
type GenericHistoricalScraper struct {
	metricsCache struct {
		metrics   []maas.Metric
		timestamp time.Time
	}
	metricsTTL time.Duration
	mu         sync.Mutex
}

// NewGenericHistoricalScraper creates a new GenericHistoricalScraper.
func NewGenericHistoricalScraper() *GenericHistoricalScraper {
	return &GenericHistoricalScraper{
		metricsTTL: 15 * time.Second, // Default cache TTL
	}
}

// Scrape collects generic historical metrics by fetching data from the AWS Connect
// API based on the user's configuration. It uses a cache to avoid making
// redundant API calls.
func (s *GenericHistoricalScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: expected AWSConnectConnector, got %T", c)
	}

	cfg := conn.GetConfig()
	hc := cfg.Collectors.GenericHistorical
	if !hc.Enabled {
		return []maas.Metric{}, nil
	}

	// Align cache TTL with schedule interval
	s.metricsTTL = time.Duration(hc.Interval) * time.Second

	// Check cache first
	now := time.Now()
	s.mu.Lock()
	if now.Sub(s.metricsCache.timestamp) < s.metricsTTL && len(s.metricsCache.metrics) > 0 {
		cached := s.metricsCache.metrics
		s.mu.Unlock()
		logrus.Debug("Returning cached generic historical metrics")
		return cached, nil
	}
	s.mu.Unlock()

	queues := conn.GetQueueManager().GetQueues()
	if len(queues) == 0 {
		logrus.WithField("instance_arn", cfg.AWS.InstanceARN).Warn("No queues found for generic historical collector")
		return []maas.Metric{}, nil
	}

	allMetrics := make([]maas.Metric, 0)

	// Process each collector configuration
	for _, col := range hc.Collectors {
		colMetrics, err := s.processHistoricalCollectorConfig(conn, queues, col, cfg)
		if err != nil {
			logrus.WithError(err).WithField("collector", col.Name).Error("Failed to process historical collector config")
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

// processHistoricalCollectorConfig handles the metrics collection for a single
// historical collector configuration.
func (s *GenericHistoricalScraper) processHistoricalCollectorConfig(conn *connectors.AWSConnectConnector, queues []connectors.Queue, col connectors.GenericHistoricalMetricConfig, cfg connectors.AWSConnectConfig) ([]maas.Metric, error) {
	allMetrics := make([]maas.Metric, 0)

	// Build threshold lookup map
	thresholdMap := make(map[string]connectors.MetricThreshold)
	for _, threshold := range col.Thresholds {
		thresholdMap[threshold.Name] = threshold
	}

	// Convert metric definitions to AWS types
	metricsToFetch := make([]types.MetricV2, len(col.Metrics))
	metricMap := make(map[string]connectors.MetricDefinition)
	for i, m := range col.Metrics {
		metricsToFetch[i] = types.MetricV2{Name: aws.String(m.AWSName)}

		// Add MetricFilters if specified
		if len(m.MetricFilters) > 0 {
			filters := make([]types.MetricFilterV2, len(m.MetricFilters))
			for j, f := range m.MetricFilters {
				filters[j] = types.MetricFilterV2{
					MetricFilterKey:    aws.String(f.Key),
					MetricFilterValues: f.Values,
				}
			}
			metricsToFetch[i].MetricFilters = filters
		}

		// Add Threshold if referenced
		if m.ThresholdRef != "" {
			if threshold, exists := thresholdMap[m.ThresholdRef]; exists {
				metricsToFetch[i].Threshold = []types.ThresholdV2{
					{
						Comparison:     aws.String(threshold.Comparison),
						ThresholdValue: aws.Float64(threshold.Value),
					},
				}
			}
		}

		metricMap[m.AWSName] = m
	}

	// Calculate time range (midnight UTC to now)
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Process queues in batches to respect API limits
	for i := 0; i < len(queues); i += genericHistoricalMaxQueuesPerRequest {
		end := i + genericHistoricalMaxQueuesPerRequest
		if end > len(queues) {
			end = len(queues)
		}

		batchQueues := queues[i:end]
		batchMetrics, err := s.processQueueBatch(conn, batchQueues, metricsToFetch, metricMap, col, cfg, midnight, now)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"collector":   col.Name,
				"batch_start": i,
				"batch_end":   end,
			}).Error("Failed to process historical queue batch")
			continue
		}

		allMetrics = append(allMetrics, batchMetrics...)
	}

	return allMetrics, nil
}

// processQueueBatch handles the metrics collection for a single batch of queues.
func (s *GenericHistoricalScraper) processQueueBatch(conn *connectors.AWSConnectConnector, batchQueues []connectors.Queue, metricsToFetch []types.MetricV2, metricMap map[string]connectors.MetricDefinition, col connectors.GenericHistoricalMetricConfig, cfg connectors.AWSConnectConfig, startTime, endTime time.Time) ([]maas.Metric, error) {
	metrics := make([]maas.Metric, 0)
	batchIDs := make([]string, len(batchQueues))
	queueMap := make(map[string]connectors.Queue)

	for j, queue := range batchQueues {
		batchIDs[j] = queue.ID
		queueMap[queue.ID] = queue
	}

	// Build filters
	filters := []types.FilterV2{
		{FilterKey: aws.String("QUEUE"), FilterValues: batchIDs},
	}

	// Apply additional filters from configuration
	for _, f := range col.Filters {
		filters = append(filters, types.FilterV2{
			FilterKey:    aws.String(strings.ToUpper(f.Key)),
			FilterValues: f.Values,
		})
	}

	input := &connect.GetMetricDataV2Input{
		ResourceArn: &cfg.AWS.InstanceARN,
		StartTime:   &startTime,
		EndTime:     &endTime,
		Filters:     filters,
		Groupings:   col.Groupings,
		Metrics:     metricsToFetch,
		Interval: &types.IntervalDetails{
			IntervalPeriod: types.IntervalPeriodTotal,
			TimeZone:       aws.String("UTC"),
		},
	}

	data, err := conn.GetMetricDataV2(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical metric data: %w", err)
	}

	// Process the metric results
	for _, result := range data.MetricResults {
		if result.Dimensions == nil {
			continue
		}

		queueID, ok := result.Dimensions["QUEUE"]
		if !ok {
			continue
		}

		queue, exists := queueMap[queueID]
		if !exists {
			logrus.WithField("queue_id", queueID).Debug("Queue not found in map, skipping")
			continue
		}

		channel := ""
		if channelValue, hasChannel := result.Dimensions["CHANNEL"]; hasChannel {
			channel = channelValue
		}

		// Process each metric collection
		for _, md := range result.Collections {
			if md.Metric == nil || md.Value == nil {
				continue
			}

			awsMetricName := aws.ToString(md.Metric.Name)
			// Find the specific metric definition that matches the filters
			def, exists := s.findMetricDefinition(awsMetricName, md.Metric.MetricFilters, col.Metrics)
			if !exists {
				logrus.WithField("metric_name", awsMetricName).Debug("Metric not in configuration, skipping")
				continue
			}

			// Build base labels
			labels := []string{
				col.Name,           // collector name
				def.PrometheusName, // metric name
				queue.ID,
				queue.Name,
				queue.Type,
				channel,
				cfg.AWS.Region,
			}

			// Add direction label if configured
			if direction, hasDirection := def.Labels["direction"]; hasDirection {
				labels = append(labels, direction)
			} else {
				labels = append(labels, "all")
			}

			// Create the metric
			metrics = append(metrics, maas.NewMetric(
				"generic_historical_metric",
				prometheus.GaugeValue,
				*md.Value,
				labels,
			))

			logrus.WithFields(logrus.Fields{
				"collector":    col.Name,
				"metric_name":  def.PrometheusName,
				"metric_value": *md.Value,
				"queue_name":   queue.Name,
			}).Debug("Collected generic historical metric")
		}
	}

	return metrics, nil
}

func (s *GenericHistoricalScraper) findMetricDefinition(awsMetricName string, filters []types.MetricFilterV2, definitions []connectors.MetricDefinition) (connectors.MetricDefinition, bool) {
	for _, def := range definitions {
		if def.AWSName == awsMetricName {
			if len(def.MetricFilters) == 0 && len(filters) == 0 {
				return def, true
			}
			if len(def.MetricFilters) > 0 && len(filters) > 0 && def.MetricFilters[0].Key == *filters[0].MetricFilterKey && def.MetricFilters[0].Values[0] == filters[0].MetricFilterValues[0] {
				return def, true
			}
		}
	}
	return connectors.MetricDefinition{}, false
}

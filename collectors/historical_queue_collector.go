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

const historicalQueueCollectorVersion = "1.0.0"

// NewHistoricalQueueCollector creates a scheduled scraper for historical queue metrics
func NewHistoricalQueueCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.HistoricalQueue.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("historical_queue", NewHistoricalScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "contacts_queued_today", "Number of contacts queued since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
		maas.WithDescription(app, "contacts_abandoned_today", "Number of contacts abandoned since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
		maas.WithDescription(app, "contacts_handled_today", "Number of contacts handled since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "channel", "direction", "region"}),
	)
}

type HistoricalScraper struct{}

func NewHistoricalScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *HistoricalScraper {
	return &HistoricalScraper{}
}

func (s *HistoricalScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: %T", c)
	}

	config := conn.GetConfig()

	queues := conn.GetQueueManager().GetQueues()

	queueMap := make(map[string]connectors.Queue)
	for _, q := range queues {
		queueMap[q.ID] = q
	}

	metricsToFetch := []types.MetricV2{
		{Name: aws.String("CONTACTS_QUEUED")},
		{Name: aws.String("CONTACTS_ABANDONED")},
		{
			Name: aws.String("CONTACTS_HANDLED"),
			MetricFilters: []types.MetricFilterV2{
				{
					MetricFilterKey:    aws.String("INITIATION_METHOD"),
					MetricFilterValues: []string{"INBOUND"},
				},
			},
		},
		{
			Name: aws.String("CONTACTS_HANDLED"),
			MetricFilters: []types.MetricFilterV2{
				{
					MetricFilterKey:    aws.String("INITIATION_METHOD"),
					MetricFilterValues: []string{"OUTBOUND", "CALLBACK", "API"},
				},
			},
		},
	}

	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	allMetrics := make([]maas.Metric, 0)

	for i := 0; i < len(queues); i += maxQueuesPerRequest {
		end := i + maxQueuesPerRequest
		if end > len(queues) {
			end = len(queues)
		}
		batch := queues[i:end]
		batchIDs := make([]string, len(batch))
		for j, q := range batch {
			batchIDs[j] = q.ID
		}

		input := &connect.GetMetricDataV2Input{
			ResourceArn: &config.AWS.InstanceARN,
			StartTime:   &midnight,
			EndTime:     &now,
			Filters: []types.FilterV2{
				{FilterKey: aws.String("QUEUE"), FilterValues: batchIDs},
				{FilterKey: aws.String("CHANNEL"), FilterValues: []string{"VOICE", "CHAT", "TASK"}},
			},
			Groupings: []string{"QUEUE", "CHANNEL"},
			Metrics:   metricsToFetch,
			Interval: &types.IntervalDetails{
				IntervalPeriod: types.IntervalPeriodTotal,
				TimeZone:       aws.String("UTC"),
			},
		}

		data, err := conn.GetMetricDataV2(context.Background(), input)
		if err != nil {
			logrus.WithError(err).Error("Failed to get historical metric data for batch")
			continue
		}

		for _, result := range data.MetricResults {
			if result.Dimensions == nil {
				continue
			}
			queueID, ok := result.Dimensions["QUEUE"]
			if !ok {
				continue
			}
			channel, ok := result.Dimensions["CHANNEL"]
			if !ok {
				continue
			}

			queue, exists := queueMap[queueID]
			if !exists {
				continue
			}

			labels := []string{queue.ID, queue.Name, queue.Type, channel}

			for _, md := range result.Collections {
				if md.Metric == nil || md.Value == nil {
					continue
				}
				var promName string
				extraLabels := labels
				switch *md.Metric.Name {
				case "CONTACTS_QUEUED":
					promName = "contacts_queued_today"
				case "CONTACTS_ABANDONED":
					promName = "contacts_abandoned_today"
				case "CONTACTS_HANDLED":
					promName = "contacts_handled_today"
					// Determine direction from metric filters
					direction := "unknown"
					if md.Metric.MetricFilters != nil {
						for _, f := range md.Metric.MetricFilters {
							if f.MetricFilterKey != nil && aws.ToString(f.MetricFilterKey) == "INITIATION_METHOD" && len(f.MetricFilterValues) > 0 {
								if len(f.MetricFilterValues) == 1 && f.MetricFilterValues[0] == "INBOUND" {
									direction = "inbound"
								} else {
									direction = "outbound"
								}
							}
						}
					}
					extraLabels = append(extraLabels, direction)
				default:
					continue
				}

				extraLabels = append(extraLabels, config.AWS.Region)
				allMetrics = append(allMetrics, maas.NewMetric(promName, prometheus.GaugeValue, *md.Value, extraLabels))
			}
		}
	}

	return allMetrics, nil
}

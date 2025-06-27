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

// historicalAvgMetricScraper fetches a single average time metric grouped by queue and channel
// metricName is the AWS metric name (e.g., AVG_HANDLE_TIME)
// promName is the exported Prometheus metric name

type historicalAvgMetricScraper struct {
	metricName string
	promName   string
}

func newHistoricalAvgMetricScraper(metricName, promName string) *historicalAvgMetricScraper {
	return &historicalAvgMetricScraper{metricName: metricName, promName: promName}
}

func (s *historicalAvgMetricScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
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

	metricsToFetch := []types.MetricV2{{Name: aws.String(s.metricName)}}

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
			labels := []string{queue.ID, queue.Name, queue.Type, channel, config.AWS.Region}
			for _, md := range result.Collections {
				if md.Metric == nil || md.Value == nil {
					continue
				}
				if aws.ToString(md.Metric.Name) == s.metricName {
					allMetrics = append(allMetrics, maas.NewMetric(s.promName, prometheus.GaugeValue, *md.Value, labels))
				}
			}
		}
	}

	return allMetrics, nil
}

// NewHistoricalAHTCollector collects AVG_HANDLE_TIME metrics grouped by queue and channel
func NewHistoricalAHTCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.HistoricalAHT.Interval) * time.Second
	timeout := 30 * time.Second
	scraper := newHistoricalAvgMetricScraper("AVG_HANDLE_TIME", "connect_historical_avg_handle_time_seconds")
	return maas.NewScheduledScraper("historical_aht", scraper,
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "connect_historical_avg_handle_time_seconds", "Average handle time since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
	)
}

// NewHistoricalACWCollector collects AVG_AFTER_CONTACT_WORK_TIME metrics grouped by queue and channel
func NewHistoricalACWCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.HistoricalACW.Interval) * time.Second
	timeout := 30 * time.Second
	scraper := newHistoricalAvgMetricScraper("AVG_AFTER_CONTACT_WORK_TIME", "connect_historical_avg_after_contact_work_time_seconds")
	return maas.NewScheduledScraper("historical_acw", scraper,
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
		maas.WithDescription(app, "connect_historical_avg_after_contact_work_time_seconds", "Average after contact work time since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "channel", "region"}),
	)
}

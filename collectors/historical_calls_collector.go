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


// NewHistoricalCallsCollector creates a scheduled scraper for historical call metrics
func NewHistoricalCallsCollector(app *kingpin.Application, config connectors.AWSConnectConfig) *maas.ScheduledScraper {
	interval := time.Duration(config.Collectors.HistoricalCalls.Interval) * time.Second
	timeout := 30 * time.Second

	return maas.NewScheduledScraper("historical_calls", NewHistoricalCallsScraper(app, config),
		maas.WithSchedule(maas.NewSchedule(
			maas.WithFrequency(interval),
			maas.WithTimeout(timeout),
		)),
               maas.WithDescription(app, "connect_historical_contacts_handled_total", "Total contacts handled since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "direction", "region"}),
               maas.WithDescription(app, "connect_historical_contacts_abandoned_total", "Total contacts abandoned since midnight UTC", []string{"queue_id", "queue_name", "queue_type", "region"}),
	)
}

type HistoricalCallsScraper struct{}

func NewHistoricalCallsScraper(app *kingpin.Application, config connectors.AWSConnectConfig) *HistoricalCallsScraper {
	return &HistoricalCallsScraper{}
}

func (s *HistoricalCallsScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
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
		{Name: aws.String("CONTACTS_HANDLED")},
		{Name: aws.String("CONTACTS_HANDLED_OUTBOUND")},
		{Name: aws.String("CONTACTS_ABANDONED")},
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
			},
			Groupings: []string{"QUEUE"},
			Metrics:   metricsToFetch,
			Interval: &types.IntervalDetails{
				IntervalPeriod: types.IntervalPeriodTotal,
				TimeZone:       aws.String("UTC"),
			},
		}

		data, err := conn.GetMetricDataV2(context.Background(), input)
		if err != nil {
			logrus.WithError(err).Error("Failed to get historical call metric data for batch")
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

			queue, exists := queueMap[queueID]
			if !exists {
				continue
			}

			base := []string{queue.ID, queue.Name, queue.Type}

			for _, md := range result.Collections {
				if md.Metric == nil || md.Value == nil {
					continue
				}
                                switch aws.ToString(md.Metric.Name) {
                                case "CONTACTS_HANDLED":
                                        labels := append(base, "inbound", config.AWS.Region)
                                        allMetrics = append(allMetrics, maas.NewMetric("connect_historical_contacts_handled_total", prometheus.CounterValue, *md.Value, labels))
                                case "CONTACTS_HANDLED_OUTBOUND":
                                        labels := append(base, "outbound", config.AWS.Region)
                                        allMetrics = append(allMetrics, maas.NewMetric("connect_historical_contacts_handled_total", prometheus.CounterValue, *md.Value, labels))
                                case "CONTACTS_ABANDONED":
                                        labels := append(base, config.AWS.Region)
                                        allMetrics = append(allMetrics, maas.NewMetric("connect_historical_contacts_abandoned_total", prometheus.CounterValue, *md.Value, labels))
				}
			}
		}
	}

	return allMetrics, nil
}

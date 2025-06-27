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
		maas.WithDescription(app, "generic_metric", "Generic realtime metric", []string{"collector", "metric_name", "queue_id", "queue_name", "queue_type", "channel", "region"}),
	)
}

// GenericRealtimeScraper gathers realtime metrics defined in configuration
type GenericRealtimeScraper struct{}

// NewGenericRealtimeScraper constructs the scraper
func NewGenericRealtimeScraper() *GenericRealtimeScraper { return &GenericRealtimeScraper{} }

// Scrape implements the Scraper interface
func (s *GenericRealtimeScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	conn, ok := c.(*connectors.AWSConnectConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type: %T", c)
	}

	cfg := conn.GetConfig()
	rt := cfg.Collectors.GenericRealtime
	if !rt.Enabled {
		return []maas.Metric{}, nil
	}

	queues := conn.GetQueueManager().GetQueues()
	queueIDs := make([]string, len(queues))
	queueMap := make(map[string]connectors.Queue)
	for i, q := range queues {
		queueIDs[i] = q.ID
		queueMap[q.ID] = q
	}

	now := time.Now().UTC()
	start := now.Add(-1 * time.Minute)

	metrics := make([]maas.Metric, 0)

	for _, col := range rt.Collectors {
		metricDefs := make([]types.MetricV2, len(col.Metrics))
		for i, m := range col.Metrics {
			metricDefs[i] = types.MetricV2{Name: aws.String(m)}
		}

		filters := []types.FilterV2{{FilterKey: aws.String("QUEUE"), FilterValues: queueIDs}}
		for _, f := range col.Filters {
			filters = append(filters, types.FilterV2{FilterKey: aws.String(strings.ToUpper(f.Key)), FilterValues: f.Values})
		}

		input := &connect.GetMetricDataV2Input{
			ResourceArn: &cfg.AWS.InstanceARN,
			StartTime:   &start,
			EndTime:     &now,
			Filters:     filters,
			Groupings:   col.Groupings,
			Metrics:     metricDefs,
		}

		data, err := conn.GetMetricDataV2(context.Background(), input)
		if err != nil {
			logrus.WithError(err).Error("Failed to get generic realtime data")
			continue
		}

		for _, res := range data.MetricResults {
			qid := res.Dimensions["QUEUE"]
			queue, ok := queueMap[qid]
			if !ok {
				continue
			}
			channel := res.Dimensions["CHANNEL"]
			labelsBase := []string{col.Name, "", queue.ID, queue.Name, queue.Type, channel, cfg.AWS.Region}
			for _, md := range res.Collections {
				if md.Metric == nil || md.Value == nil {
					continue
				}
				labels := append([]string{}, labelsBase...)
				labels[1] = aws.ToString(md.Metric.Name)
				metrics = append(metrics, maas.NewMetric("generic_metric", prometheus.GaugeValue, *md.Value, labels))
			}
		}
	}

	return metrics, nil
}

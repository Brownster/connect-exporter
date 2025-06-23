package maas

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/suite"
)

type MetricsTestSuite struct {
	suite.Suite
	*Metrics
}

func (s *MetricsTestSuite) SetupTest() {
	s.Metrics = NewMetrics()
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
}

func (s *MetricsTestSuite) TestStore() {
	const metricValue = 1
	// const expectedLength = 1

	s.Put([]prometheus.Metric{
		prometheus.MustNewConstMetric(
			prometheus.NewDesc("a", "b", nil, nil),
			prometheus.GaugeValue, metricValue,
		)})
	s.Len(s.metrics, 1)

	s.Put([]prometheus.Metric{
		prometheus.MustNewConstMetric(
			prometheus.NewDesc("a", "b", nil, nil),
			prometheus.CounterValue, metricValue,
		)})
	s.Len(s.metrics, 1)
}

func TestMetricsTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}

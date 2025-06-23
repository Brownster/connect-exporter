package maas_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	maas "github.com/mbrown007/monitoring-rss-exporter/monitoring-maas"
	"github.com/stretchr/testify/suite"
	"github.com/alecthomas/kingpin/v2"
)

type MaaSTestSuite struct {
	suite.Suite
	*kingpin.Application
	*maas.ScheduledScraper
	maas.Connector
}

type DiskScraper struct {
}

func (s *DiskScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
	return []maas.Metric{maas.NewMetric(
		"io_time_seconds_total",
		prometheus.CounterValue,
		82126.158,
		[]string{"dm-0"},
	)}, nil
}

func (s *MaaSTestSuite) SetupTest() {
	s.Application = kingpin.New("node", "node_exporter")
	s.ScheduledScraper = maas.NewScheduledScraper(
		"disk",
		&DiskScraper{},
		maas.WithDescription(s.Application, "io_time_seconds_total", "Total seconds spent doing I/Os", []string{"device"}),
	)
	s.Connector = &maas.SuccessConnector{}
}

func (s *MaaSTestSuite) TestCollect() {
	e, err := maas.NewExporter(s.Application, s.Connector,
		maas.WithLabels(&maas.MockLabels{}),
		maas.WithArgs([]string{
			"--web.listen-port=9100",
		}),
		maas.WithScheduler(maas.NewMockScheduler()),
		maas.WithScheduledScrapers(s.ScheduledScraper),
	)
	s.NoError(err)
	e.Start()

	err = testutil.CollectAndCompare(e, strings.NewReader(`
		# HELP node_disk_io_time_seconds_total Total seconds spent doing I/Os
		# TYPE node_disk_io_time_seconds_total counter
		node_disk_io_time_seconds_total{device="dm-0"} 82126.158
	`), "node_disk_io_time_seconds_total")

	s.NoError(err)
}

func TestMaaSTestSuite(t *testing.T) {
	suite.Run(t, new(MaaSTestSuite))
}

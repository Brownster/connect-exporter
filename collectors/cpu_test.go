package collectors

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
	"github.com/stretchr/testify/suite"
)

// CPUTestSuite demonstrates CPU collector testing following create_template_repo.md
type CPUTestSuite struct {
	suite.Suite
	*connectors.MockConnector
	*maas.Exporter
}

func (s *CPUTestSuite) SetupTest() {
	// Use mock connector for testing
	s.MockConnector = connectors.NewMockConnector()
	s.MockConnector.AddResponse("cpu_info", "cpu_usage: 5.2")
	
	// Create exporter with CPU collector
	exporter, err := NewConnectExporter(s.MockConnector,
		maas.WithArgs([]string{"--web.listen-port=9100"}),
		maas.WithLabels(&maas.MockLabels{}))
	s.NoError(err)
	s.Exporter = exporter
}

func (s *CPUTestSuite) TestCPUUsagePercent() {
	s.Exporter.Start()

	// Test CPU usage metric following create_template_repo.md pattern
	err := testutil.CollectAndCompare(s.Exporter, strings.NewReader(`
		# HELP connect_cpu_usage_percent CPU utilization
		# TYPE connect_cpu_usage_percent gauge
		connect_cpu_usage_percent{id="0"} 5.2
	`), "connect_cpu_usage_percent")

	s.NoError(err)
}

func TestCPUTestSuite(t *testing.T) {
	suite.Run(t, new(CPUTestSuite))
}
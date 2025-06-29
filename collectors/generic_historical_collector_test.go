// Package collectors provides the metric collectors for the AWS Connect exporter.
//
// This file contains the unit tests for the generic_historical collector. The tests
// use a mock HTTP server to simulate the AWS Connect API, which allows for
// controlled testing of the collector's logic without making actual API calls.
package collectors

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// mockGenericHistoricalServer is a mock HTTP server that simulates the AWS Connect
// API for the generic_historical collector.
type mockGenericHistoricalServer struct{ groupings []string }

// ServeHTTP is the handler for the mock HTTP server. It responds to the API
// requests made by the collector with mock data.
func (m *mockGenericHistoricalServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/queues-summary/") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"QueueSummaryList":[{"Id":"q-1","Name":"Sales_Queue_VOICE","QueueType":"STANDARD"}]}`)
		return
	}
	// Handle ListUsers endpoint for AgentManager
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/users-summary") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"UserSummaryList":[{"Id":"agent-1","Username":"test.agent","Arn":"arn:aws:connect:us-east-1:123456789012:instance/test/agent/agent-1"}]}`)
		return
	}
	// Handle GetMetricDataV2 (historical API)
	if r.Method == http.MethodPost && (strings.HasPrefix(r.URL.Path, "/metrics/data") || strings.HasPrefix(r.URL.Path, "/metrics/historical/")) {
		body, _ := io.ReadAll(r.Body)
		var req struct{ Groupings []string }
		json.Unmarshal(body, &req)
		m.groupings = req.Groupings
		w.Header().Set("Content-Type", "application/json")
		// Return historical metric data format
		io.WriteString(w, `{"MetricResults":[{"Dimensions":{"QUEUE":"q-1","CHANNEL":"VOICE"},"Collections":[{"Metric":{"Name":"CONTACTS_HANDLED"},"Value":10.0}]}]}`)
		return
	}
	http.NotFound(w, r)
}

// setupGenericHistoricalConnector is a helper function that sets up a mock AWS
// Connect connector for testing.
func setupGenericHistoricalConnector(t *testing.T, srv *httptest.Server) *connectors.AWSConnectConnector {
	t.Setenv("ENV", "development")
	t.Setenv("AWS_CONNECT_MOCK_ENDPOINT", srv.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("AWS_REGION", "eu-west-2")

	app := kingpin.New("app", "test")
	conn, err := connectors.NewConfigurableAWSConnectConnector(app)
	if err != nil {
		t.Fatal(err)
	}

	cfgData := []byte(`aws:
  region: eu-west-2
  instance_arn: arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772
collectors:
  generic_historical:
    enabled: true
    interval: 300
    collectors:
      - name: historical_queue
        metrics:
          - aws_name: CONTACTS_HANDLED
            prometheus_name: connect_historical_contacts_handled_total
            labels: {}
        groupings: [QUEUE, CHANNEL]
        filters:
          - key: CHANNEL
            values: [VOICE, CHAT, TASK]
`)

	tmpFile, err := os.CreateTemp("", "cfg*.yml")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	if _, err := tmpFile.Write(cfgData); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	inst := "arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772"
	_, err = app.Parse([]string{"--instance-arn=" + inst, "--region=eu-west-2", "--config.file=" + tmpFile.Name()})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	conn.GetQueueManager().Stop()
	return conn
}

// TestGenericHistoricalCollector tests the basic functionality of the
// generic_historical collector.
func TestGenericHistoricalCollector(t *testing.T) {
	ms := &mockGenericHistoricalServer{}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupGenericHistoricalConnector(t, srv)

	scraper := NewGenericHistoricalScraper()
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Equal(t, []string{"QUEUE", "CHANNEL"}, ms.groupings)
	found := false
	for _, m := range metrics {
		if metricName(m) == "generic_historical_metric" {
			assert.Equal(t, prometheus.GaugeValue, metricValueType(m))
			labels := metricLabels(m)
			assert.Equal(t, "historical_queue", labels[0])                          // collector name
			assert.Equal(t, "connect_historical_contacts_handled_total", labels[1]) // metric name
			found = true
		}
	}
	assert.True(t, found)
}

// TestGenericHistoricalCollectorError tests that the generic_historical collector
// handles errors from the AWS Connect API gracefully.
func TestGenericHistoricalCollectorError(t *testing.T) {
	ms := &mockGenericHistoricalServer{}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupGenericHistoricalConnector(t, srv)

	// Force error by shutting down server
	srv.Close()

	scraper := NewGenericHistoricalScraper()
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	metrics, err := scraper.Scrape(conn)
	assert.NoError(t, err)
	assert.Empty(t, metrics)
	assert.Contains(t, buf.String(), "Failed to get historical metric data")
}

// TestGenericHistoricalCollectorDisabled tests that the generic_historical
// collector does not scrape any metrics when it is disabled in the configuration.
func TestGenericHistoricalCollectorDisabled(t *testing.T) {
	ms := &mockGenericHistoricalServer{}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	t.Setenv("ENV", "development")
	t.Setenv("AWS_CONNECT_MOCK_ENDPOINT", srv.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("AWS_REGION", "eu-west-2")

	app := kingpin.New("app", "test")
	conn, err := connectors.NewConfigurableAWSConnectConnector(app)
	if err != nil {
		t.Fatal(err)
	}

	// Config with disabled collector
	cfgData := []byte(`aws:
  region: eu-west-2
  instance_arn: arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772
collectors:
  generic_historical:
    enabled: false
    interval: 300
    collectors: []
`)

	tmpFile, err := os.CreateTemp("", "cfg*.yml")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	if _, err := tmpFile.Write(cfgData); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	inst := "arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772"
	_, err = app.Parse([]string{"--instance-arn=" + inst, "--region=eu-west-2", "--config.file=" + tmpFile.Name()})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	conn.GetQueueManager().Stop()

	scraper := NewGenericHistoricalScraper()
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.Empty(t, metrics)
}

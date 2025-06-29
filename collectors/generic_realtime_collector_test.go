// Package collectors provides the metric collectors for the AWS Connect exporter.
//
// This file contains the unit tests for the generic_realtime collector. The tests
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

// mockGenericServer is a mock HTTP server that simulates the AWS Connect API for
// the generic_realtime collector.
type mockGenericServer struct{ groupings []string }

// ServeHTTP is the handler for the mock HTTP server. It responds to the API
// requests made by the collector with mock data.
func (m *mockGenericServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/queues-summary/") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"QueueSummaryList":[{"Id":"q-1","Name":"Sales_Queue_VOICE","QueueType":"STANDARD"}]}`)
		return
	}
	// Handle ListUsers endpoint for AgentManager
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/users-summary/") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"UserSummaryList":[{"Id":"agent-1","Username":"test.agent","Arn":"arn:aws:connect:us-east-1:123456789012:instance/test/agent/agent-1"}]}`)
		return
	}
	// Handle GetCurrentMetricData (realtime API)
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/metrics/current/") {
		body, _ := io.ReadAll(r.Body)
		var req struct{ Groupings []string }
		json.Unmarshal(body, &req)
		m.groupings = req.Groupings
		w.Header().Set("Content-Type", "application/json")
		// Return current metric data format
		io.WriteString(w, `{"MetricResults":[{"Dimensions":{"Queue":{"Id":"q-1"},"Channel":"VOICE"},"Collections":[{"Metric":{"Name":"AGENTS_AVAILABLE"},"Value":5.0}]}]}`)
		return
	}
	http.NotFound(w, r)
}

// setupGenericConnector is a helper function that sets up a mock AWS Connect
// connector for testing.
func setupGenericConnector(t *testing.T, srv *httptest.Server) *connectors.AWSConnectConnector {
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

	cfgData := []byte("aws:\n  region: eu-west-2\n  instance_arn: arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772\ncollectors:\n  generic_realtime:\n    enabled: true\n    interval: 30\n    collectors:\n      - name: queue_realtime_metrics\n        metrics:\n          - aws_name: AGENTS_AVAILABLE\n            prometheus_name: connect_queue_agents\n            labels:\n              state: Available\n          - aws_name: CONTACTS_IN_QUEUE\n            prometheus_name: connect_queue_contacts_in_queue\n            labels: {}\n        groupings: [QUEUE, CHANNEL]\n        filters: []\n")
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

// TestGenericRealtimeCollector tests the basic functionality of the
// generic_realtime collector.
func TestGenericRealtimeCollector(t *testing.T) {
	ms := &mockGenericServer{}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupGenericConnector(t, srv)

	scraper := NewGenericRealtimeScraper()
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Equal(t, []string{"QUEUE", "CHANNEL"}, ms.groupings)
	found := false
	for _, m := range metrics {
		if metricName(m) == "generic_metric" {
			assert.Equal(t, prometheus.GaugeValue, metricValueType(m))
			// Check if this is the agents metric
			if metricLabels(m)[1] == "connect_queue_agents" {
				assert.Equal(t, "Available", metricLabels(m)[7]) // state label
				found = true
			}
		}
	}
	assert.True(t, found)
}

// TestGenericRealtimeCollectorError tests that the generic_realtime collector
// handles errors from the AWS Connect API gracefully.
func TestGenericRealtimeCollectorError(t *testing.T) {
	ms := &mockGenericServer{}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupGenericConnector(t, srv)

	// Force error by shutting down server
	srv.Close()

	scraper := NewGenericRealtimeScraper()
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	metrics, err := scraper.Scrape(conn)
	assert.NoError(t, err)
	assert.Empty(t, metrics)
	assert.Contains(t, buf.String(), "Failed to get current metric data")
}

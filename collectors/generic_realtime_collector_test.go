package collectors

import (
	"bytes"
	"encoding/json"
	"io"
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

type mockGenericServer struct{ groupings []string }

func (m *mockGenericServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/queues-summary/") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"QueueSummaryList":[{"Id":"q-1","Name":"Sales_Queue_VOICE","QueueType":"STANDARD"}]}`) //nolint:errcheck
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/metrics/data" {
		body, _ := io.ReadAll(r.Body)
		var req struct{ Groupings []string }
		json.Unmarshal(body, &req) //nolint:errcheck
		m.groupings = req.Groupings
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"MetricResults":[{"Dimensions":{"QUEUE":"q-1","CHANNEL":"VOICE"},"Collections":[{"Metric":{"Name":"AGENTS_AVAILABLE"},"Value":5.0}]}]}`) //nolint:errcheck
		return
	}
	http.NotFound(w, r)
}

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

	cfgData := []byte("aws:\n  region: eu-west-2\n  instance_arn: arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772\ncollectors:\n  generic_realtime:\n    enabled: true\n    interval: 30\n    collectors:\n      - name: agents_available_by_queue\n        metrics: [AGENTS_AVAILABLE]\n        groupings: [QUEUE, CHANNEL]\n")
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
			assert.Equal(t, "AGENTS_AVAILABLE", metricLabels(m)[1])
			found = true
		}
	}
	assert.True(t, found)
}

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
	assert.Contains(t, buf.String(), "Failed to get generic realtime data")
}

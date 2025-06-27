package collectors

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockServerTime struct {
	groupings []string
	metric    string
	returnErr bool
}

func (m *mockServerTime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.returnErr && r.URL.Path == "/metrics/data" {
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}
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
		io.WriteString(w, `{"MetricResults":[{"Dimensions":{"QUEUE":"q-1","CHANNEL":"VOICE"},"Collections":[{"Metric":{"Name":"`+m.metric+`"},"Value":42.0}]}]}`) //nolint:errcheck
		return
	}
	http.NotFound(w, r)
}

func setupTimeConnector(t *testing.T, srv *httptest.Server) *connectors.AWSConnectConnector {
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
	inst := "arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772"
	_, err = app.Parse([]string{"--instance-arn=" + inst, "--region=eu-west-2", "--config.file=/dev/null"})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	conn.GetQueueManager().Stop()
	return conn
}

func TestHistoricalAHTCollector(t *testing.T) {
	ms := &mockServerTime{metric: "AVG_HANDLE_TIME"}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupTimeConnector(t, srv)

	scraper := newHistoricalAvgMetricScraper("AVG_HANDLE_TIME", "connect_historical_avg_handle_time_seconds")
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Equal(t, []string{"QUEUE", "CHANNEL"}, ms.groupings)
	found := false
	for _, m := range metrics {
		if metricName(m) == "connect_historical_avg_handle_time_seconds" {
			assert.Equal(t, prometheus.GaugeValue, metricValueType(m))
			assert.Len(t, metricLabels(m), 5)
			found = true
		}
	}
	assert.True(t, found)
}

func TestHistoricalACWCollector(t *testing.T) {
	ms := &mockServerTime{metric: "AVG_AFTER_CONTACT_WORK_TIME"}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupTimeConnector(t, srv)

	scraper := newHistoricalAvgMetricScraper("AVG_AFTER_CONTACT_WORK_TIME", "connect_historical_avg_after_contact_work_time_seconds")
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Equal(t, []string{"QUEUE", "CHANNEL"}, ms.groupings)
	found := false
	for _, m := range metrics {
		if metricName(m) == "connect_historical_avg_after_contact_work_time_seconds" {
			assert.Equal(t, prometheus.GaugeValue, metricValueType(m))
			assert.Len(t, metricLabels(m), 5)
			found = true
		}
	}
	assert.True(t, found)
}

func TestHistoricalTimeCollectorError(t *testing.T) {
	ms := &mockServerTime{returnErr: true}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupTimeConnector(t, srv)

	scraper := newHistoricalAvgMetricScraper("AVG_HANDLE_TIME", "connect_historical_avg_handle_time_seconds")
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.Empty(t, metrics)
	assert.Contains(t, buf.String(), "Failed to get historical metric data")
}

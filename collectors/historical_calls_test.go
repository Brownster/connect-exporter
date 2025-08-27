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

type mockServerCalls struct {
	groupings []string
	returnErr bool
}

func (m *mockServerCalls) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
               io.WriteString(w, `{"MetricResults":[{"Dimensions":{"QUEUE":"q-1"},"Collections":[{"Metric":{"Name":"CONTACTS_HANDLED"},"Value":100.0}]}]}`) //nolint:errcheck
		return
	}
	http.NotFound(w, r)
}

func setupCallsConnector(t *testing.T, srv *httptest.Server) *connectors.AWSConnectConnector {
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

func TestHistoricalCallsCollector(t *testing.T) {
	ms := &mockServerCalls{}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupCallsConnector(t, srv)

	scraper := NewHistoricalCallsScraper(kingpin.New("app", "test"), conn.GetConfig())
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Equal(t, []string{"QUEUE"}, ms.groupings)
	assert.Equal(t, prometheus.CounterValue, metricValueType(metrics[0]))
	assert.GreaterOrEqual(t, len(metricLabels(metrics[0])), 4)
}

func TestHistoricalCallsCollectorError(t *testing.T) {
	ms := &mockServerCalls{returnErr: true}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupCallsConnector(t, srv)

	scraper := NewHistoricalCallsScraper(kingpin.New("app", "test"), conn.GetConfig())
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.Empty(t, metrics)
	assert.Contains(t, buf.String(), "Failed to get historical call metric data")
}

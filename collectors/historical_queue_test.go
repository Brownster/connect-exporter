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

type mockServer struct {
	t         *testing.T
	groupings []string
	returnErr bool
}

func (m *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.returnErr && r.URL.Path == "/metrics/data" {
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/queues-summary/") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"QueueSummaryList":[{"Id":"q-1","Name":"Sales_Queue_VOICE","QueueType":"STANDARD"}]}`)
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/metrics/data" {
		body, _ := io.ReadAll(r.Body)
		var req struct{ Groupings []string }
		json.Unmarshal(body, &req)
		m.groupings = req.Groupings
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"MetricResults":[{"Dimensions":{"QUEUE":"q-1","CHANNEL":"VOICE"},"Collections":[{"Metric":{"Name":"CONTACTS_HANDLED"},"Value":150.0}]}]}`)
		return
	}
	http.NotFound(w, r)
}

func setupConnector(t *testing.T, srv *httptest.Server) *connectors.AWSConnectConnector {
	os.Setenv("ENV", "development")
	os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", srv.URL)
	os.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	os.Setenv("AWS_REGION", "eu-west-2")

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

func TestHistoricalQueueCollector(t *testing.T) {
	ms := &mockServer{}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupConnector(t, srv)

	scraper := NewHistoricalScraper(kingpin.New("app", "test"), conn.GetConfig())
	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Equal(t, []string{"QUEUE", "CHANNEL"}, ms.groupings)
	found := false
	for _, m := range metrics {
		if metricName(m) == "connect_historical_handled_total" {
			assert.Equal(t, prometheus.CounterValue, metricValueType(m))
			assert.Len(t, metricLabels(m), 3)
			found = true
		}
	}
	assert.True(t, found)
}

func TestHistoricalQueueCollectorError(t *testing.T) {
	ms := &mockServer{returnErr: true}
	srv := httptest.NewServer(ms)
	defer srv.Close()
	conn := setupConnector(t, srv)

	scraper := NewHistoricalScraper(kingpin.New("app", "test"), conn.GetConfig())
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	metrics, err := scraper.Scrape(conn)

	assert.NoError(t, err)
	assert.Empty(t, metrics)
	assert.Contains(t, buf.String(), "Failed to get historical metric data")
}

package integration_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func waitForURL(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func TestExporterWithMockAPI(t *testing.T) {
	mockDir := filepath.Join("..", "docs", "tools", "aws-mock-api-main")
	if _, err := os.Stat(filepath.Join(mockDir, "node_modules")); err != nil {
		npmCmd := exec.Command("npm", "install")
		npmCmd.Dir = mockDir
		if out, err := npmCmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to install mock API deps: %v\n%s", err, string(out))
		}
	}
	mockPath := filepath.Join(mockDir, "server.js")
	mockCmd := exec.Command("node", mockPath)
	if err := mockCmd.Start(); err != nil {
		t.Fatalf("failed to start mock API: %v", err)
	}
	defer func() { _ = mockCmd.Process.Kill() }()

	if err := waitForURL("http://localhost:3000/health", 10*time.Second); err != nil {
		t.Fatalf("mock API not ready: %v", err)
	}

	exporterPath := filepath.Join("..", "cmd", "connect-exporter_exporter")
	exporterCmd := exec.Command("go", "run", exporterPath,
		"--web.listen-port=9100",
		"--config.file=../config.development.yml",
		"--no-advertise",
		"--labels.fqdn=localhost",
		"--labels.location=test",
		"--labels.country=US",
		"--labels.ip-address=127.0.0.1",
	)
	exporterCmd.Env = append(os.Environ(),
		"ENV=development",
		"AWS_ACCESS_KEY_ID=dummy",
		"AWS_SECRET_ACCESS_KEY=dummy",
		"AWS_REGION=us-east-1",
	)
	var out bytes.Buffer
	exporterCmd.Stdout = &out
	exporterCmd.Stderr = &out
	if err := exporterCmd.Start(); err != nil {
		t.Fatalf("failed to start exporter: %v", err)
	}
	defer func() { _ = exporterCmd.Process.Kill() }()

	if err := waitForURL("http://localhost:9100/metrics", 20*time.Second); err != nil {
		t.Fatalf("exporter not ready: %v\nOutput: %s", err, out.String())
	}

	resp, err := http.Get("http://localhost:9100/metrics")
	if err != nil {
		t.Fatalf("failed to fetch metrics: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	metrics := string(body)

	// Debug: Print some metrics to see what's available
	t.Logf("Metrics contains aws_connect_exporter: %v", strings.Contains(metrics, "aws_connect_exporter"))
	if !strings.Contains(metrics, "aws_connect_exporter_agent_realtime_status_connect_agent_status_state") {
		t.Errorf("missing agent_state metrics")
	}
	if !strings.Contains(metrics, "agent_id=\"6aed748d-3a99-410c-9e3a-afe3fda0af82\"") {
		t.Logf("Metrics preview: %s", metrics[:min(500, len(metrics))])
		t.Errorf("expected agent metric for alice.smith")
	}

	// Verify historical queue metrics are exposed (now via generic collector)
	if !strings.Contains(metrics, "aws_connect_exporter_generic_historical_generic_historical_metric") {
		t.Errorf("missing generic historical metric")
	}
	if !strings.Contains(metrics, "connect_historical_contacts_abandoned_total") {
		t.Errorf("missing historical queue abandoned metric")
	}
	if !strings.Contains(metrics, "connect_historical_contacts_handled_total") {
		t.Errorf("missing historical queue handled metric")
	}

	// Verify queue discovery found expected queues
	if !strings.Contains(metrics, "queue_id=\"queue-sales-01\"") || !strings.Contains(metrics, "queue_name=\"Sales Queue\"") {
		t.Errorf("Sales Queue metrics not found, queue discovery may have failed")
	}
}

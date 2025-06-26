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
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
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
	defer mockCmd.Process.Kill()

	// Wait a moment for the mock API to start
	time.Sleep(2 * time.Second)

	exporterPath := filepath.Join("..", "cmd", "connect-exporter_exporter")
	exporterCmd := exec.Command("go", "run", exporterPath,
		"--web.listen-port=9100",
		"--config.file=../config.development.yml",
		"--no-advertise",
		"--no-historical_calls.enabled",
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
	defer exporterCmd.Process.Kill()

	if err := waitForURL("http://localhost:9100/metrics", 20*time.Second); err != nil {
		t.Fatalf("exporter not ready: %v\nOutput:%s", err, out.String())
	}

	resp, err := http.Get("http://localhost:9100/metrics")
	if err != nil {
		t.Fatalf("failed to fetch metrics: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	metrics := string(body)

	// Debug: Print some metrics to see what's available
	t.Logf("Metrics contains aws_connect_exporter: %v", strings.Contains(metrics, "aws_connect_exporter"))
	if !strings.Contains(metrics, "aws_connect_exporter_agent_realtime_status_agent_state") {
		t.Errorf("missing agent_state metrics")
	}
	if !strings.Contains(metrics, "aws_connect_exporter_agent_state_agents") {
		t.Errorf("missing aggregated agent state metrics")
	}
	if !strings.Contains(metrics, "agent_id=\"user-001\"") {
		t.Logf("Metrics preview: %s", metrics[:min(500, len(metrics))])
		t.Errorf("expected agent metric for user-001")
	}
}

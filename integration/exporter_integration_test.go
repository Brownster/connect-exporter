package integration_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
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

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// waitForPortClosed waits for a port to become available (closed)
func waitForPortClosed(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortAvailable(port) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("port %d still in use after %v", port, timeout)
}

func waitForURL(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

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

		// Exponential backoff with jitter
		time.Sleep(backoff)
		backoff = time.Duration(float64(backoff) * 1.5)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
	return fmt.Errorf("timeout waiting for %s after %v", url, timeout)
}

func TestExporterWithMockAPI(t *testing.T) {
	// Check if exporter port is available
	if !isPortAvailable(9090) {
		t.Skip("Port 9090 is not available, skipping integration test")
	}

	// Check if mock API is running, start it if not
	var mockCmd *exec.Cmd
	var startedMockAPI bool

	if resp, err := http.Get("http://localhost:3000/health"); err != nil || resp.StatusCode != http.StatusOK {
		// Mock API not running, start it
		if !isPortAvailable(3000) {
			t.Skip("Port 3000 is not available and mock API is not running, skipping integration test")
		}

		mockDir := filepath.Join("..", "docs", "tools", "aws-mock-api-main")
		if _, err := os.Stat(filepath.Join(mockDir, "node_modules")); err != nil {
			npmCmd := exec.Command("npm", "install")
			npmCmd.Dir = mockDir
			if out, err := npmCmd.CombinedOutput(); err != nil {
				t.Fatalf("failed to install mock API deps: %v\n%s", err, string(out))
			}
		}
		mockPath := filepath.Join(mockDir, "server.js")
		mockCmd = exec.Command("node", mockPath)
		if err := mockCmd.Start(); err != nil {
			t.Fatalf("failed to start mock API: %v", err)
		}
		startedMockAPI = true

		// Wait longer for mock API startup to avoid intermittent timeouts
		if err := waitForURL("http://localhost:3000/health", 30*time.Second); err != nil {
			t.Fatalf("mock API not ready: %v", err)
		}

		// Additional delay to ensure mock API is fully initialized
		time.Sleep(2 * time.Second)
	} else {
		// Mock API is already running
		if resp != nil {
			resp.Body.Close()
		}
		t.Log("Using existing mock API on port 3000")
	}

	if startedMockAPI && mockCmd != nil {
		defer func() {
			_ = mockCmd.Process.Kill()
			// Wait for port to be released
			_ = waitForPortClosed(3000, 5*time.Second)
		}()
	}

	exporterPath := "/home/marc/Documents/github/dev/connect-exporter/connect-exporter"
	exporterCmd := exec.Command(exporterPath,
		"--config.file=config.development.yml",
		"--web.listen-port=9090",
		"--no-advertise",
		"--labels.fqdn=localhost",
		"--labels.location=test",
		"--labels.country=US",
		"--labels.ip-address=127.0.0.1",
	)
	exporterCmd.Dir = ".." // Set working directory to project root
	exporterCmd.Env = append(os.Environ(),
		"ENV=development",
		"AWS_ACCESS_KEY_ID=dummy",
		"AWS_SECRET_ACCESS_KEY=dummy",
		"AWS_REGION=us-east-1",
	)
	var out bytes.Buffer
	exporterCmd.Stdout = &out
	exporterCmd.Stderr = &out

	t.Logf("Starting exporter with command: %v", exporterCmd.Args)
	t.Logf("Working directory: %s", exporterCmd.Dir)
	t.Logf("Environment: ENV=development, AWS_REGION=us-east-1")

	if err := exporterCmd.Start(); err != nil {
		t.Fatalf("failed to start exporter: %v", err)
	}

	// Give exporter a moment to start and log initial output
	time.Sleep(5 * time.Second)
	t.Logf("Initial exporter output after 5s: %s", out.String())
	defer func() {
		_ = exporterCmd.Process.Kill()
		// Wait for port to be released
		_ = waitForPortClosed(9090, 5*time.Second)
	}()

	// Wait longer for exporter startup and provide more diagnostic info
	if err := waitForURL("http://localhost:9090/metrics", 45*time.Second); err != nil {
		t.Logf("Exporter command output: %s", out.String())
		t.Fatalf("exporter not ready: %v\nOutput: %s", err, out.String())
	}

	// Brief delay to ensure metrics are fully populated
	time.Sleep(3 * time.Second)

	resp, err := http.Get("http://localhost:9090/metrics")
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

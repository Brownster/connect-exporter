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
    mockPath := filepath.Join("..", "docs", "tools", "aws-mock-api-main", "simple-server.js")
    mockCmd := exec.Command("node", mockPath)
    if err := mockCmd.Start(); err != nil {
        t.Fatalf("failed to start mock API: %v", err)
    }
    defer mockCmd.Process.Kill()

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
    if !strings.Contains(metrics, "aws_connect_exporter_realtime_queue_contacts_in_queue") {
        t.Errorf("missing realtime queue metric")
    }
    if !strings.Contains(metrics, "aws_connect_exporter_agent_status_agent_status_count") {
        t.Errorf("missing agent status metric")
    }
}


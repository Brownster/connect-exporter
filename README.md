# AWS Connect Exporter

Prometheus exporter that collects metrics from AWS Connect using the monitoring-maas framework.

## Prerequisites

- Go 1.21 or newer
- AWS credentials with permissions to access AWS Connect

## Building

```bash
go build -o aws-connect-exporter ./cmd/connect-exporter_exporter
```

## Minimal Configuration

```yaml
listen_address: 127.0.0.1
listen_port: 9090

aws:
  region: "us-east-1"
  instance_arn: "arn:aws:connect:us-east-1:123456789012:instance/your-instance-id"

collectors:
  realtime_queue:
    enabled: true
    interval: 60
  agent_status:
    enabled: true
    interval: 300
  agent_realtime_status:
    enabled: true
    interval: 60
```

## Running

```bash
./aws-connect-exporter \
  --web.listen-port=9090 \
  --labels.fqdn=localhost \
  --labels.location=dev \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1 \
  --config.file=config.aws-connect.yml \
  --no-advertise
```

## Metrics Exposed

- `aws_connect_realtime_queue_contacts_in_queue`
- `aws_connect_realtime_queue_oldest_contact_age_seconds`
- `aws_connect_realtime_queue_agents_non_productive`
- `aws_connect_realtime_queue_slots_available`
- `aws_connect_realtime_queue_slots_active`
- `aws_connect_realtime_queue_agents_available`
- `aws_connect_realtime_queue_agents_staffed`
- `aws_connect_realtime_queue_agents_on_contact`
- `aws_connect_realtime_queue_agents_after_contact_work`
- `aws_connect_realtime_queue_agents_error`
- `aws_connect_historical_queue_contacts_queued_today`
- `aws_connect_historical_queue_contacts_abandoned_today`
- `aws_connect_historical_queue_contacts_handled_today`
- `aws_connect_agent_status_count`
- `aws_connect_agent_status_display_order`
- `aws_connect_agent_status_custom`
- `aws_connect_agent_realtime_status_agents`

All queue metrics include a `channel` label. Possible values are
`VOICE`, `CHAT`, or `TASK`.

## Required IAM Permissions

- `connect:ListQueues`
- `connect:GetCurrentMetricData`
- `connect:GetMetricDataV2`
- `connect:ListAgentStatuses`
- `connect:GetCurrentUserData`

## Testing with Mock AWS API

For testing and development, you can use the included mock AWS Connect API server instead of connecting to real AWS services.

### Prerequisites for Testing

- Node.js (v16 or newer)
- npm

### Step-by-Step Testing Instructions

1. **Start the Mock API Server**
   ```bash
   cd docs/tools/aws-mock-api-main
   npm install
   npm start
   ```
   The mock server will start on `http://localhost:3000` and simulate AWS Connect API endpoints.

2. **Configure the Exporter for Mock API**
   Create a test configuration file `config.test.yml`:
   ```yaml
   listen_address: 127.0.0.1
   listen_port: 9091
   
   aws:
     region: "eu-west-2"
     instance_arn: "arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772"
     endpoint: "http://localhost:3000"  # Point to mock API
   
   collectors:
     realtime_queue:
       enabled: true
       interval: 60
     agent_status:
       enabled: true
       interval: 300
     agent_realtime_status:
       enabled: true
       interval: 60
   ```

3. **Build the Exporter**
   ```bash
   go build -o connect-exporter ./cmd/connect-exporter_exporter
   ```

4. **Run the Exporter with Mock API**
   ```bash
   ./connect-exporter \
     --web.listen-port=9091 \
     --labels.fqdn=localhost \
     --labels.location=test \
     --labels.country=UK \
     --labels.ip-address=127.0.0.1 \
     --config.file=config.test.yml \
     --no-advertise
   ```

5. **Verify Metrics Collection**
   Open another terminal and check that metrics are being collected:
   ```bash
   curl http://localhost:9091/metrics
   ```
   You should see AWS Connect metrics with mock data.

### Mock API Features

The mock API server provides:
- **Mock Queues**: Sales, Support, and Monitoring queues with different channels (VOICE, CHAT, TASK)
- **Realistic Metrics**: Random but correlated metric values that simulate real AWS Connect data
- **Error Simulation**: Configurable error rate for testing error handling
- **Pagination**: Supports AWS-style pagination with NextToken
- **Multiple Channels**: VOICE, CHAT, and TASK channel support

### Adjusting Mock API Behavior

You can modify the error rate during testing:
```bash
# Set 10% error rate
curl -X POST http://localhost:3000/admin/errorRate \
  -H 'Content-Type: application/json' \
  -d '{"rate": 0.1}'

# Set 0% error rate (no errors)
curl -X POST http://localhost:3000/admin/errorRate \
  -H 'Content-Type: application/json' \
  -d '{"rate": 0.0}'
```

### Running Tests

To run the integration tests with the mock API:
```bash
# Make sure mock API is running first
go test ./... -v
```


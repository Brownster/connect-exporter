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

Metrics are published under the `aws_connect_exporter_*` namespace. The main
metrics produced by the exporter are listed below:

- `aws_connect_exporter_realtime_queue_connect_queue_contacts_in_queue`
- `aws_connect_exporter_realtime_queue_connect_queue_oldest_contact_age_seconds`
- `aws_connect_exporter_realtime_queue_connect_queue_slots_available`
- `aws_connect_exporter_realtime_queue_connect_queue_slots_active`
- `aws_connect_exporter_realtime_queue_connect_queue_agents` (label: `state`)
- `aws_connect_exporter_historical_queue_connect_historical_contacts_abandoned_total`
- `aws_connect_exporter_historical_queue_connect_historical_contacts_handled_total`
- `aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total`
- `aws_connect_exporter_historical_calls_connect_historical_contacts_abandoned_total`
- `aws_connect_exporter_agent_status_connect_agent_statuses_total`
- `aws_connect_exporter_agent_realtime_status_connect_agent_status_state`
- `aws_connect_exporter_agent_state_agents`

The monitoring framework also exposes internal scrape metrics (e.g.
`maas_scrape_total`, `maas_scrape_failed_total`, and
`maas_scrape_duration_seconds`) to help troubleshoot collection issues.


All queue metrics include a `channel` label. Possible values are
`VOICE`, `CHAT`, or `TASK`.

## Metrics Glossary and Mapping

This exporter provides real-time and historical metrics from the Amazon Connect API. The metric names in Prometheus may differ slightly from the terminology used in the AWS Console or in business requirements. The following table maps common terms to the metrics provided by this exporter.

| Business Metric / Stakeholder Term | Prometheus Metric & Query | AWS API Metric | Description |
| ---------------------------------- | ------------------------- | -------------- | ----------- |
| **Callers in Queue** | `aws_connect_realtime_queue_connect_queue_contacts_in_queue{...}` | `CONTACTS_IN_QUEUE` | The number of contacts currently waiting in a specific queue. |
| **Longest Wait Time** | `aws_connect_realtime_queue_connect_queue_oldest_contact_age_seconds{...}` | `OLDEST_CONTACT_AGE` | The age, in seconds, of the longest-waiting contact in a queue. |
| **Agent Status per Queue** | `aws_connect_realtime_queue_connect_queue_agents{state="..."}` | `AGENTS_AVAILABLE`, etc. | The number of agents in a specific state (`Available`, `OnContact`). |
| **Total Concurrent Calls** | `sum(aws_connect_realtime_queue_connect_queue_agents{state="OnContact"})` | `AGENTS_ON_CONTACT` | **Calculated Total:** The number of agents handling any contact (voice, chat, task). |
| **Total Calls Today (Inbound)** | `increase(aws_connect_historical_calls_connect_historical_contacts_handled_total{direction="inbound"}[1d])` | `CONTACTS_HANDLED` | **Calculated Total:** The number of inbound contacts handled today. |
| **Total Calls Today (Outbound)** | `increase(aws_connect_historical_calls_connect_historical_contacts_handled_total{direction="outbound"}[1d])` | `CONTACTS_HANDLED_OUTBOUND` | **Calculated Total:** The number of outbound contacts handled today. |

### Understanding the Historical Collectors

The exporter ships with two complementary collectors for historical metrics:

**historical_queue Collector (The Analyst's Tool)**

- **Purpose:** Granular, deep-dive analysis.
- **Use Case:** Answering detailed questions like, "What is the abandonment rate specifically for our CHAT channel in the Support queue?"
- **Provides:** Depth and detail.

**historical_calls Collector (The Dashboard Builder's Tool)**

- **Purpose:** High-level, business-facing dashboarding and trend analysis.
- **Use Case:** Answering common questions like, "Show me a graph of our total inbound call rate today," which is a simple `rate(aws_connect_historical_calls_connect_historical_contacts_handled_total[5m])` query. Or, "How many calls did we handle yesterday?" using `increase(aws_connect_historical_calls_connect_historical_contacts_handled_total[1d])`.
- **Provides:** Convenience and compatibility with standard PromQL functions.

## Required IAM Permissions

- `connect:ListQueues`
- `connect:GetCurrentMetricData`
- `connect:GetMetricDataV2`
- `connect:ListAgentStatuses`
- `connect:GetCurrentUserData`
- `connect:ListUsers`
- `connect:ListRoutingProfileQueues`

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
   Verify it is running:
   ```bash
   curl http://localhost:3000/health
   ```

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
   ENV=development ./connect-exporter \
     --web.listen-port=9091 \
     --labels.fqdn=localhost \
     --labels.location=test \
     --labels.country=UK \
     --labels.ip-address=127.0.0.1 \
     --config.file=config.test.yml \
     --no-advertise
   ```
   Note: The `ENV=development` environment variable is required to enable mock API mode.

5. **Verify Metrics Collection**
   Open another terminal and check that metrics are being collected:
   ```bash
   curl http://localhost:9091/metrics
   ```
   You should see AWS Connect metrics with mock data.

### Enhanced Mock API Features

The mock API server now provides **complete coverage** for all collectors, enabling full dashboard development without production AWS access:

#### **✅ Complete Collector Coverage**
- **Realtime Queue Collector**: ✅ 100% Working - Queue performance, wait times, agent availability
- **Agent Status Collector**: ✅ 100% Working - Status definitions and categorization
- **Agent Realtime Status Collector**: ✅ 100% Working - Live agent status distribution  
- **Historical Queue Collector**: ✅ 100% Working - Daily call volumes, abandonment rates

#### **🎯 Dashboard-Ready Data**
- **11 Realistic Users**: Distributed across different statuses (5 Available, 3 Busy, 2 After Contact Work, 1 Custom Break)
- **5 Mock Queues**: Sales, Support, and Monitoring queues with VOICE, CHAT, and TASK channels
- **Dynamic Metrics**: Time-based values that change realistically for testing dashboards
- **Operational Metrics**: All key Kensington metrics supported:
  - Queue wait times and caller counts
  - Agent status distribution by queue/channel  
  - Daily call volumes (inbound/outbound split)
  - Abandonment rates and contact handling

#### **🔧 Technical Features**
- **AWS API Compatibility**: Handles both GetCurrentMetricData and GetMetricDataV2 structures
- **Error Simulation**: Configurable error rate for testing resilience
- **Pagination Support**: AWS-style pagination with NextToken
- **Realistic Data Generation**: Correlated metrics that simulate real contact center behavior
- **Health Monitoring**: `GET /health` endpoint for service verification

### Complete Metrics Output Example

With the enhanced mock API, you'll see all operational metrics needed for dashboard building:

```bash
curl http://localhost:9091/metrics | grep aws_connect_exporter
```

**Sample Output:**
```
# Real-time Queue Performance
aws_connect_exporter_realtime_queue_connect_queue_agents{channel="VOICE",queue_id="queue-sales-01",queue_name="Sales Queue",queue_type="STANDARD",state="NonProductive",region="eu-west-2"} 2
aws_connect_exporter_realtime_queue_connect_queue_slots_available{channel="VOICE",queue_id="queue-sales-01",queue_name="Sales Queue",queue_type="STANDARD",region="eu-west-2"} 4

# Agent Status Distribution
aws_connect_exporter_agent_realtime_status_connect_agent_status_state{agent_id="user-001",username="alice.smith",routing_profile_id="rp-sales",queue_id="queue-sales-01",status_id="status-001",status_name="Available",status_type="unknown",region="eu-west-2"} 1

# Daily Performance Metrics
aws_connect_exporter_historical_queue_connect_historical_contacts_abandoned_total{channel="VOICE",queue_id="queue-sales-01",queue_name="Sales Queue",queue_type="STANDARD",region="eu-west-2"} 6

# Agent Status Definitions
aws_connect_exporter_agent_status_connect_agent_statuses_total{region="eu-west-2",status_id="status-001",status_name="Available",status_type="ROUTABLE"} 1
aws_connect_exporter_agent_status_connect_agent_statuses_total{region="eu-west-2",status_id="status-002",status_name="Busy",status_type="ROUTABLE"} 1
```

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

### Coverage Analysis

For detailed technical analysis of mock API coverage and collector testing, see:
- **[MOCK_API_GAPS.md](MOCK_API_GAPS.md)** - Complete coverage analysis and testing results

## 📊 Grafana Dashboards

Two comprehensive dashboards are provided for operational monitoring:

### 🚀 **Real-time Operations Dashboard**
- **Purpose:** Live operational monitoring (30s refresh)
- **Key Metrics:** Queue availability, agent status distribution, operational alerts
- **Use Case:** NOC monitoring, immediate response to issues

### 📈 **Daily Performance Analytics Dashboard**  
- **Purpose:** Strategic performance analysis (5m refresh)
- **Key Metrics:** Daily volumes, abandonment rates, trend analysis
- **Use Case:** Management reporting, performance optimization

### Quick Dashboard Setup
```bash
cd dashboards
./import-dashboards.sh
```

For detailed dashboard documentation, see **[dashboards/README.md](dashboards/README.md)**

### Running Tests

To run the integration tests with the mock API:
```bash
# Make sure mock API is running first
go test ./... -v
```


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
- `aws_connect_realtime_queue_agents_available`
- `aws_connect_realtime_queue_agents_staffed`
- `aws_connect_realtime_queue_agents_on_contact`
- `aws_connect_realtime_queue_agents_after_contact_work`
- `aws_connect_realtime_queue_agents_error`
- `aws_connect_agent_status_count`
- `aws_connect_agent_status_display_order`
- `aws_connect_agent_status_custom`

## Required IAM Permissions

- `connect:ListQueues`
- `connect:GetCurrentMetricData`
- `connect:ListAgentStatuses`


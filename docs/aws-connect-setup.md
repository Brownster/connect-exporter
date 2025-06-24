# AWS Connect Exporter Setup Guide

This guide covers the setup and configuration of the AWS Connect Exporter using the monitoring-maas framework.

## Prerequisites

- Go 1.21 or newer
- AWS Account with AWS Connect instance
- AWS CLI configured (optional, for easier credential management)

## AWS IAM Permissions

The exporter requires the following AWS IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "connect:ListQueues",
        "connect:GetCurrentMetricData",
        "connect:GetMetricDataV2",
        "connect:ListAgentStatuses",
        "connect:GetCurrentUserData"
      ],
      "Resource": "arn:aws:connect:*:*:instance/your-instance-id"
    }
  ]
}
```

### Creating IAM Role for EC2 Instance

If running on EC2, create an IAM role with the above policy:

```bash
# Create trust policy for EC2
cat > trust-policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

# Create the role
aws iam create-role --role-name ConnectExporterRole --assume-role-policy-document file://trust-policy.json

# Create and attach the policy
aws iam create-policy --policy-name ConnectExporterPolicy --policy-document file://connect-exporter-policy.json
aws iam attach-role-policy --role-name ConnectExporterRole --policy-arn arn:aws:iam::ACCOUNT:policy/ConnectExporterPolicy

# Create instance profile
aws iam create-instance-profile --instance-profile-name ConnectExporterProfile
aws iam add-role-to-instance-profile --instance-profile-name ConnectExporterProfile --role-name ConnectExporterRole
```

## Configuration

### 1. Basic Configuration

Create a configuration file `config.yml`:

```yaml
aws:
  region: "us-east-1"
  instance_arn: "arn:aws:connect:us-east-1:123456789012:instance/your-instance-id"

collectors:
  realtime_queue:
    enabled: true
    interval: 60  # seconds
  agent_status:
    enabled: true
    interval: 300  # seconds
```

### 2. Environment Variables

Alternatively, configure using environment variables:

```bash
export AWS_REGION=us-east-1
export AWS_DEFAULT_REGION=us-east-1
```

### 3. CLI Flags

Override configuration with command-line flags:

```bash
./aws-connect-exporter \
  --instance-arn="arn:aws:connect:us-east-1:123456789012:instance/your-instance-id" \
  --region="us-east-1" \
  --config.file="config.yml"
```

## Building and Running

### Build from Source

```bash
# Clone the repository
git clone https://github.com/Brownster/connect-exporter.git
cd connect-exporter

# Build the exporter
go build -o aws-connect-exporter ./cmd/connect-exporter_exporter

# Run with configuration file
./aws-connect-exporter --config.file=config.aws-connect.yml
```

### Using Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o aws-connect-exporter ./cmd/connect-exporter_exporter

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/aws-connect-exporter /aws-connect-exporter
COPY config.aws-connect.yml /config.yml
EXPOSE 9090
CMD ["/aws-connect-exporter", "--config.file=/config.yml"]
```

## Metrics

### Realtime Queue Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `aws_connect_realtime_queue_contacts_in_queue` | Gauge | Number of contacts waiting in queue | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_oldest_contact_age_seconds` | Gauge | Age in seconds of oldest contact in queue | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_agents_non_productive` | Gauge | Number of agents in Non-Productive Time (NPT) | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_slots_available` | Gauge | Number of contact slots available | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_slots_active` | Gauge | Number of contact slots currently active | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_agents_available` | Gauge | Number of agents available | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_agents_staffed` | Gauge | Number of agents staffed | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_agents_on_contact` | Gauge | Number of agents on contact | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_agents_after_contact_work` | Gauge | Number of agents in after contact work | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_realtime_queue_agents_error` | Gauge | Number of agents in error state | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |

The `channel` label indicates whether a queue handles `VOICE`, `CHAT`, or `TASK` contacts. This applies to both realtime and historical queue metrics.

### Historical Queue Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `aws_connect_historical_queue_contacts_queued_today` | Gauge | Number of contacts queued since midnight UTC | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_historical_queue_contacts_abandoned_today` | Gauge | Number of contacts abandoned since midnight UTC | `queue_id`, `queue_name`, `queue_type`, `channel`, `region` |
| `aws_connect_historical_queue_contacts_handled_today` | Gauge | Number of contacts handled since midnight UTC | `queue_id`, `queue_name`, `queue_type`, `channel`, `direction`, `region` |

### Agent Status Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `aws_connect_agent_status_count` | Gauge | Number of agents in each status | `status_id`, `status_name`, `status_type`, `region` |
| `aws_connect_agent_status_display_order` | Gauge | Display order of status types | `status_id`, `status_name`, `status_type`, `region` |
| `aws_connect_agent_status_custom` | Gauge | Custom status indicator (1=custom, 0=built-in) | `status_id`, `status_name`, `status_type`, `region` |
| `aws_connect_agent_realtime_status_agents` | Gauge | Number of agents in each realtime status | `status_id`, `status_name`, `status_type`, `region` |

## Monitoring

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'aws-connect-exporter'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 60s
    scrape_timeout: 30s
```

### Grafana Dashboard

Sample Grafana queries:

```promql
# Current contacts in queue
aws_connect_realtime_queue_contacts_in_queue

# Average wait time by queue
aws_connect_realtime_queue_oldest_contact_age_seconds

# Agent status distribution
sum by (status_name) (aws_connect_agent_status_count)
```

## Troubleshooting

### Common Issues

1. **Permission Denied**: Verify IAM permissions and Instance ARN
2. **Instance Not Found**: Check Instance ARN format and region
3. **High Memory Usage**: Reduce collection frequency for large instances
4. **Timeout Errors**: Increase timeout values in configuration

### Debug Logging

Enable debug logging:

```bash
./aws-connect-exporter --log.level=debug --config.file=config.yml
```

### Health Check

Check exporter health:

```bash
curl http://localhost:9090/health
curl http://localhost:9090/metrics
```

## Performance Tuning

### Batch Processing

The exporter automatically batches queue requests (max 100 queues per request) to optimize AWS API usage.

### Collection Intervals

Recommended intervals:
- **Realtime Queue Metrics**: 30-60 seconds
- **Agent Status Metrics**: 5-10 minutes

### Resource Usage

Expected resource usage:
- **Memory**: 50-100MB for typical instances
- **CPU**: Minimal, spikes during collection
- **Network**: ~1-5 API calls per minute per collector

## Cost Optimization

### AWS API Costs

Estimated AWS Connect API costs:
- **ListQueues**: ~$0.0004 per call
- **GetCurrentMetricData**: ~$0.0004 per call
- **GetMetricDataV2**: ~$0.0004 per call
- **ListAgentStatuses**: ~$0.0004 per call
- **GetCurrentUserData**: ~$0.0004 per call

For a typical deployment (60-second intervals):
- Daily cost: ~$0.05-0.15
- Monthly cost: ~$1.50-4.50

### Development Cost Reduction

Use the included mock API server for development:

```bash
# Start mock API
cd docs/tools/aws-mock-api-main
node server.js

# Set development mode
export ENV=development
./aws-connect-exporter --config.file=config.development.yml
```

This eliminates AWS API costs during development and testing.
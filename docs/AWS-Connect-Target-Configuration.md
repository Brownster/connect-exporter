# AWS Connect Target Configuration

This document describes how to set up an AWS Connect integration with SM3.

Integration is done via APIs below:
- [AWS Connect API](https://docs.aws.amazon.com/connect/latest/APIReference/) (it requires AWS credentials, it uses AWS SDK for Go).
- [AWS Connect Real-time Metrics API](https://docs.aws.amazon.com/connect/latest/APIReference/API_GetCurrentMetricData.html) (requires IAM permissions for real-time data).
- [AWS Connect Historical Metrics API](https://docs.aws.amazon.com/connect/latest/APIReference/API_GetMetricDataV2.html) (requires IAM permissions for historical data).

## Quick Reference for Technicians

**📺 For Wallboard/Dashboard Requirements:**
- See **[WALLBOARD_MAPPING.md](WALLBOARD_MAPPING.md)** - Translate customer wallboard needs into configuration

**📋 For Service Level Agreements (SLA monitoring):**
- See **[THRESHOLD_CONFIG.md](THRESHOLD_CONFIG.md)** - Complete guide for configuring customer-specific service level thresholds

**📊 For Custom Metrics (beyond default set):**
- See **[adding-metrics.md](adding-metrics.md)** - Step-by-step guide for adding customer-requested metrics

**🔧 Basic Setup:** Continue reading this document for standard deployment configuration

## Instructions

### Create AWS IAM User/Role and set permissions
Create an IAM user or role with the required permissions listed in the [IAM Permissions section](#iam-permissions) below.

### Configure AWS Credentials
Set up AWS credentials using one of the following methods:
- **IAM Role** (recommended for EC2 instances)
- **Environment Variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`  
- **AWS Profile**: `AWS_PROFILE=your-profile-name`

### Find your AWS Connect Instance ARN
Navigate to your AWS Connect console and copy the Instance ARN from the instance overview page.
Format: `arn:aws:connect:region:account:instance/instance-id`

Use `instance_arn`, AWS credentials, and configuration values in your exporter.

## Target Configuration

```yaml
exporter_awsconnect:
  awsconnect_sabio.co.uk:
    listen_port: 8120
    extra_args: "--web.listen-port=8120 --labels.fqdn=monitoring-server --labels.location=datacenter --labels.country=UK --labels.ip-address=10.0.1.100 --config.file=config.aws-connect-validated.yml --no-advertise --log.level=info"
    instance_arn: "arn:aws:connect:eu-west-2:123456789012:instance/your-instance-id"
    aws_region: "eu-west-2"
    aws_access_key_id: "ENC[PKCS7...]"
    aws_secret_access_key: "ENC[PKCS7...]"
```

## Extra Args

| Parameter | Description |
|-----------|-------------|
| `--web.listen-port` | Port for metrics endpoint. Default: 9090 |
| `--labels.fqdn` | **Required**. Fully qualified domain name for the exporter instance |
| `--labels.location` | **Required**. Physical location/datacenter identifier |
| `--labels.country` | **Required**. Country code (e.g., UK, US, CA) |
| `--labels.ip-address` | **Required**. IP address of the exporter instance |
| `--config.file` | Path to configuration file. Default: config.yml |
| `--no-advertise` | Disable service advertisement. Recommended for testing |
| `--log.level` | Logging level: debug, info, warn, error. Default: info |

## Configuration Features

### Agent Metrics
- **Enhanced Labeling**: Includes `first_name`, `last_name`, `username`, `agent_id`
- **Real-time Status**: Agent availability, routing profile assignments
- **Queue Associations**: Agent-to-queue mappings with routing profiles
- **Status Details**: Current agent state with status type information

### Queue Metrics  
- **Real-time Metrics**: Contact volumes, wait times, agent availability
- **Queue Capacity**: Available slots, active contacts, queue depth
- **Channel Support**: Voice, chat, task channel metrics
- **Performance KPIs**: Service levels, abandonment rates

### Historical Metrics
- **Daily Performance**: Contact handling volumes (handled, abandoned, queued)
- **Service Level Compliance**: Configurable thresholds with actual vs target
- **Operational KPIs**: Average handle times, queue answer times
- **Trend Analysis**: Historical data for capacity planning

### Advanced Features
- **Batch Processing**: Handles 100+ queues with automatic API batching
- **Rate Limiting**: Built-in 2 TPS compliance with exponential backoff
- **Error Resilience**: Retry logic with graceful failure handling
- **Agent Name Caching**: 12-hour cache with rate-limited refresh

## Custom Configuration Guides

### Service Level Thresholds (SLA Monitoring)
For customers requiring specific service level agreements (e.g., "80% of calls answered within 20 seconds"), see the comprehensive threshold configuration guide:

**📋 [THRESHOLD_CONFIG.md](THRESHOLD_CONFIG.md)**
- Step-by-step SLA configuration walkthrough
- Multiple customer scenarios (Premium/Standard tiers, Compliance monitoring)
- Best practices for technicians
- Comprehensive troubleshooting guide

### Adding Custom Metrics
For customers needing metrics beyond the default set (e.g., callback tracking, VIP queue monitoring), see the detailed metrics addition guide:

**📊 [adding-metrics.md](adding-metrics.md)**
- Customer requirements analysis framework
- Real-time vs historical metric configuration
- Common customer scenarios with examples
- Testing and validation procedures

## IAM Permissions

Required AWS IAM policy for the exporter service account:

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
        "connect:GetCurrentUserData",
        "connect:ListUsers",
        "connect:DescribeUser",
        "connect:ListRoutingProfiles",
        "connect:ListRoutingProfileQueues"
      ],
      "Resource": "arn:aws:connect:*:*:instance/your-instance-id"
    }
  ]
}
```

**Note**: Replace `your-instance-id` with your actual AWS Connect instance ID.

## API Requests

| Collector | Permissions Required | API Endpoint | Scrape Frequency | API Requests per Day |
|-----------|---------------------|---------------|------------------|----------------------|
| Agent Realtime Status | `connect:GetCurrentUserData`<br>`connect:ListAgentStatuses`<br>`connect:DescribeUser` | `/current-user-data`<br>`/agent-status` | 60s | 2,880 |
| Generic Realtime | `connect:GetCurrentMetricData`<br>`connect:ListQueues` | `/current-metric-data` | 30s | 2,880 |  
| Generic Historical | `connect:GetMetricDataV2`<br>`connect:ListQueues` | `/metric-data-v2` | 300s | 288 |
| Routing Profiles | `connect:ListRoutingProfiles`<br>`connect:ListRoutingProfileQueues` | `/routing-profiles` | 300s | 288 |
| Agent Management | `connect:ListUsers`<br>`connect:DescribeUser` | `/users`<br>`/users/{userId}` | 21,600s (6h) | 8 |

**Total Daily API Requests**: ~6,344

## Rate Limits & Best Practices

AWS Connect API rate limits (per region):
- **Most APIs**: 2 TPS (transactions per second)
- **GetCurrentMetricData**: 5 TPS
- **GetMetricDataV2**: 5 TPS  
- **ListUsers/DescribeUser**: 2 TPS

### Built-in Rate Limiting
The exporter includes:
- **Token bucket rate limiter**: 2 calls/sec with burst of 5
- **Exponential backoff**: 1s → 5s max retry delay
- **Batch processing**: Respects 100-item API limits
- **Error resilience**: Continues on partial failures

### Monitoring Recommendations
- Monitor `aws_connect_exporter_scrape_duration_seconds` for API performance
- Check `aws_connect_exporter_scrape_success` for collection health  
- Watch for `ThrottlingException` errors in logs
- Adjust collection intervals if rate limits are hit consistently

## Deployment Considerations

### Resource Requirements
- **CPU**: Minimal, spikes during collection cycles
- **Memory**: 50-150MB depending on instance size
- **Network**: 5-15 API calls per minute (based on configuration)
- **Disk**: <1GB for binary and configuration

### Availability
- Configure appropriate scrape intervals to avoid API exhaustion
- Monitor exporter health with `/health` endpoint

### Security
- Store AWS credentials securely (IAM roles preferred)
- Use least-privilege IAM policies

## Troubleshooting

### Common Issues

1. **"ThrottlingException" Errors**
   - Increase collection intervals in configuration
   - Check for other AWS Connect API consumers
   - The exporter includes automatic retry with backoff

2. **Empty Agent Names (`first_name`/`last_name`)**
   - Verify `connect:DescribeUser` IAM permission
   - Ensure agents have IdentityInfo populated in AWS Connect
   - Check agent name collection logs with `--log.level=debug`

3. **"100 queue limit exceeded"**
   - This is handled automatically with batching
   - Verify batch processing in logs
   - Contact AWS Support if limits need adjustment

4. **High Memory Usage**
   - Reduce collection frequencies for large instances
   - Disable unnecessary collectors in configuration
   - Monitor with `aws_connect_exporter_scrape_total` metrics

5. **Service Level Thresholds Not Working**
   - See detailed troubleshooting in **[THRESHOLD_CONFIG.md](THRESHOLD_CONFIG.md)**
   - Common issues: incorrect comparison operators, missing historical data
   - Validation procedures for customer SLA requirements

6. **Custom Metrics Not Appearing**
   - See comprehensive guide in **[adding-metrics.md](adding-metrics.md)**
   - Verify AWS metric names and instance support
   - Step-by-step validation process included

### Debug Mode
```bash
./aws-connect-exporter --log.level=debug --config.file=config.yml
```

This will provide detailed logging for troubleshooting API calls, rate limiting, and metric collection.

### Advanced Configuration Support
For complex customer requirements involving custom metrics or SLA monitoring, refer to:
- **[THRESHOLD_CONFIG.md](THRESHOLD_CONFIG.md)** - Service level agreements and compliance monitoring
- **[adding-metrics.md](adding-metrics.md)** - Custom metric implementation

---

**Important Links:**
- [AWS Connect API Documentation](https://docs.aws.amazon.com/connect/latest/APIReference/)
- [AWS Connect Metrics and Reports](https://docs.aws.amazon.com/connect/latest/adminguide/monitoring-cloudwatch.html)
- [AWS IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)

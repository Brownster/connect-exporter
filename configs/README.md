# AWS Connect Exporter - Configuration Files

This directory contains various configuration files for different environments and use cases.

## Configuration Files

### 🔧 Development Configurations

- **`config.development.yml`** - Development configuration using mock AWS Connect API
  - Uses mock API server (no AWS costs)
  - Fast collection intervals for testing
  - Comprehensive realtime and historical metrics
  - Includes agent name collection with rate limiting
  - **Usage**: `ENV=development ./bin/aws-connect-exporter --config.file=configs/config.development.yml`

- **`config.test.yml`** - Unit testing configuration
  - Minimal configuration for automated tests
  - Short intervals and timeouts
  - Used by integration test suite

### 🎯 Specialized Configurations

- **`config.aws-connect-realtime-only.yml`** - Realtime metrics only
  - Disables historical collectors for cost optimization
  - Focus on live queue and agent monitoring
  - Ideal for real-time dashboards

- **`config.queue-test.yml`** - Queue-focused testing
  - Specific configuration for queue metric validation
  - Used for queue batching tests (100+ queues)

- **`config.aws-connect.yml`** - Full production template
  - Comprehensive production configuration template
  - All collectors enabled with production intervals
  - Contains examples of all available metrics

## ⚠️ Production Configuration

**Note**: The main production configuration file is kept in the project root:

- **`../config.aws-connect-validated.yml`** - **ACTIVE PRODUCTION CONFIG**
  - Currently validated against real AWS Connect instance
  - Contains 16+ comprehensive historical metrics  
  - Used for testing and validation
  - **This is the primary config file for testing**

## Configuration Structure

All configuration files follow this structure:

```yaml
# AWS Configuration
aws:
  region: "us-east-1"
  instance_arn: "arn:aws:connect:us-east-1:123456789012:instance/your-instance-id"

# Collector Configuration
collectors:
  # Agent realtime status with enhanced labeling
  agent_realtime_status:
    enabled: true
    interval: 60

  # Generic realtime metrics (configuration-driven)
  generic_realtime:
    enabled: true
    interval: 30
    collectors:
      - name: "queue_realtime_metrics"
        metrics:
          - aws_name: "CONTACTS_IN_QUEUE"
            prometheus_name: "connect_queue_contacts_in_queue"
            labels: {}
        groupings: ["QUEUE", "CHANNEL"]
        filters: []

  # Generic historical metrics (configuration-driven)
  generic_historical:
    enabled: true
    interval: 300
    collectors:
      - name: "historical_calls"
        metrics:
          - aws_name: "CONTACTS_HANDLED"
            prometheus_name: "connect_historical_contacts_handled_total"
            labels: {}
        groupings: ["QUEUE", "CHANNEL"]
        thresholds:
          - name: "service_level_20s"
            comparison: "LT"
            value: 20.0

# Global settings
global:
  timeout: 30s
  labels:
    environment: "production"
    service: "aws-connect"
  retry:
    max_attempts: 3
    backoff_duration: 1s
    max_backoff_duration: 5s
```

## Key Features

### 🔄 **Configuration-Driven Metrics**
- **Generic Collectors**: Define any AWS Connect metric via YAML
- **No Code Changes**: Add new metrics without recompiling
- **Flexible Groupings**: QUEUE, CHANNEL, custom groupings per collector
- **Filter Support**: Channel, initiation method, custom filters

### 📊 **Comprehensive Metric Coverage**

**Realtime Metrics:**
- Queue performance (contacts in queue, wait times)
- Agent availability by state and routing profile
- Agent realtime status with enhanced labeling
- Queue capacity and utilization

**Historical Metrics:**
- Daily contact volumes (handled, abandoned, queued)
- Service level performance with thresholds
- Average handle times and queue answer times
- Abandonment rates and agent answer rates

### 🏷️ **Enhanced Labeling System**

**Agent Metrics Include:**
- `agent_id`, `username`, `first_name`, `last_name`
- `routing_profile_id`, `routing_profile_name`
- `status_id`, `status_name`, `status_type`
- `queue_id`, `queue_name`, `region`

**Queue Metrics Include:**
- `queue_id`, `queue_name`, `queue_type`, `queue_arn`
- `channel` (VOICE, CHAT, TASK)
- `direction` (inbound, outbound) - for historical metrics
- `state` (Available, OnContact, etc.) - for agent states
- `region`

## Environment Variables

### Development Mode
```bash
export ENV=development                                    # Enables mock API mode
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000   # Optional mock API endpoint
```

### AWS Configuration
```bash
export AWS_REGION=us-east-1
export AWS_DEFAULT_REGION=us-east-1
export AWS_PROFILE=your-profile
```

## Collector Control

### Enabling/Disabling Collectors
```yaml
collectors:
  agent_realtime_status:
    enabled: false    # Disable agent collection entirely
  generic_realtime:
    enabled: true     # Keep queue metrics only
```

### Collection Intervals

**Development (Fast):**
- Realtime: 30 seconds
- Historical: 300 seconds (5 minutes)

**Production (Optimized):**
- Realtime: 60-120 seconds
- Historical: 300-600 seconds (5-10 minutes)

## Cost Optimization

### Development
- Use `config.development.yml` with mock API (zero AWS costs)
- Test all features without API charges

### Production
- Adjust collection intervals based on needs
- Use `agent_realtime_status: enabled: false` to disable agent tracking
- Filter specific channels: `["VOICE"]` instead of `["VOICE", "CHAT", "TASK"]`

## Validation

All configurations have been validated for:
- ✅ **Syntax**: Valid YAML structure
- ✅ **AWS Connect API**: Compatible with current API versions
- ✅ **Metric Generation**: Confirmed metric output
- ✅ **Batching**: Handles 100+ queues with proper API batching
- ✅ **Error Handling**: Graceful failure and retry behavior

## Support

For detailed configuration help:
- **Setup Guide**: `../docs/aws-connect-setup.md`
- **Adding Metrics**: `../docs/adding-metrics.md`
- **Development Guide**: `../docs/development-with-mock-api.md`
- **Troubleshooting**: `../docs/debugging-guide.md`
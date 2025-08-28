# Adding New Metrics to AWS Connect Exporter

This guide helps technicians add custom metrics for specific customer requirements. Use this when customers need monitoring beyond the default metric set.

## When to Add Custom Metrics

**Common scenarios:**
- Customer needs specific KPIs not in default configuration (e.g., callback metrics)  
- Business-specific reporting requirements (e.g., VIP queue handling times)
- Compliance monitoring (e.g., PCI queue answer rates)
- Department-specific metrics (e.g., sales vs support performance)

## Prerequisites

Before adding metrics:
- ✅ Identify the exact AWS Connect metric name from [AWS Connect Metrics Reference](https://docs.aws.amazon.com/connect/latest/adminguide/historical-metrics-definitions.html)
- ✅ Understand if it's a real-time or historical metric
- ✅ Know the customer's specific requirements (units, labels, groupings)
- ✅ Have access to AWS Connect console to verify metric availability

## Step-by-Step Process

### Step 1: Customer Requirements Analysis
**Customer request:** "We need to monitor average wait time for our VIP queue"

**Analysis questions:**
1. **What's the AWS metric name?** → `AVG_QUEUE_ANSWER_TIME`
2. **Real-time or historical?** → Real-time (immediate visibility)
3. **Which queues?** → VIP queue only
4. **How often should we collect?** → Every 30 seconds
5. **What labels do they need?** → Queue name, priority level

### Step 2: Find the AWS Connect Metric Name

**Method 1: AWS Connect Console**
1. Go to Analytics and optimization > Real-time metrics
2. Look for the metric in the available columns
3. Note the exact name (case-sensitive)

**Method 2: AWS Documentation**
- Check [AWS Connect Metrics Reference](https://docs.aws.amazon.com/connect/latest/adminguide/real-time-metrics-definitions.html)
- Real-time metrics: `AGENTS_*`, `CONTACTS_*`, `SLOTS_*`
- Historical metrics: `*_TOTAL`, `SERVICE_LEVEL`, `AVG_*`

### Step 3: Configure the Metric

**For Real-time Metrics:**
```yaml
collectors:
  generic_realtime:
    enabled: true
    interval: 30  # 30 seconds
    collectors:
      - name: "vip_queue_monitoring"      # Descriptive name
        metrics:
          - aws_name: "AVG_QUEUE_ANSWER_TIME"
            prometheus_name: "connect_queue_avg_answer_time_seconds"
            labels:
              priority: "vip"              # Custom label for identification
              customer: "acme_corp"        # Customer identifier
        groupings: ["QUEUE"]               # Group by queue
        filters:
          - key: "QUEUES"                  # Filter to specific queue
            values: ["arn:aws:connect:us-east-1:123456789012:instance/instance-id/queue/vip-queue-id"]
```

**For Historical Metrics:**
```yaml
collectors:
  generic_historical:
    enabled: true  
    interval: 300  # 5 minutes (don't set too frequent for historical)
    collectors:
      - name: "daily_performance_metrics"
        metrics:
          - aws_name: "CONTACTS_HANDLED"
            prometheus_name: "connect_daily_contacts_handled_total"
            labels:
              department: "sales"
              metric_type: "volume"
        groupings: ["QUEUE"]
        filters: []
```

### Step 4: Choose Appropriate Groupings

| Grouping | Use When | Example |
|----------|----------|---------|
| `["QUEUE"]` | Per-queue metrics | Different SLAs per queue |
| `["CHANNEL"]` | Per-channel metrics | Voice vs Chat vs Task |
| `["QUEUE", "CHANNEL"]` | Both dimensions | Queue performance by channel |
| `[]` | Overall/aggregated | Total instance-wide metrics |

### Step 5: Configure Filters (Optional)

**Filter by specific queues:**
```yaml
filters:
  - key: "QUEUES"
    values: 
      - "arn:aws:connect:us-east-1:123456789012:instance/instance-id/queue/sales-queue-id"
      - "arn:aws:connect:us-east-1:123456789012:instance/instance-id/queue/support-queue-id"
```

**Filter by channels:**
```yaml  
filters:
  - key: "CHANNELS"
    values: ["VOICE", "CHAT"]  # Exclude TASK channel
```

### Step 6: Naming Conventions

**Prometheus Metric Names:**
- Use lowercase with underscores: `connect_queue_avg_answer_time_seconds`
- Include units in name: `_seconds`, `_percent`, `_total`
- Be descriptive: `connect_historical_service_level_percent` not `connect_sl`

**Labels:**
- Use consistent naming: `queue_name`, `department`, `priority`
- Avoid high cardinality: Don't use timestamps or unique IDs
- Add customer context: `customer: "acme_corp"`, `environment: "prod"`

## Common Customer Metric Requests

### Scenario 1: Callback Metrics
**Customer:** "We need to track callbacks requested vs callbacks completed"

```yaml
collectors:
  generic_historical:
    enabled: true
    interval: 300
    collectors:
      - name: "callback_tracking"
        metrics:
          - aws_name: "CALLBACK_CONTACTS_HANDLED"
            prometheus_name: "connect_callbacks_handled_total"
            labels:
              contact_type: "callback"
          - aws_name: "CONTACTS_QUEUED"
            prometheus_name: "connect_callbacks_requested_total"  
            labels:
              contact_type: "callback"
        groupings: ["QUEUE"]
        filters:
          - key: "QUEUES"
            values: ["arn:aws:connect:region:account:instance/instance-id/queue/callback-queue-id"]
```

### Scenario 2: Agent Performance Metrics
**Customer:** "Track individual agent handle times and occupancy rates"

```yaml
collectors:
  generic_realtime:
    enabled: true
    interval: 60  # 1 minute for agent metrics
    collectors:
      - name: "agent_performance"
        metrics:
          - aws_name: "AGENTS_ON_CONTACT"
            prometheus_name: "connect_agents_on_contact"
            labels:
              department: "sales"
          - aws_name: "AGENTS_AFTER_CONTACT_WORK"
            prometheus_name: "connect_agents_in_acw"
            labels:
              department: "sales"
        groupings: ["QUEUE"]  # This gives per-queue agent counts
        filters: []
```

### Scenario 3: Channel-Specific Metrics
**Customer:** "We need separate metrics for voice vs chat performance"

```yaml
collectors:
  generic_realtime:
    enabled: true
    interval: 30
    collectors:
      - name: "voice_metrics"
        metrics:
          - aws_name: "AVG_QUEUE_ANSWER_TIME"
            prometheus_name: "connect_voice_avg_answer_time_seconds"
            labels:
              channel_type: "voice"
        groupings: ["QUEUE", "CHANNEL"]
        filters:
          - key: "CHANNELS"
            values: ["VOICE"]
      
      - name: "chat_metrics"
        metrics:
          - aws_name: "AVG_QUEUE_ANSWER_TIME"
            prometheus_name: "connect_chat_avg_answer_time_seconds"
            labels:
              channel_type: "chat"
        groupings: ["QUEUE", "CHANNEL"]
        filters:
          - key: "CHANNELS"
            values: ["CHAT"]
```

## Testing and Validation

### Step 1: Configuration Validation
```bash
# Check YAML syntax
python -c "import yaml; yaml.safe_load(open('config.aws-connect-validated.yml'))"

# Validate configuration loading
./bin/aws-connect-exporter --config.file=config.aws-connect-validated.yml --log.level=debug --dry-run
```

### Step 2: Deploy and Test
```bash
# 1. Backup current config
cp config.aws-connect-validated.yml config.aws-connect-validated.yml.backup

# 2. Deploy new configuration
sudo systemctl restart aws-connect-exporter

# 3. Check exporter starts successfully
sudo systemctl status aws-connect-exporter

# 4. Wait for collection cycle (check interval in config)
sleep 60

# 5. Verify new metrics appear
curl -s http://localhost:9090/metrics | grep "your_new_metric_name"
```

### Step 3: Validate with Customer
```bash
# Get sample metric values
curl -s http://localhost:9090/metrics | grep connect_queue_avg_answer_time | head -3

# Compare with AWS Connect console:
# Go to Analytics -> Real-time metrics -> Check the same queues
# Values should be similar (within collection interval window)
```

## Mock API Configuration (Development/Testing)

When developing or testing new metrics, configure the mock API:

### Step 1: Add to Mock API
1. Open `docs/tools/aws-mock-api-main/metrics-config.json`
2. Add metric entry:

```json
{
  "name": "AVG_QUEUE_ANSWER_TIME",
  "unit": "SECONDS", 
  "type": "random",
  "min": 15,
  "max": 45
}
```

**Mock API Value Types:**
- `"random"`: Random values between min/max
- `"ratio"`: Percentage values (0-100)
- `"increasing"`: Steadily increasing counter
- `"decreasing"`: Steadily decreasing values

### Step 2: Test Against Mock API
```bash
# Start mock API (in separate terminal)
cd docs/tools/aws-mock-api-main
node server.js

# Run exporter against mock API  
ENV=development ./bin/aws-connect-exporter \
  --config.file=configs/config.development.yml \
  --web.listen-port=9091 --no-advertise --log.level=debug

# Check metrics appear
curl -s http://localhost:9091/metrics | grep your_metric_name
```

## AWS Connect Metric Reference

### Real-time Metrics (30s-60s intervals)
| AWS Metric Name | Description | Unit | Grouping |
|-----------------|-------------|------|----------|
| `AGENTS_AVAILABLE` | Available agents | Count | QUEUE |
| `AGENTS_ON_CONTACT` | Agents handling contacts | Count | QUEUE |
| `CONTACTS_IN_QUEUE` | Contacts waiting | Count | QUEUE |
| `AVG_QUEUE_ANSWER_TIME` | Average wait time | Seconds | QUEUE |
| `SLOTS_ACTIVE` | Active agent slots | Count | QUEUE |

### Historical Metrics (5m+ intervals)
| AWS Metric Name | Description | Unit | Grouping |
|-----------------|-------------|------|----------|
| `CONTACTS_HANDLED` | Total handled contacts | Count | QUEUE |
| `CONTACTS_ABANDONED` | Abandoned contacts | Count | QUEUE |
| `SERVICE_LEVEL` | Service level % | Percent | QUEUE |
| `AVG_HANDLE_TIME` | Average handle time | Seconds | QUEUE |
| `AVG_ABANDONMENT_TIME` | Average abandonment time | Seconds | QUEUE |

## Troubleshooting New Metrics

### Common Issues

1. **Metric doesn't appear after restart**
   - Check AWS metric name spelling (case-sensitive)
   - Verify metric is available for your AWS Connect instance
   - Check exporter logs: `journalctl -u aws-connect-exporter -f`

2. **Values are always zero**
   - Historical metrics need historical data (24-48 hour delay)
   - Check if queue has activity during measurement period
   - Verify groupings and filters are correct

3. **Too many metrics/high cardinality**
   - Avoid using unique values in labels (agent IDs, timestamps)
   - Use filters to limit scope to specific queues
   - Consider separate collectors for different customer requirements

### Getting Help

When escalating metric addition issues:
1. **Customer requirement**: Exact metric needed and business use case
2. **AWS metric name**: Confirmed from AWS documentation/console
3. **Configuration snippet**: The YAML configuration attempted
4. **Error logs**: Exporter logs showing the issue
5. **AWS Connect context**: Instance details, queue names, historical data availability

# Wallboard Requirements to Configuration Mapping

## Quick Reference for Deployment Technicians

This guide helps translate customer wallboard requirements into AWS Connect exporter configuration. Use this when customers describe what they want to see on their dashboards/wallboards.

## Common Customer Requests → Configuration

### 🎯 "Show me current queue status"

**Customer says:** *"I want to see how many calls are waiting and how many agents are available"*

**Configuration:**
```yaml
collectors:
  generic_realtime:
    enabled: true
    interval: 30  # 30 seconds for real-time display
    collectors:
      - name: "wallboard_queue_status"
        metrics:
          - aws_name: "CONTACTS_IN_QUEUE"
            prometheus_name: "connect_calls_waiting"
            labels:
              display: "wallboard"
              metric_type: "current_status"
          - aws_name: "AGENTS_AVAILABLE" 
            prometheus_name: "connect_agents_available"
            labels:
              display: "wallboard"
              metric_type: "current_status"
          - aws_name: "AGENTS_ON_CONTACT"
            prometheus_name: "connect_agents_busy"
            labels:
              display: "wallboard" 
              metric_type: "current_status"
        groupings: ["QUEUE"]
        filters: []
```

### 📞 "Show me service level performance"

**Customer says:** *"I need to see if we're meeting our 80% in 20 seconds target"*

**Configuration:**
```yaml
collectors:
  generic_historical:
    enabled: true
    interval: 300  # 5 minutes for SLA metrics
    collectors:
      - name: "wallboard_sla_monitoring"
        metrics:
          - aws_name: "SERVICE_LEVEL"
            prometheus_name: "connect_service_level_20s_percent"
            labels:
              display: "wallboard"
              sla_target: "20s_80percent"
            threshold_ref: "sla_20_seconds"
        thresholds:
          - name: "sla_20_seconds"
            comparison: "LT"
            value: 20.0
        groupings: ["QUEUE"]
        filters: []
```

### ⏱️ "Show me average wait times"

**Customer says:** *"Display current average wait time for customers in queue"*

**Configuration:**
```yaml
collectors:
  generic_realtime:
    enabled: true
    interval: 30
    collectors:
      - name: "wallboard_wait_times"
        metrics:
          - aws_name: "AVG_QUEUE_ANSWER_TIME"
            prometheus_name: "connect_avg_wait_time_seconds"
            labels:
              display: "wallboard"
              metric_type: "wait_time"
          - aws_name: "OLDEST_CONTACT_AGE"
            prometheus_name: "connect_longest_wait_seconds"
            labels:
              display: "wallboard"
              metric_type: "wait_time"
        groupings: ["QUEUE"]
        filters: []
```

### 👥 "Show me agent status breakdown"

**Customer says:** *"I want to see how many agents are available, busy, in ACW, etc."*

**Configuration:**
```yaml
collectors:
  agent_realtime_status:
    enabled: true
    interval: 60  # 1 minute for agent status
  generic_realtime:
    enabled: true
    interval: 30
    collectors:
      - name: "wallboard_agent_breakdown"
        metrics:
          - aws_name: "AGENTS_AVAILABLE"
            prometheus_name: "connect_agents_available"
            labels:
              display: "wallboard"
              status_type: "available"
          - aws_name: "AGENTS_ON_CONTACT"
            prometheus_name: "connect_agents_on_calls"
            labels:
              display: "wallboard"
              status_type: "busy"
          - aws_name: "AGENTS_AFTER_CONTACT_WORK"
            prometheus_name: "connect_agents_in_acw"
            labels:
              display: "wallboard"
              status_type: "acw"
        groupings: ["QUEUE"]
        filters: []
```

### 📊 "Show me daily performance stats"

**Customer says:** *"Display today's call volumes, handled vs abandoned"*

**Configuration:**
```yaml
collectors:
  generic_historical:
    enabled: true
    interval: 300  # 5 minutes
    collectors:
      - name: "wallboard_daily_stats"
        metrics:
          - aws_name: "CONTACTS_HANDLED"
            prometheus_name: "connect_calls_handled_today"
            labels:
              display: "wallboard"
              metric_type: "daily_volume"
          - aws_name: "CONTACTS_ABANDONED"
            prometheus_name: "connect_calls_abandoned_today"
            labels:
              display: "wallboard"
              metric_type: "daily_volume"
          - aws_name: "CONTACTS_QUEUED"
            prometheus_name: "connect_calls_received_today"
            labels:
              display: "wallboard"
              metric_type: "daily_volume"
        groupings: ["QUEUE"]
        filters: []
```

## Wallboard Display Patterns

### Pattern 1: Real-time Operations Dashboard
**Customer Need:** *"Live view of current queue activity"*
- **Update Frequency:** 30 seconds
- **Collector:** `generic_realtime`
- **Key Metrics:** CONTACTS_IN_QUEUE, AGENTS_AVAILABLE, AVG_QUEUE_ANSWER_TIME

### Pattern 2: SLA Compliance Monitor  
**Customer Need:** *"Track service level targets throughout the day"*
- **Update Frequency:** 5 minutes
- **Collector:** `generic_historical`
- **Key Metrics:** SERVICE_LEVEL (with thresholds)

### Pattern 3: Agent Performance Display
**Customer Need:** *"Show agent productivity and availability"*
- **Update Frequency:** 60 seconds
- **Collector:** `agent_realtime_status` + `generic_realtime`
- **Key Metrics:** Agent status breakdown, individual agent performance

### Pattern 4: Executive Summary Board
**Customer Need:** *"High-level daily/weekly performance overview"*
- **Update Frequency:** 5-15 minutes
- **Collector:** `generic_historical`
- **Key Metrics:** Daily volumes, service levels, trends

## Queue-Specific Wallboards

### High-Priority Queue Monitoring
**Customer says:** *"I only want to see metrics for VIP and escalation queues"*

```yaml
filters:
  - key: "QUEUES"
    values: 
      - "arn:aws:connect:region:account:instance/instance-id/queue/vip-queue-id"
      - "arn:aws:connect:region:account:instance/instance-id/queue/escalation-queue-id"
```

### Department-Specific Views
**Customer says:** *"Sales team wants their own dashboard, separate from support"*

```yaml
# Sales Team Configuration
collectors:
  - name: "sales_wallboard"
    metrics:
      - aws_name: "CONTACTS_IN_QUEUE"
        prometheus_name: "connect_sales_calls_waiting"
        labels:
          department: "sales"
          display: "wallboard"
    filters:
      - key: "QUEUES"
        values: ["arn:aws:connect:region:account:instance/instance-id/queue/sales-queue-id"]

# Support Team Configuration  
collectors:
  - name: "support_wallboard"
    metrics:
      - aws_name: "CONTACTS_IN_QUEUE"
        prometheus_name: "connect_support_calls_waiting"
        labels:
          department: "support"
          display: "wallboard"
    filters:
      - key: "QUEUES"
        values: ["arn:aws:connect:region:account:instance/instance-id/queue/support-queue-id"]
```

## Multi-Channel Wallboards

### Channel-Specific Display
**Customer says:** *"Show voice, chat, and email metrics separately"*

```yaml
collectors:
  - name: "voice_wallboard"
    metrics:
      - aws_name: "CONTACTS_IN_QUEUE"
        prometheus_name: "connect_voice_calls_waiting"
        labels:
          channel: "voice"
          display: "wallboard"
    filters:
      - key: "CHANNELS"
        values: ["VOICE"]
  
  - name: "chat_wallboard" 
    metrics:
      - aws_name: "CONTACTS_IN_QUEUE"
        prometheus_name: "connect_chat_sessions_waiting"
        labels:
          channel: "chat"
          display: "wallboard"
    filters:
      - key: "CHANNELS"
        values: ["CHAT"]
```

## Wallboard Label Strategy

### Consistent Labeling for Dashboards
Use these labels to make wallboard creation easier:

```yaml
labels:
  display: "wallboard"           # Identifies wallboard metrics
  metric_type: "current_status"  # Groups related metrics
  priority: "high"              # Indicates importance
  department: "sales"           # Department/team identifier
  channel: "voice"              # Communication channel
  sla_target: "20s_80percent"   # SLA requirement identifier
```

## Quick Deployment Checklist

### Pre-Deployment Questions for Customer
- [ ] **Which queues?** Get exact queue names/ARNs
- [ ] **What time periods?** Real-time (30s) or historical (5m+)?  
- [ ] **SLA targets?** Specific thresholds (e.g., "80% in 20 seconds")
- [ ] **Separate channels?** Voice/Chat/Task displayed separately?
- [ ] **Department splits?** Different teams need separate views?
- [ ] **Update frequency?** How often should wallboard refresh?

### Post-Deployment Validation
```bash
# 1. Check metrics appear with wallboard labels
curl -s http://localhost:9090/metrics | grep 'display="wallboard"'

# 2. Verify update frequency matches customer needs
curl -s http://localhost:9090/metrics | grep connect_calls_waiting | head -3

# 3. Confirm queue filtering works correctly  
curl -s http://localhost:9090/metrics | grep queue_name | sort | uniq
```

## Common Wallboard Metric Names

Use these standardized Prometheus metric names for consistency:

### Real-time Status Metrics
- `connect_calls_waiting` - Current queue depth
- `connect_agents_available` - Available agents
- `connect_agents_busy` - Agents on calls
- `connect_avg_wait_time_seconds` - Current average wait
- `connect_longest_wait_seconds` - Oldest contact age

### Performance Metrics  
- `connect_service_level_Xs_percent` - SLA compliance (X = threshold)
- `connect_calls_handled_today` - Daily handled volume
- `connect_calls_abandoned_today` - Daily abandoned volume
- `connect_avg_handle_time_seconds` - Average call duration

### Agent Metrics
- `connect_agent_status` - Individual agent states
- `connect_agents_in_acw` - After contact work count
- `connect_agents_logged_in` - Total logged in agents

## Troubleshooting Wallboard Deployments

### "Wallboard shows zero values"
1. **Check real-time vs historical**: Wallboards usually need real-time data
2. **Verify queue activity**: Empty queues show zero legitimately  
3. **Confirm update intervals**: 30s for real-time, 300s+ for historical

### "Wrong queues showing"
1. **Check queue ARNs**: Must be exact matches
2. **Verify filters**: Ensure QUEUES filter includes right queues
3. **Test without filters**: See if all queues appear first

### "SLA metrics not working"
1. **Add thresholds**: SERVICE_LEVEL requires threshold configuration
2. **Check historical data**: SLA metrics need 24-48h of historical data
3. **Verify comparison operator**: Usually "LT" for "within X seconds"
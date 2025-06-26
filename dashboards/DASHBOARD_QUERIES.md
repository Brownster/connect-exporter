# Dashboard Query Reference

This document provides the key PromQL queries used in the Kensington dashboards for reference and customization.

## 🚀 Real-time Operations Dashboard Queries

### Queue Overview Panel
```promql
# Total available slots across all queues
sum(aws_connect_exporter_realtime_queue_connect_queue_slots_available)

# Total non-productive agents
sum(aws_connect_exporter_realtime_queue_connect_queue_agents{state="NonProductive"})
```

### Agent Status Distribution (Pie Chart)
```promql
# Agent counts by status
count by(status_name) (aws_connect_exporter_agent_realtime_status_connect_agent_status_state)
```

### Queue Performance Table
```promql
# Available slots by queue and channel
aws_connect_exporter_realtime_queue_connect_queue_slots_available

# Non-productive agents by queue
aws_connect_exporter_realtime_queue_connect_queue_agents{state="NonProductive"}
```

### Operational Alerts
```promql
# Alert when available slots are low (< 3)
sum(aws_connect_exporter_realtime_queue_connect_queue_slots_available) < 3

# Alert when non-productive agents are high (> 5)
sum(aws_connect_exporter_realtime_queue_connect_queue_agents{state="NonProductive"}) > 5
```

## 📈 Daily Performance Analytics Dashboard Queries

### Daily Overview Panel
```promql
# Total calls handled today
sum(aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total)

# Total contacts abandoned today
sum(aws_connect_exporter_historical_queue_connect_historical_contacts_abandoned_total)

# Overall abandonment rate percentage
sum(aws_connect_exporter_historical_queue_connect_historical_contacts_abandoned_total) /
(
    sum(aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total) +
    sum(aws_connect_exporter_historical_calls_connect_historical_contacts_abandoned_total)
) * 100
```

### Queue Performance Analysis (Bar Gauge)
```promql
# Abandonment rate by queue
aws_connect_exporter_historical_queue_connect_historical_contacts_abandoned_total /
(aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total +
 aws_connect_exporter_historical_calls_connect_historical_contacts_abandoned_total) * 100
```

### Channel Distribution (Pie Chart)
```promql
# Contact volume by channel
sum by (channel) (
    aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total +
    aws_connect_exporter_historical_calls_connect_historical_contacts_abandoned_total
)
```

### Performance KPIs
```promql
# Average abandonment rate across all queues
avg(aws_connect_exporter_historical_queue_connect_historical_contacts_abandoned_total /
    (aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total +
     aws_connect_exporter_historical_calls_connect_historical_contacts_abandoned_total) * 100)

# Total daily volume
sum(aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total)
```

## 🔧 Advanced Custom Queries

### Service Level Calculation (if handled metrics become available)
```promql
# Service level percentage (when handled metrics are added)
(sum(aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total) /
 (sum(aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total) +
  sum(aws_connect_exporter_historical_calls_connect_historical_contacts_abandoned_total))) * 100
```

### Agent Utilization (when more agent metrics are available)
```promql
# Agent utilization percentage
(sum(aws_connect_exporter_realtime_queue_connect_queue_agents{state="OnContact"}) /
 sum(aws_connect_exporter_realtime_queue_connect_queue_agents{state="Staffed"})) * 100
```

### Queue Efficiency Score
```promql
# Custom efficiency score (lower abandonment = higher efficiency)
100 - (aws_connect_exporter_historical_calls_connect_calls_abandoned_total /
       (aws_connect_exporter_historical_calls_connect_calls_handled_total +
        aws_connect_exporter_historical_calls_connect_calls_abandoned_total) * 100)
```

## 📊 Sample Data from Mock API

Based on the enhanced mock API, here's what typical query results look like:

### Real-time Metrics
```
aws_connect_exporter_realtime_queue_slots_available{channel="VOICE",queue_id="queue-sales-01",queue_name="Sales Queue",queue_type="STANDARD",region="eu-west-2"} 3
aws_connect_exporter_agent_state_agents{queue="Sales Queue",state="NonProductive"} 1
aws_connect_exporter_agent_realtime_status_agent_state{agent_id="user-001",username="alice.smith",routing_profile_id="",queue_id="",status_id="status-001",status_name="Available",status_type="unknown",region="eu-west-2"} 1
```

### Daily Performance Metrics
```
aws_connect_exporter_historical_calls_connect_calls_handled_total{queue_id="queue-sales-01",queue_name="Sales Queue",queue_type="STANDARD",direction="inbound",region="eu-west-2"} 40
aws_connect_exporter_historical_calls_connect_calls_abandoned_total{queue_id="queue-sales-01",queue_name="Sales Queue",queue_type="STANDARD",region="eu-west-2"} 3
```

## 🎯 Dashboard Thresholds

### Abandonment Rate Thresholds
```yaml
Green (Excellent): < 5%
Yellow (Good): 5% - 10%
Orange (Warning): 10% - 15%
Red (Critical): > 15%
```

### Available Slots Thresholds
```yaml
Red (Critical): < 3 slots
Yellow (Warning): 3-4 slots
Green (Good): 5+ slots
```

### Agent Status Color Coding
```yaml
Available: Green
Busy/On Contact: Blue
After Contact Work: Yellow
Custom Break: Orange
```

## 💡 Query Optimization Tips

1. **Use Recording Rules** for complex calculations that are used multiple times
2. **Limit Time Ranges** in queries to improve performance
3. **Use Rate Functions** for counters that reset (if any become available)
4. **Aggregate Early** in queries to reduce data points

### Example Recording Rule (prometheus.yml)
```yaml
groups:
  - name: kensington_contact_center
    rules:
      - record: kensington:abandonment_rate
        expr: aws_connect_exporter_historical_calls_connect_calls_abandoned_total /
              (aws_connect_exporter_historical_calls_connect_calls_handled_total +
               aws_connect_exporter_historical_calls_connect_calls_abandoned_total) * 100
      
      - record: kensington:total_volume_today
        expr: sum(aws_connect_exporter_historical_calls_connect_calls_handled_total)
```

This reference enables teams to customize dashboards, create alerts, and build additional monitoring based on Kensington's specific operational requirements.
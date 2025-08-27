# AWS Connect Exporter Dashboard Enhancements

## Summary
Updated the Grafana dashboard to leverage the new enhanced labeling system with agent caching and improved metric structures.

## Key Enhancements

### 1. Enhanced Agent Status Panel
**Old Query:**
```
aws_connect_exporter_agent_realtime_status_connect_agent_status_state{status_type!="OFFLINE"}
```

**New Enhanced Query:**
```
aws_connect_exporter_agent_realtime_status_connect_agent_status_state
```

**New Labels Available:**
- `agent_id` - Unique agent identifier
- `username` - Agent username (e.g., alice.smith)
- `first_name` - Agent first name (e.g., Alice)
- `last_name` - Agent last name (e.g., Smith) 
- `routing_profile_id` - Agent routing profile
- `queue_name` - Queue name with full details
- `status_name` - Current agent status
- `region` - AWS region

### 2. Enhanced Queue Metrics
**Old Query:**
```
sum by(queue_name, channel, state) (aws_connect_exporter_generic_realtime_generic_metric{metric_name="connect_queue_agents"})
```

**New Enhanced Query:**
```
sum by(state) (aws_connect_exporter_realtime_queue_connect_queue_agents)
```

**New Labels Available:**
- `region` - AWS region
- `queue_id` - Queue identifier
- `queue_name` - Full queue name
- `queue_type` - Queue type (STANDARD/AGENT)
- `queue_arn` - Queue ARN
- `channel` - Channel type (VOICE/CHAT/TASK)
- `state` - Agent state (Available/OnContact/etc.)

### 3. Enhanced Historical Metrics with Direction Tracking
**Old Query:**
```
sum by(channel,queue_name) (aws_connect_exporter_generic_historical_generic_historical_metric{metric_name="connect_historical_contacts_abandoned_total"})
```

**New Enhanced Query:**
```
sum by(queue_name, channel, direction) (aws_connect_exporter_historical_calls_connect_historical_contacts_handled_total)
```

**New Labels Available:**
- `direction` - Call direction (inbound/outbound) - **NEW**
- `region` - AWS region
- `queue_id`, `queue_name`, `queue_type`, `queue_arn` - Full queue details
- `channel` - Channel type

### 4. Enhanced Agent Status Distribution
**New Query:**
```
count by(status_name, routing_profile_id) (aws_connect_exporter_agent_realtime_status_connect_agent_status_state)
```

**Benefits:**
- Shows agent distribution by routing profile
- Enables filtering by specific routing profiles
- Better visibility into team assignments

## Files Changed

1. **dashboards/realtime-ops-enhanced-dashboard.json** - New enhanced dashboard
2. **misc/realtime-ops-dashboard-backup.json** - Backup of original dashboard
3. **DASHBOARD_ENHANCEMENTS.md** - This documentation

## Dashboard Panels

1. **Enhanced Agent Status Table** - Shows agents with usernames, names, and profiles
2. **Queue Agents by Status Pie Chart** - Enhanced with full queue labeling
3. **Historical Contacts Timeline** - Now tracks inbound vs outbound separately
4. **Average Handle Time** - Enhanced with full queue and channel details
5. **Agent Status by Routing Profile** - New donut chart showing team distribution

## Grafana Import Instructions

1. Navigate to Grafana → Dashboards → Import
2. Upload `dashboards/realtime-ops-enhanced-dashboard.json`
3. Configure Prometheus datasource (uid: P6091CB58F7FEFEA5)
4. Save and view the enhanced dashboard

## Benefits of Enhanced Dashboard

- **Better Agent Visibility** - See usernames, names, and routing profiles
- **Direction Tracking** - Separate inbound vs outbound call metrics
- **Improved Filtering** - Enhanced labels enable better dashboard filtering
- **Team Management** - Routing profile breakdown for team insights
- **Consistent Labeling** - All metrics now follow consistent schema
- **Future-Ready** - Dashboard structure supports additional enhancements

## Testing

All queries tested against live enhanced exporter:
- ✅ Agent status metrics with usernames
- ✅ Queue metrics with enhanced labels  
- ✅ Historical metrics with direction tracking
- ✅ 21 total metrics being generated
- ✅ 13 metrics with enhanced labeling
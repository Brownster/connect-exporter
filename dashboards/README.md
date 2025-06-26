# Kensington Contact Center Dashboards

This directory contains two comprehensive Grafana dashboards designed specifically for Kensington's contact center operations, built on the AWS Connect Exporter metrics.

## 📊 Dashboard Overview

### 1. **Real-time Operations Dashboard** (`kensington-realtime-operations.json`)
**Purpose:** Live operational monitoring and immediate response  
**Refresh Rate:** 30 seconds  
**Time Range:** Last 1 hour  

**Key Features:**
- 📞 **Queue Overview** - Real-time available slots and non-productive agent counts
- 👥 **Agent Status Distribution** - Live pie chart showing agent availability 
- 📋 **Queue Performance by Channel** - Table view of VOICE/CHAT/TASK performance
- 📈 **Real-time Metrics Trend** - Time series of key operational metrics
- ⚠️ **Operational Alerts** - Automatic alerting for low availability or high non-productive time

### 2. **Daily Performance Analytics Dashboard** (`kensington-daily-performance.json`)
**Purpose:** Strategic performance analysis and trend monitoring  
**Refresh Rate:** 5 minutes  
**Time Range:** Last 7 days  

**Key Features:**
- 📊 **Daily Performance Overview** - Total queued, abandoned, and abandonment rate
- 📈 **7-Day Contact Volume Trend** - Historical trend analysis
- 🎯 **Queue Performance Analysis** - Abandonment rate by queue with color coding
- 📋 **Performance Table** - Detailed breakdown by queue and channel
- 📞 **Channel Distribution** - Contact volume split across VOICE/CHAT/TASK
- 🎯 **Performance KPIs** - Key performance indicators and benchmarks

## 🎯 Metrics Mapping

### Real-time Dashboard Metrics:
```prometheus
# Queue availability and agent productivity
aws_connect_exporter_realtime_queue_slots_available
aws_connect_exporter_realtime_queue_agents_non_productive

# Live agent status distribution
aws_connect_exporter_agent_realtime_status_agent_state
```

### Daily Performance Dashboard Metrics:
```prometheus
# Daily contact volumes and outcomes
aws_connect_exporter_historical_queue_contacts_queued_today
aws_connect_exporter_historical_queue_contacts_abandoned_today

# Calculated KPIs
abandonment_rate = abandoned_today / queued_today * 100
```

## 🚀 Installation Instructions

### Prerequisites
- Grafana (v8.0+)
- Prometheus data source configured
- AWS Connect Exporter running and collecting metrics

### Step 1: Import Dashboards
1. Open Grafana UI
2. Navigate to **+ (Create)** → **Import**
3. Upload the JSON files or paste the JSON content
4. Configure the Prometheus data source
5. Save the dashboard

### Step 2: Configure Data Source
Ensure your Prometheus data source is configured to scrape from your AWS Connect Exporter:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'aws-connect-exporter'
    static_configs:
      - targets: ['localhost:9091']  # Your exporter port
    scrape_interval: 30s
```

### Step 3: Set Up Alerting (Optional)
The real-time dashboard includes alert conditions. To enable Grafana alerting:
1. Go to **Alerting** → **Alert Rules**
2. Create rules based on the dashboard queries
3. Configure notification channels (Slack, email, etc.)

## 📋 Dashboard Customization

### Thresholds and Colors
You can adjust the color thresholds to match Kensington's SLAs:

**Available Slots Thresholds:**
- 🔴 Red: < 3 slots (Critical)
- 🟡 Yellow: 3-4 slots (Warning)  
- 🟢 Green: 5+ slots (Good)

**Abandonment Rate Thresholds:**
- 🟢 Green: < 5% (Excellent)
- 🟡 Yellow: 5-10% (Good)
- 🟠 Orange: 10-15% (Warning)
- 🔴 Red: > 15% (Critical)

### Adding Custom Panels
To add additional metrics or modify existing panels:

1. **Edit Dashboard** → **Add Panel**
2. **Configure Query:** Use PromQL syntax with available metrics
3. **Set Visualization:** Choose appropriate chart type
4. **Apply Formatting:** Set units, colors, and thresholds

Example custom query for average wait time:
```promql
aws_connect_exporter_realtime_queue_oldest_contact_age_seconds
```

## 🎨 Dashboard Previews

### Real-time Operations Dashboard
```
┌─────────────────────────────────────────────────────────┐
│ 📞 Queue Overview                                       │
│ Available Slots: 5    Non-Productive: 2               │
├──────────────────────┬──────────────────────────────────┤
│ 👥 Agent Status      │ 📋 Queue Performance            │
│ Distribution         │ Sales Queue (VOICE): 5 slots    │
│ [Pie Chart]          │ Support Queue (VOICE): 3 slots  │
│                      │ Chat Queue (CHAT): 8 slots      │
├──────────────────────┴──────────────────────────────────┤
│ 📈 Real-time Queue Metrics Trend                       │
│ [Time Series Chart - Last 1 Hour]                      │
├─────────────────────────────────────────────────────────┤
│ ⚠️ Operational Alerts                                   │
│ All Good ✅                                            │
└─────────────────────────────────────────────────────────┘
```

### Daily Performance Analytics Dashboard
```
┌─────────────────────────────────────────────────────────┐
│ 📊 Daily Performance Overview                          │
│ Queued: 43   Abandoned: 3   Rate: 7.0%               │
├──────────────────────┬──────────────────────────────────┤
│ 📈 7-Day Volume      │ 🎯 Queue Performance            │
│ Trend                │ Analysis                         │
│ [Line Chart]         │ [Bar Gauge - Abandonment Rates] │
├──────────────────────┴──────────────────────────────────┤
│ 📋 Daily Performance by Queue and Channel              │
│ [Detailed Table with Color Coding]                     │
├──────────────────────┬──────────────────────────────────┤
│ 📞 Channel           │ 🎯 Performance KPIs             │
│ Distribution         │ Avg Abandonment: 7.0%          │
│ [Donut Chart]        │ Total Volume: 43                │
└──────────────────────┴──────────────────────────────────┘
```

## 🔧 Troubleshooting

### No Data Appearing
1. **Check Exporter Status:** Ensure AWS Connect Exporter is running
2. **Verify Metrics:** `curl http://localhost:9091/metrics | grep aws_connect`
3. **Prometheus Scraping:** Check Prometheus targets page
4. **Data Source:** Verify Grafana data source connection

### Metrics Not Updating
1. **Refresh Rate:** Check dashboard refresh settings
2. **Time Range:** Ensure time range includes recent data
3. **Query Issues:** Test PromQL queries in Prometheus UI

### Performance Issues
1. **Reduce Refresh Rate:** Change from 30s to 1m for less load
2. **Limit Time Range:** Use shorter time windows for real-time dashboard
3. **Optimize Queries:** Use recording rules for complex calculations

## 📞 Support and Customization

These dashboards are designed to be starting points for Kensington's specific needs. For additional customization or support:

1. **Metric Reference:** See `../README.md` for complete metrics list
2. **Mock API Testing:** Use enhanced mock API for dashboard development
3. **Configuration:** Adjust thresholds and colors to match SLAs

The dashboards provide comprehensive visibility into contact center operations while remaining lightweight and responsive for real-time monitoring.
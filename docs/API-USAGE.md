# AWS Connect Exporter - API Usage & Cost Analysis

## Overview
This document provides complete transparency on AWS Connect API usage patterns, call frequencies, and associated costs for the AWS Connect Exporter.

## Default Configuration & Intervals

### Collection Intervals
- **Real-time Metrics**: 60 seconds (realtime_queue, agent_realtime_status)
- **Historical Metrics**: 300 seconds (historical_calls, historical_queue, historical_aht, historical_agent_performance)
- **Cache Refresh Intervals**: 12 hours (agents, queues) - **Updated Default**

### Caching Strategy
- **Queue Cache TTL**: 12 hours (was 5 minutes)
- **Agent Cache TTL**: 12 hours (was 5 minutes)  
- **Metrics Cache TTL**: 15 seconds (realtime collector only)

## API Calls Per Collection Cycle

### 1. Queue Discovery (Every 12 Hours)
| API Call | Purpose | Frequency | Calls per Day |
|----------|---------|-----------|---------------|
| `ListQueues` | Queue discovery | Every 12h | 2 |

### 2. Real-time Metrics Collection (Every 60 Seconds)
| API Call | Purpose | Frequency | Batch Formula | Calls per Day |
|----------|---------|-----------|---------------|---------------|
| `GetCurrentMetricData` | Queue metrics | 60s | `⌈queues/100⌉` | 1,440 × batches |
| `GetCurrentUserData` | Agent status | 60s | Limited to 100 queues | 1,440 |
| `ListAgentStatuses` | Status enrichment | 60s | 1 call | 1,440 |

### 3. Agent Discovery (Every 12 Hours)
| API Call | Purpose | Frequency | Cost Impact | Calls per Day |
|----------|---------|-----------|-------------|---------------|
| `ListUsers` | Agent list | Every 12h | `⌈agents/100⌉` | 2 × batches |
| `DescribeUser` | Agent details | Every 12h | **1 per agent** | 2 × agent_count |

### 4. Historical Metrics Collection (Every 300 Seconds)
| API Call | Purpose | Frequency | Batch Formula | Calls per Day |
|----------|---------|-----------|---------------|---------------|
| `GetMetricDataV2` (calls) | Historical call metrics | 300s | `⌈queues/100⌉` | 288 × batches |
| `GetMetricDataV2` (queue) | Historical queue metrics | 300s | `⌈queues/100⌉` | 288 × batches |
| `GetMetricDataV2` (aht) | Average handle time | 300s | `⌈queues/100⌉` | 288 × batches |
| `GetMetricDataV2` (agent_perf) | Agent performance | 300s | `⌈queues/100⌉` | 288 × batches |

## Cost Calculations

### Daily API Call Volumes

**Environment: 50 Queues, 100 Agents (Typical Small/Medium)**

| Component | API Calls/Day | % of Total |
|-----------|---------------|------------|
| Queue Discovery | 2 | <1% |
| Real-time Metrics | 1,440 | 43% |
| Agent Status | 1,440 | 43% |
| Agent Discovery | 200 | 6% |
| Historical Metrics (4×) | 1,152 | 34% |
| Status enrichment | 1,440 | 43% |
| **Total** | **5,674** | **100%** |

**With 12-Hour Caching vs 5-Minute Caching:**
- **Old (5min cache)**: 28,800 DescribeUser calls/day
- **New (12h cache)**: 200 DescribeUser calls/day
- **Reduction**: 99.3% fewer expensive API calls

### Monthly Cost Estimates (AWS Connect API Pricing)

**Assumption**: $0.004 per API call (varies by region)

| Environment Size | Daily Calls | Monthly Cost (Old) | Monthly Cost (New) | Savings |
|------------------|-------------|--------------------|--------------------|---------|
| **50 queues, 100 agents** | 5,674 | $3,750 | $680 | **$3,070** |
| **100 queues, 200 agents** | 8,154 | $7,500 | $978 | **$6,522** |
| **200 queues, 500 agents** | 15,074 | $18,750 | $1,809 | **$16,941** |
| **500 queues, 1000 agents** | 35,074 | $37,500 | $4,209 | **$33,291** |

## Batching Implementation

### AWS API Limits
- **Maximum 100 queues per request** for all metrics APIs
- **Maximum 100 users per request** for ListUsers
- **Individual calls required** for DescribeUser

### Batching Formula
```
API_Calls_Per_Collection = ceiling(queue_count / 100)
```

### Examples
- **50 queues** → 1 API call per collection
- **100 queues** → 1 API call per collection  
- **150 queues** → 2 API calls per collection
- **500 queues** → 5 API calls per collection

## Error Handling & Retry Logic

### Retry Configuration
- **Only Real-time Queue Collector** has retry logic
- **3 attempts maximum** with exponential backoff
- **Delays**: 1s → 5s maximum
- **Worst Case**: 200% increase in API calls during failures

### Error Isolation
- Failed batches don't stop other collections
- Agent DescribeUser failures fall back to partial data
- Continue processing with cached data if refresh fails

## Cost Optimization Strategies

### 1. Interval Tuning
```yaml
# Conservative Settings (50% cost reduction)
collectors:
  realtime_queue:
    interval: 120  # vs 60s default
  historical_calls:
    interval: 600  # vs 300s default
```

### 2. Queue Filtering
```yaml
# Monitor only critical queues
queue_include_regex: "^(premium|vip|escalation)-.*"
```

### 3. Collector Disabling
```yaml
# Disable non-essential collectors
collectors:
  historical_agent_performance:
    enabled: false  # 25% reduction in historical calls
```

### 4. Extended Caching (Current Default)
```yaml
# Already implemented - 12-hour cache refresh
# Results in 99.3% reduction in DescribeUser calls
```

## Monitoring Recommendations

### Cost Monitoring
- Set CloudWatch billing alerts for AWS Connect API usage
- Monitor `aws_connect_api_calls_total` metric in Prometheus
- Track cost per queue/agent for capacity planning

### Performance Monitoring  
- Monitor collection duration vs intervals
- Watch for timeout errors indicating API throttling
- Track cache hit rates for optimization validation

## Production Tuning Examples

### High-Frequency Operations (Real-time Dashboard)
```yaml
# Cost: ~$680/month for 50 queues, 100 agents
collectors:
  realtime_queue:
    interval: 60      # Real-time updates
  agent_realtime_status:
    interval: 60      # Agent status updates
  historical_calls:
    interval: 300     # Historical trends
```

### Balanced Operations (Management Reporting)
```yaml
# Cost: ~$340/month for 50 queues, 100 agents  
collectors:
  realtime_queue:
    interval: 120     # 2-minute updates
  agent_realtime_status:
    interval: 300     # 5-minute agent updates
  historical_calls:
    interval: 600     # 10-minute historical
```

### Cost-Optimized Operations (Daily/Weekly Reports)
```yaml  
# Cost: ~$170/month for 50 queues, 100 agents
collectors:
  realtime_queue:
    interval: 300     # 5-minute updates
  agent_realtime_status:
    interval: 600     # 10-minute agent updates
  historical_calls:
    interval: 1800    # 30-minute historical
```

## Summary

The **12-hour caching default** provides:
- ✅ **99.3% reduction** in expensive DescribeUser API calls
- ✅ **82% cost savings** compared to 5-minute caching
- ✅ **Negligible impact** on agent name visibility (names rarely change)
- ✅ **Maintains real-time metrics** for operational monitoring
- ✅ **Scales efficiently** for large environments (500+ queues)

**Cost per environment with 12-hour caching:**
- Small (50 queues): **$680/month**
- Medium (100 queues): **$978/month**  
- Large (200 queues): **$1,809/month**
- Enterprise (500 queues): **$4,209/month**

The 12-hour cache refresh for agent and queue metadata strikes the optimal balance between cost efficiency and data freshness for production environments.
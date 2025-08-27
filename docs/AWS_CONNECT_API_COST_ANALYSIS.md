# AWS Connect Exporter - API Cost Transparency Analysis

## Executive Summary

This document provides a comprehensive analysis of AWS Connect API calls made by the Connect Exporter, including frequency, batching behavior, caching patterns, and cost optimization recommendations. The analysis covers all collectors, their scraping intervals, and provides hourly/daily API call volume calculations.

## Table of Contents

1. [API Calls Overview](#api-calls-overview)
2. [Collector Analysis](#collector-analysis)
3. [Caching Behavior](#caching-behavior)
4. [API Call Volume Calculations](#api-call-volume-calculations)
5. [Cost Optimization Recommendations](#cost-optimization-recommendations)
6. [Tuning Parameters](#tuning-parameters)

---

## API Calls Overview

### AWS Connect APIs Used

The exporter makes calls to the following AWS Connect APIs:

| API Call | Purpose | Frequency | Batching | Retry Logic |
|----------|---------|-----------|----------|-------------|
| `ListQueues` | Queue discovery and caching | Every 5m (default) | Paginated | No |
| `ListUsers` | Agent discovery for caching | Every 5m (default) | Paginated (100/batch) | No |
| `DescribeUser` | Detailed agent info enrichment | Per agent, every 5m | 1 call per agent | No |
| `GetCurrentMetricData` | Real-time queue metrics | Every 60s (default) | Max 100 queues/batch | Yes (3 attempts) |
| `GetCurrentUserData` | Real-time agent status | Every 60s (default) | Max 100 queues filter | No |
| `ListAgentStatuses` | Agent status enrichment | Per realtime status call | No batching | No |
| `GetMetricDataV2` | Historical metrics | Every 300s (default) | Max 100 queues/batch | No |

---

## Collector Analysis

### 1. Realtime Queue Collector

**File**: `/collectors/realtime_queue_collector.go`

**Configuration**:
```yaml
realtime_queue:
  enabled: true
  interval: 60                    # seconds (1 minute)
  metrics_cache_ttl: 15s         # Internal caching
  queue_cache_ttl: 5m            # Not used in this collector
  queue_refresh_interval: 5m     # Queue discovery frequency
```

**API Calls per Collection Cycle**:
- `GetCurrentMetricData`: `ceiling(queue_count / 100)` calls
- Queue discovery via `ListQueues`: Every 5 minutes (shared cache)

**Metrics Collected**:
- 10 different current metrics per queue (AGENTS_AVAILABLE, CONTACTS_IN_QUEUE, etc.)
- Grouped by queue and channel (VOICE, CHAT, TASK)

**Batching**: Yes, processes up to 100 queues per API call
**Retry Logic**: Yes, 3 attempts with exponential backoff (1s → 5s max)
**Internal Caching**: 15-second TTL to avoid duplicate API calls within collection window

### 2. Agent Realtime Status Collector

**File**: `/collectors/agent_realtime_status_collector.go`

**Configuration**:
```yaml
agent_realtime_status:
  enabled: true
  interval: 60                   # seconds (1 minute)
```

**API Calls per Collection Cycle**:
- `GetCurrentUserData`: 1 call (limited to first 100 queues)
- `ListAgentStatuses`: 1 call for status enrichment
- Agent discovery via `ListUsers` + `DescribeUser`: Every 5 minutes (shared cache)

**Limitation**: Currently only processes first 100 queues due to incomplete batching implementation

### 3. Agent Status Collector

**File**: `/collectors/agent_status_collector.go`

**Configuration**:
```yaml
agent_status:
  enabled: true
  interval: 300                  # seconds (5 minutes)
```

**API Calls per Collection Cycle**:
- `ListAgentStatuses`: `ceiling(agent_statuses / 100)` calls (paginated)

**Batching**: Uses pagination with MaxResults=100

### 4. Historical Queue Collector

**File**: `/collectors/historical_queue_collector.go`

**Configuration**:
```yaml
historical_queue:
  enabled: true
  interval: 300                  # seconds (5 minutes)
```

**API Calls per Collection Cycle**:
- `GetMetricDataV2`: `ceiling(queue_count / 100)` calls
- Collects metrics since midnight UTC

**Metrics Collected**:
- CONTACTS_ABANDONED
- CONTACTS_HANDLED (with INBOUND filter)
- CONTACTS_HANDLED (with OUTBOUND/CALLBACK/API filter)

### 5. Historical Calls Collector

**File**: `/collectors/historical_calls_collector.go`

**Configuration**:
```yaml
historical_calls:
  enabled: true
  interval: 300                  # seconds (5 minutes)
```

**API Calls per Collection Cycle**:
- `GetMetricDataV2`: `ceiling(queue_count / 100)` calls

**Metrics Collected**:
- CONTACTS_HANDLED
- CONTACTS_HANDLED_OUTBOUND
- CONTACTS_ABANDONED

### 6. Historical Average Handle Time (AHT) Collector

**File**: `/collectors/historical_avg_time_collector.go`

**Configuration**:
```yaml
historical_aht:
  enabled: true
  interval: 300                  # seconds (5 minutes)
```

**API Calls per Collection Cycle**:
- `GetMetricDataV2`: `ceiling(queue_count / 100)` calls for AVG_HANDLE_TIME

### 7. Historical After Contact Work (ACW) Collector

**Configuration**:
```yaml
historical_acw:
  enabled: true
  interval: 300                  # seconds (5 minutes)
```

**API Calls per Collection Cycle**:
- `GetMetricDataV2`: `ceiling(queue_count / 100)` calls for AVG_AFTER_CONTACT_WORK_TIME

---

## Caching Behavior

### Queue Discovery Cache

**Implementation**: `QueueManager` in `/connectors/queue_manager.go`

- **Refresh Interval**: 5 minutes (default)
- **API Calls**: `ListQueues` with pagination
- **Shared Usage**: All collectors use the same cached queue list
- **Regex Filtering**: Optional queue name filtering to reduce scope

### Agent Discovery Cache

**Implementation**: `AgentManager` in `/connectors/agent_manager.go`

- **Refresh Interval**: 5 minutes (default, same as queue refresh)
- **API Calls**: 
  - `ListUsers`: Paginated, 100 users per call
  - `DescribeUser`: 1 call per agent for detailed information
- **Shared Usage**: Agent realtime status collector uses this cache

### Metrics Cache (Realtime Queue Only)

- **TTL**: 15 seconds (configurable via `metrics_cache_ttl`)
- **Purpose**: Prevents duplicate API calls if multiple scrapes happen within TTL window
- **Implementation**: In-memory cache with timestamp validation

---

## API Call Volume Calculations

### Base Assumptions

- **Queues**: 50 (typical medium-sized contact center)
- **Agents**: 100 (typical medium-sized contact center)
- **Agent Statuses**: 10 (typical AWS Connect instance)

### Default Configuration API Calls

#### Per Minute (60-second intervals):
```
Realtime Queue Metrics:
- GetCurrentMetricData: ceiling(50/100) = 1 call

Agent Realtime Status:
- GetCurrentUserData: 1 call
- ListAgentStatuses: 1 call
Total per minute: 3 calls
```

#### Per 5 Minutes (300-second intervals):
```
Shared Caches (every 5 minutes):
- ListQueues: ~1 call (small instances)
- ListUsers: ceiling(100/100) = 1 call  
- DescribeUser: 100 calls (1 per agent)

Historical Collectors (4 collectors):
- GetMetricDataV2: ceiling(50/100) × 4 = 4 calls

Agent Status:
- ListAgentStatuses: ceiling(10/100) = 1 call

Total per 5 minutes: 107 calls
```

### Hourly API Call Volume

| Collector Type | Calls per Hour | Notes |
|----------------|----------------|-------|
| Realtime Queue | 60 | 1 call/minute |
| Agent Realtime Status | 120 | 2 calls/minute |
| Historical (4 collectors) | 48 | 4 calls every 5 minutes |
| Agent Status | 12 | 1 call every 5 minutes |
| Queue Discovery | 12 | 1 call every 5 minutes |
| Agent Discovery | 12 + 1200 | ListUsers + DescribeUser |
| **Total per Hour** | **1,464** | |

### Daily API Call Volume

| Time Period | API Calls | Cost Estimate* |
|-------------|-----------|----------------|
| **Per Hour** | 1,464 | ~$0.07 |
| **Per Day** | 35,136 | ~$1.76 |
| **Per Month** | ~1,054,080 | ~$52.70 |

*Based on estimated AWS Connect API pricing of ~$0.005 per 1000 API calls

### Scaling with Queue/Agent Count

#### 100 Queues, 200 Agents:
- Realtime calls: 120/hour (ceiling(100/100) × 60)
- Historical calls: 96/hour (ceiling(100/100) × 4 × 12)
- Agent discovery: 2,412/hour (12 ListUsers + 2,400 DescribeUser)
- **Total per hour**: 2,628 calls
- **Monthly**: ~1,892,160 calls (~$94.61/month)

#### 500 Queues, 1000 Agents:
- Realtime calls: 300/hour (ceiling(500/100) × 60) 
- Historical calls: 240/hour (ceiling(500/100) × 4 × 12)
- Agent discovery: 12,012/hour (12 ListUsers + 12,000 DescribeUser)
- **Total per hour**: 12,564 calls
- **Monthly**: ~9,046,080 calls (~$452.30/month)

---

## Cost Optimization Recommendations

### 1. Immediate Optimizations

#### Increase Collection Intervals
```yaml
collectors:
  realtime_queue:
    interval: 120        # Reduce from 60s to 120s (-50% calls)
  
  agent_realtime_status:
    interval: 120        # Reduce from 60s to 120s (-50% calls)
  
  historical_queue:
    interval: 600        # Reduce from 300s to 600s (-50% calls)
  
  historical_calls:
    interval: 600        # Reduce from 300s to 600s (-50% calls)
  
  historical_aht:
    interval: 1800       # Reduce from 300s to 1800s (-83% calls)
  
  historical_acw:
    interval: 1800       # Reduce from 300s to 1800s (-83% calls)
```
**Impact**: Reduces API calls by ~65% while maintaining reasonable monitoring frequency

#### Extend Cache TTLs
```yaml
collectors:
  realtime_queue:
    queue_refresh_interval: 15m    # Reduce from 5m to 15m
    metrics_cache_ttl: 30s        # Increase from 15s to 30s
```
**Impact**: Reduces queue discovery and agent discovery calls by ~67%

#### Use Queue Filtering
```yaml
collectors:
  realtime_queue:
    queues_include_regex: "^(Sales|Support|Emergency).*"
```
**Impact**: Dramatically reduces API calls by focusing on critical queues only

### 2. Disable Non-Critical Collectors

For cost-sensitive environments, disable historical collectors that may not be essential:

```yaml
collectors:
  historical_acw:
    enabled: false       # ACW metrics may not be critical
  
  historical_aht:
    enabled: false       # If handle time isn't monitored actively
```

### 3. Implement Smart Caching

**Agent Discovery Optimization**: 
- Current: 100 agents × 12 times/hour = 1,200 DescribeUser calls/hour
- Recommended: Cache agent details for 1 hour, refresh only changed agents
- **Potential savings**: 90% reduction in DescribeUser calls

### 4. Off-Peak Scheduling

For non-critical historical metrics, schedule collection during off-peak hours:

```yaml
# Example: Historical metrics only during business hours
collectors:
  historical_queue:
    enabled: true
    interval: 300
    schedule: "0 9-17 * * 1-5"  # Business hours only
```

---

## Tuning Parameters

### Cost vs. Freshness Tradeoffs

| Requirement | Realtime Interval | Historical Interval | Queue Refresh | Monthly Cost Estimate* |
|-------------|------------------|-------------------|---------------|----------------------|
| **Real-time Operations** | 30s | 300s | 5m | $70 |
| **Standard Monitoring** | 60s | 600s | 10m | $35 |
| **Cost Optimized** | 120s | 1800s | 15m | $18 |
| **Budget Minimal** | 300s | 3600s | 30m | $10 |

*Based on 50 queues, 100 agents

### Environment-Specific Recommendations

#### High-Volume Contact Centers (500+ Queues)
- Enable only critical metrics
- Use queue filtering extensively  
- Consider custom metrics collection schedules
- Implement agent discovery optimization

#### Small Contact Centers (<20 Queues)
- Can use default intervals
- Enable all collectors for comprehensive monitoring
- Total cost typically <$10/month

#### Development/Testing
```yaml
collectors:
  realtime_queue:
    interval: 300        # 5 minutes for testing
  historical_*:
    enabled: false       # Disable historical in dev
```

---

## Retry Logic Impact on Costs

The realtime queue collector implements retry logic that can increase API call volume:

```go
retry.Attempts(3)                    // Up to 3 attempts
retry.Delay(1 * time.Second)         // 1s initial delay  
retry.MaxDelay(5 * time.Second)      // 5s maximum delay
```

**Worst Case Scenario**: If all API calls fail twice before succeeding, total calls could increase by 200%.

**Monitoring Recommendations**:
- Monitor retry rates in logs
- Alert on high retry rates (>10%) indicating API issues
- Consider reducing retry attempts in cost-sensitive environments

---

## Summary

The AWS Connect Exporter provides comprehensive monitoring with configurable cost controls. Key cost factors:

1. **Queue Count**: Linear scaling (1 additional API call per 100 queues per collector)
2. **Agent Count**: Significant impact due to DescribeUser calls (1 call per agent every 5 minutes)
3. **Collection Frequency**: Direct multiplier on API call volume
4. **Enabled Collectors**: Each historical collector adds 20% to API calls

**Recommended Starting Configuration**: Use standard monitoring intervals with queue filtering to balance cost and operational visibility.

For large-scale deployments, implement agent discovery caching optimizations and consider dedicated cost optimization based on specific monitoring requirements.
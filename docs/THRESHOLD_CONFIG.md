# AWS Connect Metric Thresholds Configuration

## Overview

Some AWS Connect metrics (like `SERVICE_LEVEL`) require threshold parameters to function correctly with the AWS API. This document provides step-by-step guidance for technicians setting up customer-specific metric requirements.

**When to use this guide:**
- Customer requires service level monitoring with specific SLA targets
- Need to measure performance against custom thresholds (e.g., "answer within 30 seconds")
- Setting up alerts or dashboards based on threshold breaches
- AWS Connect metrics return no data without proper threshold configuration

## Prerequisites

Before configuring thresholds, ensure you have:
- ✅ Working AWS Connect exporter deployment
- ✅ Access to the configuration file (usually `config.aws-connect-validated.yml`)
- ✅ Customer's specific SLA requirements (e.g., "80% of calls answered within 20 seconds")
- ✅ Understanding of the customer's queue structure and priorities

## Quick Start: Customer SLA Example

**Scenario:** Customer needs "80% of calls answered within 20 seconds" monitoring.

### Step 1: Identify the Requirements
- **Metric needed:** SERVICE_LEVEL
- **Threshold:** 20 seconds  
- **Target:** 80% (this will be visible in the metric value, not the threshold)
- **Comparison:** Less Than (calls answered in under 20 seconds)

### Step 2: Configure the Threshold
Add this to your configuration file under the appropriate collector:

```yaml
collectors:
  generic_historical:
    enabled: true
    interval: 300  # 5 minutes
    collectors:
      - name: "customer_sla_monitoring"
        metrics:
          - aws_name: "SERVICE_LEVEL"
            prometheus_name: "connect_service_level_20s_percent"
            labels:
              sla_target: "20s"
              customer: "customer_name"  # Replace with actual customer name
            threshold_ref: "sla_20_seconds"
            
        thresholds:
          - name: "sla_20_seconds"
            comparison: "LT"  # Less Than 20 seconds
            value: 20.0       # 20 seconds
            
        groupings: ["QUEUE"]  # Group by queue to see per-queue SLA
        filters: []
```

### Step 3: Restart and Validate
1. Restart the exporter: `sudo systemctl restart aws-connect-exporter`
2. Check metrics appear: `curl http://localhost:9090/metrics | grep connect_service_level_20s_percent`
3. Verify values are reasonable (should be 0-100 representing percentage)

## Configuration Structure

Thresholds are defined separately from metrics and referenced by name, promoting reusability and clean separation of concerns.

### Detailed Configuration Example

```yaml
collectors:
  generic_historical:
    enabled: true
    interval: 300
    collectors:
      - name: "historical_queue"
        metrics:
          # Standard metric without threshold
          - aws_name: "CONTACTS_HANDLED"
            prometheus_name: "connect_historical_contacts_handled_total"
            labels: {}
          
          # Metric with threshold reference
          - aws_name: "SERVICE_LEVEL"
            prometheus_name: "connect_historical_service_level"
            labels: {}
            threshold_ref: "service_level_20s"  # References threshold below
        
        # Threshold definitions
        thresholds:
          - name: "service_level_20s"
            comparison: "LT"  # Less Than
            value: 20.0       # 20 seconds
          - name: "service_level_30s"
            comparison: "LT"
            value: 30.0       # 30 seconds
```

## Threshold Configuration

### MetricThreshold Structure

```yaml
thresholds:
  - name: "unique_threshold_name"    # Required: Unique identifier
    comparison: "LT"                 # Required: "LT", "LTE", or "GT"
    value: 20.0                      # Required: Threshold value (1-604800 for time-based)
```

### Comparison Operators

The comparison operator determines what you're measuring **relative to your threshold value**:

- `"LT"` - **Less Than** (Most Common)
  - **Use for:** "Answered within X seconds", "Handled under X minutes"
  - **Example:** `LT 20.0` = contacts answered in **less than** 20 seconds
  - **SERVICE_LEVEL with LT:** Percentage of calls answered **faster** than threshold
  - **Real-world:** "80% of calls answered within 20 seconds" = `LT 20.0`

- `"LTE"` - **Less Than or Equal To**
  - **Use for:** "At or under X seconds", inclusive thresholds
  - **Example:** `LTE 30.0` = contacts handled in **30 seconds or less**
  - **Difference from LT:** Includes exact threshold matches

- `"GT"` - **Greater Than** (Rarely Used)
  - **Use for:** "Longer than X seconds", measuring delays/problems
  - **Example:** `GT 60.0` = contacts that waited **more than** 60 seconds
  - **SERVICE_LEVEL with GT:** Percentage of calls that took **longer** than threshold
  - **Warning:** Usually not what customers want for SLA monitoring

### ⚠️ Common Mistake
**Customer says:** "Monitor 80% service level at 20 seconds"
- **❌ WRONG:** `GT 20.0` (measures calls taking longer than 20s)
- **✅ CORRECT:** `LT 20.0` (measures calls answered within 20s)

### Value Ranges

- **Time-based thresholds**: 1-604800 seconds (1 second to 1 week)
- **Percentage-based thresholds**: 0.0-100.0
- **Count-based thresholds**: Any positive number

## Metric Reference

To use a threshold with a metric, add the `threshold_ref` field:

```yaml
- aws_name: "SERVICE_LEVEL"
  prometheus_name: "connect_historical_service_level"
  labels: {}
  threshold_ref: "service_level_20s"  # Must match a threshold name
```

## Supported Metrics

Metrics that commonly require thresholds:

- `SERVICE_LEVEL` - Requires time-based threshold (seconds)
- `AGENT_ADHERENCE_TIME` - May require time-based threshold
- Custom service level metrics

## Common Customer Scenarios

### Scenario 1: Multiple SLA Tiers
**Customer requirement:** "Monitor both 20-second and 30-second service levels for different queue priorities"

```yaml
thresholds:
  - name: "premium_sla_20s"
    comparison: "LT"
    value: 20.0
  - name: "standard_sla_30s"  
    comparison: "LT"
    value: 30.0

metrics:
  - aws_name: "SERVICE_LEVEL"
    prometheus_name: "connect_service_level_premium"
    labels:
      tier: "premium"
      target_seconds: "20"
    threshold_ref: "premium_sla_20s"
  - aws_name: "SERVICE_LEVEL"
    prometheus_name: "connect_service_level_standard"
    labels:
      tier: "standard"
      target_seconds: "30"
    threshold_ref: "standard_sla_30s"
```

### Scenario 2: Compliance Monitoring
**Customer requirement:** "Meet regulatory requirement of 90% calls answered within 60 seconds"

```yaml
thresholds:
  - name: "compliance_60s"
    comparison: "LT"
    value: 60.0

metrics:
  - aws_name: "SERVICE_LEVEL"
    prometheus_name: "connect_compliance_service_level"
    labels:
      compliance_type: "regulatory"
      requirement: "90_percent_60s"
    threshold_ref: "compliance_60s"
```

### Scenario 3: Queue-Specific SLAs
**Customer requirement:** "Different SLAs per queue type (Sales: 15s, Support: 30s, Billing: 45s)"

```yaml
thresholds:
  - name: "sales_sla_15s"
    comparison: "LT"
    value: 15.0
  - name: "support_sla_30s"
    comparison: "LT"
    value: 30.0
  - name: "billing_sla_45s"
    comparison: "LT"
    value: 45.0

# Use filters to apply different thresholds to different queues
collectors:
  - name: "sales_queues_sla"
    metrics:
      - aws_name: "SERVICE_LEVEL"
        prometheus_name: "connect_queue_service_level"
        labels:
          department: "sales"
        threshold_ref: "sales_sla_15s"
    groupings: ["QUEUE"]
    filters:
      - key: "QUEUES"
        values: ["arn:aws:connect:region:account:instance/instance-id/queue/sales-queue-id"]
```

## Best Practices for Technicians

### 1. Customer Requirements Gathering
- **Always ask for specific SLA targets** (not just "good service level")
- **Identify queue priorities** - different queues may need different thresholds
- **Understand business hours** - some customers only care about SLA during business hours
- **Clarify measurement periods** - daily, weekly, monthly reporting needs

### 2. Threshold Naming Conventions
- **Use descriptive names**: `service_level_20s`, `adherence_5min`, `compliance_60s`
- **Include customer context**: `acme_corp_premium_sla`, `finance_dept_15s`
- **Avoid generic names**: `threshold1`, `sla`, `default`

### 3. Configuration Management
- **Document threshold rationale**: Add comments explaining why specific values were chosen
- **Version control**: Keep configuration changes tracked
- **Test before deployment**: Always validate against customer's AWS Connect instance
- **Monitor collection success**: Check exporter logs for threshold-related errors

### 4. Validation Checklist
Before deploying threshold configurations:
- [ ] Threshold values match customer requirements exactly
- [ ] Comparison operators are correct (LT for "answered within X seconds")
- [ ] Prometheus metric names follow naming conventions
- [ ] Labels provide enough context for filtering/alerting
- [ ] Configuration syntax is valid (YAML formatting)
- [ ] Tested against customer's AWS Connect instance

## Example: Multiple Service Levels

```yaml
thresholds:
  - name: "service_level_20s"
    comparison: "LT"
    value: 20.0
  - name: "service_level_30s"  
    comparison: "LT"
    value: 30.0
  - name: "service_level_60s"
    comparison: "LT" 
    value: 60.0

metrics:
  - aws_name: "SERVICE_LEVEL"
    prometheus_name: "connect_historical_service_level_20s"
    labels:
      threshold: "20s"
    threshold_ref: "service_level_20s"
  - aws_name: "SERVICE_LEVEL"
    prometheus_name: "connect_historical_service_level_30s"
    labels:
      threshold: "30s"  
    threshold_ref: "service_level_30s"
```

## Troubleshooting Guide for Technicians

### Common Issues and Solutions

#### 1. "Threshold not found" Error
**Symptom:** Exporter logs show `threshold 'xxx' not found`
**Cause:** Mismatch between `threshold_ref` and threshold `name`

**Solution:**
```yaml
# ❌ WRONG - case mismatch
threshold_ref: "service_level_20s"
thresholds:
  - name: "Service_Level_20s"  # Different case

# ✅ CORRECT - exact match
threshold_ref: "service_level_20s"
thresholds:
  - name: "service_level_20s"  # Exact match
```

#### 2. Service Level Metrics Return Zero/No Data
**Symptom:** `connect_service_level_*` metrics always show 0 or no data points
**Causes & Solutions:**

**A. Incorrect Comparison Operator:**
```yaml
# ❌ WRONG - "GT" means "greater than X seconds" (longer wait times)
comparison: "GT"  
value: 20.0

# ✅ CORRECT - "LT" means "less than X seconds" (answered quickly)  
comparison: "LT"
value: 20.0
```

**B. No Historical Data Available:**
- SERVICE_LEVEL is a historical metric (not real-time)
- Requires historical data in AWS Connect (may take 24-48 hours for new instances)
- Check AWS Connect Reports to verify historical data exists

**C. Queue Has No Activity:**
- Empty queues won't generate service level data
- Verify the queue processes contacts during the measurement period

#### 3. Values Don't Match Customer Expectations
**Symptom:** Metric shows 65% but customer reports 80% service level

**Root Cause Analysis:**
1. **Check measurement period**: AWS Connect vs customer's reporting timeframe
2. **Verify threshold value**: Customer says "20 seconds" - confirm they mean 20.0, not 0:20 (20 milliseconds)
3. **Confirm queue scope**: Are you measuring the right queues?

**Validation Steps:**
```bash
# Check what queues are being measured
curl -s http://localhost:9090/metrics | grep connect_service_level | head -5

# Compare with AWS Connect Reports in the console
# Navigate to: Analytics and optimization > Historical metrics > Service level
```

#### 4. Configuration Syntax Errors
**Symptom:** Exporter fails to start or ignores threshold configuration

**Common YAML Mistakes:**
```yaml
# ❌ WRONG - Missing quotes around comparison
comparison: LT

# ✅ CORRECT - Quoted string
comparison: "LT"

# ❌ WRONG - Inconsistent indentation  
thresholds:
  - name: "test"
  comparison: "LT"  # Should be indented
  value: 20.0

# ✅ CORRECT - Consistent indentation
thresholds:
  - name: "test"
    comparison: "LT"  
    value: 20.0
```

### Validation Process for New Deployments

#### Step 1: Pre-Deployment Testing
```bash
# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('config.aws-connect-validated.yml'))"

# Check configuration is loading correctly
./bin/aws-connect-exporter --config.file=config.aws-connect-validated.yml --log.level=debug --dry-run
```

#### Step 2: Post-Deployment Validation  
```bash
# 1. Check exporter is running
curl http://localhost:9090/health

# 2. Wait 5-10 minutes for first collection, then check metrics appear
curl -s http://localhost:9090/metrics | grep service_level | head -3

# 3. Verify metric values are reasonable (0-100 for percentages)
curl -s http://localhost:9090/metrics | grep service_level | grep -v "#"
```

#### Step 3: Customer Data Verification
1. **Compare with AWS Console**: Check Historical Reports in AWS Connect console
2. **Verify timeframes match**: Ensure measurement periods align  
3. **Check queue selection**: Confirm the right queues are included
4. **Document baseline**: Record initial values for customer validation

### Emergency Rollback Procedure

If threshold configuration causes issues:

```bash
# 1. Stop the exporter
sudo systemctl stop aws-connect-exporter

# 2. Restore previous configuration
cp config.aws-connect-validated.yml.backup config.aws-connect-validated.yml

# 3. Start with basic configuration (no thresholds)
# Remove or comment out the problematic threshold sections

# 4. Restart exporter  
sudo systemctl start aws-connect-exporter

# 5. Verify basic metrics are working
curl http://localhost:9090/metrics | grep connect_queue_
```

### Getting Help

When escalating threshold configuration issues:

1. **Collect logs**: `journalctl -u aws-connect-exporter -n 100`
2. **Configuration snapshot**: Current config file (mask sensitive data)
3. **Customer requirements**: Exact SLA requirements and expectations
4. **AWS Connect context**: Instance size, queue count, historical data availability
5. **Metric samples**: Output from `/metrics` endpoint showing the issue
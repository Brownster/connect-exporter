# AWS Connect Metric Thresholds Configuration

## Overview

Some AWS Connect metrics (like `SERVICE_LEVEL`) require threshold parameters to function correctly with the AWS API. This document explains how to configure thresholds using the clean threshold architecture.

## Configuration Structure

Thresholds are defined separately from metrics and referenced by name, promoting reusability and clean separation of concerns.

### Basic Example

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

- `"LT"` - Less Than
- `"LTE"` - Less Than or Equal To  
- `"GT"` - Greater Than

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

## Best Practices

1. **Reusable Thresholds**: Define common thresholds once and reference them multiple times
2. **Descriptive Names**: Use clear threshold names like `service_level_20s` or `adherence_5min`
3. **Documentation**: Comment your threshold choices in the config
4. **Testing**: Test threshold metrics against your AWS Connect instance to ensure they work

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

## Troubleshooting

### Common Issues

1. **Threshold not found**: Ensure `threshold_ref` matches exactly with a defined threshold `name`
2. **Invalid comparison**: Use only `"LT"`, `"LTE"`, or `"GT"`
3. **Invalid value range**: Check AWS documentation for valid threshold ranges
4. **API errors**: Some metrics may not support thresholds on all AWS Connect instances

### Debug Steps

1. Check exporter logs for threshold-related errors
2. Verify threshold name spelling and case sensitivity
3. Test with a simple threshold first (e.g., 20 seconds for SERVICE_LEVEL)
4. Validate your AWS Connect instance supports the specific metric with thresholds
# Adding New Metrics

You can teach the exporter and the mock API about new AWS Connect metrics. This lets you monitor exactly what you care about.

## Exporter configuration

1. Open your configuration file (for example `config.simple.yml`).
2. Under `collectors.generic_realtime.collectors` add a new metric entry.

Example:

```yaml
collectors:
  generic_realtime:
    enabled: true
    interval: 30
    collectors:
      - name: "queue_realtime_metrics"
        metrics:
          - aws_name: "AGENTS_AVAILABLE"
            prometheus_name: "connect_queue_agents"
            labels:
              state: "Available"
          - aws_name: "CONTACTS_IN_QUEUE"    # new metric
            prometheus_name: "connect_queue_contacts_in_queue"
            labels: {}
        groupings: ["QUEUE", "CHANNEL"]
        filters: []
```

The `aws_name` must match the name used by AWS Connect. The `prometheus_name` is what appears in Prometheus.
Restart the exporter after editing the file.

## Mock API metrics

The mock API reads `docs/tools/aws-mock-api-main/metrics-config.json`. To add a new metric:

1. Open `metrics-config.json`.
2. Add an entry to the `"metrics"` list.

Example:

```json
{ "name": "CONTACTS_IN_QUEUE", "unit": "COUNT", "type": "random", "min": 1, "max": 8 }
```

Each metric describes how to generate the value (`random`, `ratio`, etc.). Restart the mock API to load the changes.

Now your new metric will show up when you scrape the exporter.

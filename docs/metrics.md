# Metrics

The exporter exposes the following Prometheus metrics:

| Metric | Labels | Description |
|--------|--------|-------------|
| `rss_exporter_service_status` | `service`, `customer` (optional), `state` | Current service state: `ok`, `service_issue`, or `outage`. |
| `rss_exporter_service_issue_info` | `service`, `customer` (optional), `service_name` (optional), `region` (optional), `title`, `link`, `guid` | Information about the active incident, value is always `1` when present. |

The `service_name` and `region` labels are only populated for providers that
include this information in their feeds, such as **aws** and **azure**.

Example scrape output:

```text
# HELP rss_exporter_service_issue_info Details for the currently active service issue.
# TYPE rss_exporter_service_issue_info gauge
rss_exporter_service_issue_info{guid="storage-eastus_issue",link="https://status.azure.com/en-us/status",region="eastus",service="azure",service_name="storage",title="Service issue: Storage - East US"} 1
# HELP rss_exporter_service_status Current service status parsed from configured feeds.
# TYPE rss_exporter_service_status gauge
rss_exporter_service_status{service="azure",state="ok"} 0
rss_exporter_service_status{service="azure",state="outage"} 0
rss_exporter_service_status{service="azure",state="service_issue"} 1
```


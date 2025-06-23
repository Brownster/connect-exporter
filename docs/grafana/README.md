# Grafana Dashboards

This directory contains example dashboards for visualizing metrics exported by
**RSS Exporter**.

- `traffic_lights.json` – shows the status of selected services as simple
  green/yellow/red indicators.
- `provider_traffic_lights.json` – lets you choose a provider and displays all
  associated services.

Import these JSON files into Grafana and select your Prometheus datasource.

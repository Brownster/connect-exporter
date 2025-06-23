# Configuration

The exporter is configured via a YAML file passed with the `-config.file` flag. The default path is `config.yml`.

## Top-level options

| Field           | Description                         | Default |
|-----------------|-------------------------------------|---------|
| `listen_address`| Address to bind the HTTP server     | `0.0.0.0` |
| `listen_port`   | Port for the HTTP server            | `9091` |
| `log_level`     | Log verbosity (`trace`, `debug`, `info`, `warn`) | `info` |
| `services`      | List of RSS/Atom feeds to monitor   | - |

### Service fields

Each entry under `services` defines a single feed.

| Field      | Description                                                      |
|------------|------------------------------------------------------------------|
| `name`     | Unique identifier for the service.                               |
| `provider` | Optional scraper to use (`aws`, `gcp`, `azure`, etc.). When omitted the service name is inspected. |
| `customer` | Optional customer or tenant name. Appears as a metric label.     |
| `url`      | RSS or Atom feed URL.                                            |
| `interval` | Polling interval in seconds (defaults to `300` when not set).    |

Example configuration:

```yaml
listen_address: 127.0.0.1
listen_port: 9091
log_level: debug
services:
  - name: openai
    provider: openai
    url: https://status.openai.com/history.atom
    interval: 300
```


# AWS Connect Exporter

A production-ready Prometheus exporter for AWS Connect that provides comprehensive contact center metrics with advanced configuration capabilities.

## 🚀 Quick Start

**New to the project?** Run the setup script:

```bash
./scripts/setup.sh
```

**Already set up?** Jump to usage:

```bash
# Development (no AWS costs)
ENV=development ./bin/aws-connect-exporter --config.file=configs/config.development.yml --no-advertise

# Production  
./bin/aws-connect-exporter --config.file=config.aws-connect-validated.yml --no-advertise
```

## 📁 Project Structure

```
aws-connect-exporter/
├── bin/                          # 📦 Compiled binaries
│   ├── aws-connect-exporter     # Main production binary
│   └── README.md               # Binary documentation
├── configs/                     # ⚙️  Configuration files
│   ├── config.development.yml  # Development config (mock API)
│   ├── config.aws-connect.yml  # Production template
│   └── README.md              # Configuration documentation
├── docs/                        # 📚 Documentation
│   ├── aws-connect-setup.md    # Complete setup guide
│   ├── quick-start.md          # Quick start guide
│   └── tools/aws-mock-api-main/ # Mock API for development
├── scripts/                     # 🔧 Build and utility scripts
│   ├── build.sh               # Build automation
│   ├── setup.sh               # Quick setup for new users
│   └── import-dashboards.sh   # Grafana dashboard import
├── dashboards/                  # 📊 Grafana dashboards
├── cmd/                        # Go application entry point
├── collectors/                 # Metric collectors
├── connectors/                 # AWS Connect API connectors
└── config.aws-connect-validated.yml  # 🎯 ACTIVE TEST CONFIG
```

## ✨ Key Features

### 🔄 **Configuration-Driven Architecture**
- **No Code Changes**: Add any AWS Connect metric via YAML configuration
- **Generic Collectors**: Flexible realtime and historical metric collection
- **Batch Processing**: Handles 100+ queues with proper AWS API compliance
- **Production Validated**: 16+ comprehensive metrics tested against real AWS Connect

### 📊 **Comprehensive Metric Coverage**

**Agent Metrics** (Enhanced Labeling):
- Realtime agent status with `first_name`, `last_name`, `username`
- Routing profile assignments and queue associations
- Agent state distribution and performance tracking

**Queue Metrics** (Realtime):
- Contact volumes and wait times
- Agent availability by state and queue
- Queue capacity and utilization metrics

**Historical Metrics** (Daily Performance):
- Contact handling volumes (handled, abandoned, queued)
- Service level performance with configurable thresholds
- Average handle times, queue answer times, abandonment rates

### 🏷️ **Rich Labeling System**
- **Agent Labels**: `agent_id`, `username`, `first_name`, `last_name`, `routing_profile_id`, `status_name`
- **Queue Labels**: `queue_id`, `queue_name`, `queue_type`, `channel`, `region`
- **Historical Labels**: `direction` (inbound/outbound), service level thresholds

## 🛠️ Installation & Setup

### Prerequisites
- **Go 1.21+** for building from source
- **AWS Account** with AWS Connect instance
- **IAM Permissions** for AWS Connect API access
- **Node.js** (optional, for development mock API)

### Option 1: Quick Setup (Recommended)
```bash
git clone https://github.com/Brownster/connect-exporter.git
cd connect-exporter
./scripts/setup.sh
```

### Option 2: Manual Setup
```bash
# 1. Install dependencies
go mod tidy

# 2. Build binary
./scripts/build.sh
# OR manually: go build -o bin/aws-connect-exporter ./cmd/connect-exporter_exporter

# 3. Configure AWS credentials
export AWS_PROFILE=your-profile
export AWS_REGION=us-east-1

# 4. Update configuration with your AWS Connect instance ARN
# Edit config.aws-connect-validated.yml or create your own

# 5. Run exporter
./bin/aws-connect-exporter \
  --web.listen-port=9090 \
  --labels.fqdn=your-hostname \
  --labels.location=your-location \
  --labels.country=US \
  --labels.ip-address=127.0.0.1 \
  --config.file=config.aws-connect-validated.yml \
  --no-advertise
```

## 💻 Development Mode

For cost-free development and testing:

```bash
# Terminal 1: Start mock AWS Connect API
cd docs/tools/aws-mock-api-main
node server.js

# Terminal 2: Run exporter with mock API
ENV=development ./bin/aws-connect-exporter \
  --web.listen-port=9090 \
  --labels.fqdn=localhost \
  --labels.location=test \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1 \
  --config.file=configs/config.development.yml \
  --no-advertise \
  --log.level=debug
```

## 📈 Monitoring

### Metrics Endpoint
```bash
# Check metrics
curl http://localhost:9090/metrics

# Health check  
curl http://localhost:9090/health
```

### Sample Prometheus Queries
```promql
# Current contacts in queue
aws_connect_exporter_generic_realtime_generic_metric{metric_name="connect_queue_contacts_in_queue"}

# Agent status distribution
sum by (status_name) (aws_connect_exporter_agent_realtime_status_connect_agent_status_state)

# Daily handled contacts by queue
sum by (queue_name) (aws_connect_exporter_generic_historical_generic_metric{metric_name="connect_historical_contacts_handled_total"})
```

### Grafana Dashboards
Pre-built dashboards available in `dashboards/`:
- **Real-time Operations**: Live queue and agent monitoring
- **Daily Performance**: Historical metrics and trends

Import with: `./scripts/import-dashboards.sh`

## ⚙️ Configuration

### Minimal Configuration
```yaml
aws:
  region: "us-east-1"
  instance_arn: "arn:aws:connect:us-east-1:123456789012:instance/your-instance-id"

collectors:
  agent_realtime_status:
    enabled: true
    interval: 60
  generic_realtime:
    enabled: true
    interval: 30
    collectors:
      - name: "queue_metrics"
        metrics:
          - aws_name: "CONTACTS_IN_QUEUE"
            prometheus_name: "connect_queue_contacts_in_queue"
        groupings: ["QUEUE", "CHANNEL"]
```

### Configuration Files
- **`config.aws-connect-validated.yml`** - Production-validated configuration ⭐
- **`configs/config.development.yml`** - Development with mock API
- **`configs/config.aws-connect.yml`** - Full production template

See `configs/README.md` for detailed configuration documentation.

## 🔧 IAM Permissions

Required AWS permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "connect:ListQueues",
        "connect:GetCurrentMetricData", 
        "connect:GetMetricDataV2",
        "connect:ListAgentStatuses",
        "connect:GetCurrentUserData",
        "connect:ListUsers",
        "connect:DescribeUser",
        "connect:ListRoutingProfiles"
      ],
      "Resource": "arn:aws:connect:*:*:instance/your-instance-id"
    }
  ]
}
```

## 📊 Architecture

![Architecture Diagram](docs/architecture-diagram.png)

The exporter uses a **configuration-driven architecture** with:

- **Generic Collectors**: Flexible metric definitions via YAML
- **AWS Connect Connectors**: Efficient API client with caching and batching
- **Agent Manager**: Enhanced agent metadata with 12-hour caching
- **Queue Manager**: Smart queue discovery with AGENT queue filtering
- **Monitoring-as-a-Service (MAAS) Framework**: Production-ready scaffolding

## 🔍 Troubleshooting

### Common Issues

1. **"unexpected false" Error**
   - Ensure all required CLI flags are provided
   - Use `--no-advertise` instead of `--advertise=false`

2. **Empty Agent Names**
   - Verify `connect:DescribeUser` IAM permission
   - Check agent IdentityInfo is populated in AWS Connect

3. **High Memory Usage**
   - Reduce collection intervals for large instances
   - Disable unnecessary collectors in configuration

4. **API Rate Limiting**
   - Increase collection intervals
   - The exporter includes automatic retry logic

### Debug Logging
```bash
./bin/aws-connect-exporter --log.level=debug --config.file=your-config.yml
```

## 📚 Documentation

- **[Setup Guide](docs/aws-connect-setup.md)** - Complete installation and configuration
- **[Quick Start](docs/quick-start.md)** - Get up and running fast
- **[Configuration](configs/README.md)** - All configuration options explained
- **[Development Guide](docs/development-with-mock-api.md)** - Development with mock API
- **[Architecture](docs/DEVELOPER.md)** - Technical architecture and patterns
- **[Adding Metrics](docs/adding-metrics.md)** - How to add custom metrics

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and test with the mock API
4. Run the build: `./scripts/build.sh`
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **GitHub Issues**: [Report bugs or request features](https://github.com/Brownster/connect-exporter/issues)
- **Documentation**: Check the `docs/` directory
- **Configuration Help**: See `configs/README.md`

---

**⭐ If this project helps you, please give it a star!**
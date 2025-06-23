# Template Exporter

A comprehensive template repository for creating Prometheus exporters using the monitoring-maas framework.

## 🚀 Quick Start

### 1. Create Your New Exporter

```bash
# Clone this template
git clone https://github.com/Brownster/template_exporter.git my-new-exporter
cd my-new-exporter

# Run the setup script
./scripts/setup-new-exporter.sh my-service-name my-github-org

# Validate the setup
./scripts/validate-template.sh
```

### 2. Implement Your Logic

Edit the generated collector files and implement your specific scraping logic:

```bash
# Your main files to customize:
- cmd/my_service_name_exporter/main.go
- collectors/my_service_name_exporter.go  
- collectors/my_service_name_collector.go
- config.template.yml
```

### 3. Test Your Exporter

```bash
go test ./...
go build -o my_exporter ./cmd/my_service_name_exporter
./my_exporter --config.file=config.template.yml
```

## 📚 What's Included

### 🏗️ **Core Template Structure**
- **Multiple collector patterns**: HTTP API, SSH commands, file monitoring, SNMP
- **Production-ready main.go**: Following create_template_repo.md patterns
- **Comprehensive testing**: Mock connectors and metric validation
- **Template configuration**: Multi-service configuration examples

### 🔧 **Automation Scripts**
- `scripts/setup-new-exporter.sh` - Automated new exporter creation
- `scripts/validate-template.sh` - Template validation and verification  
- `scripts/migrate-to-production.sh` - Production migration automation

### 🧪 **Testing Framework**
- Mock connectors with response simulation
- Comprehensive test patterns and helpers
- Realistic test data for different scenarios
- Performance and error testing utilities

### 🚀 **CI/CD Templates**
- Complete GitHub Actions workflows
- Automated testing, building, and releasing
- Multi-platform Docker builds
- Dependency management with Dependabot

### 📦 **Configuration Templates**
- Docker and Docker Compose setups
- Prometheus and Grafana configurations
- Alerting rules examples
- Linting and code quality configs

## 🎯 **Collector Patterns**

### HTTP API Monitoring
```go
func NewAPICollector(app *kingpin.Application, service APIService) *maas.ScheduledScraper {
    return maas.NewScheduledScraper("api", NewAPIScraper(service),
        maas.WithSchedule(maas.NewSchedule(...)),
        maas.WithDescription(app, "available", "API availability", []string{"endpoint"}),
    )
}
```

### System Command Monitoring  
```go
func NewSystemCollector(app *kingpin.Application) *maas.ScheduledScraper {
    return maas.NewScheduledScraper("system", NewSystemScraper(app),
        maas.WithSchedule(maas.NewSchedule(...)),
        maas.WithDescription(app, "uptime_seconds", "System uptime", nil),
    )
}
```

### File Monitoring
```go
func NewFileCollector(app *kingpin.Application, filePath string) *maas.ScheduledScraper {
    return maas.NewScheduledScraper("file", NewFileScraper(filePath),
        maas.WithSchedule(maas.NewSchedule(...)),
        maas.WithDescription(app, "size_bytes", "File size", []string{"path"}),
    )
}
```

## 🔄 **Development → Production Workflow**

### Development Phase
- Use local `monitoring-maas` folder for fast iteration
- Test with mock connectors and sample data
- Follow template patterns and examples

### Production Migration
```bash
# When ready for production deployment
./scripts/migrate-to-production.sh
```

This removes the local monitoring-maas and updates imports to use:
- `github.com/sabio-engineering-product/monitoring-maas`
- `github.com/sabio-engineering-product/monitoring-connectors`

## 📖 **Documentation**

- [Template Developer Guide](docs/TEMPLATE_DEVELOPER_GUIDE.md) - Comprehensive development guide
- [Create Template Repo Guide](create_template_repo.md) - Original framework patterns
- [Template Examples](collectors/template_*.go) - Working collector examples
- [Testing Patterns](collectors/template_test_helpers.go) - Testing framework guide

## 🛠️ **Example Usage**

### HTTP API Exporter
```bash
./scripts/setup-new-exporter.sh api-monitor my-org
# Creates an exporter for monitoring REST APIs
```

### System Monitoring Exporter
```bash
./scripts/setup-new-exporter.sh system-monitor my-org  
# Creates an exporter for SSH-based system monitoring
```

### Database Exporter
```bash
./scripts/setup-new-exporter.sh db-monitor my-org
# Creates an exporter for database monitoring
```

## 🏢 **Enterprise Features**

- **Standardized Structure**: Consistent across all exporters
- **Production Ready**: Error handling, timeouts, graceful shutdown
- **Monitoring**: Built-in metrics for exporter health
- **Security**: Best practices for secrets and authentication
- **Scalability**: Efficient scheduling and resource usage

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch
3. Add tests for your changes
4. Ensure all tests pass
5. Submit a pull request

## 📋 **Requirements**

- Go 1.21 or newer
- Docker (optional, for containerization)
- Access to monitoring-maas framework (for production use)

## 🎉 **Benefits**

- **Fast Development**: Working exporter in minutes
- **Best Practices**: Proven patterns from production exporters
- **Comprehensive Testing**: Full test coverage with mock framework
- **Production Ready**: Immediate deployment capability
- **Team Consistency**: Standardized structure across projects

---

**Transform your monitoring requirements into production-ready Prometheus exporters with proven patterns and comprehensive tooling.**
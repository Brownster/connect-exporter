# Development with Mock AWS Connect API

This guide covers local development using the included mock AWS Connect API server to reduce costs and speed up development.

## Overview

The mock API server simulates AWS Connect API endpoints locally, providing:
- Realistic queue and agent status data
- Error simulation for testing resilience
- Configurable error rates
- AWS-like pagination and responses
- Zero AWS costs during development

## Quick Start

### 1. Start the Mock API Server

```bash
# Navigate to the mock API directory
cd docs/tools/aws-mock-api-main

# Install dependencies (first time only)
npm install express body-parser

# Start the server
node server.js
```

Verify the server is running:
```bash
curl http://localhost:3000/health
```

The server starts on `http://localhost:3000` by default.

### 2. Configure Development Environment

Set environment variables to enable development mode:

```bash
export ENV=development
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000  # optional, defaults to localhost:3000
```

### 3. Run the Exporter

```bash
./aws-connect-exporter --config.file=config.development.yml
```

## Mock API Features

### Supported Endpoints

The mock API provides the following AWS Connect endpoints:

#### Queue Operations
- **ListQueues**: Returns paginated list of 4 predefined queues
- **GetCurrentMetricData**: Provides realistic metrics for queue monitoring
- **GetMetricDataV2**: Simulates historical metric queries grouped by queue and channel

#### Agent Status Operations  
- **ListAgentStatuses**: Returns paginated agent status information

### Mock Data

#### Predefined Queues
1. **Sales Queue** (`queue-sales-01`)
   - Type: STANDARD, Channel: VOICE
   - Realistic sales queue metrics

2. **Support Queue** (`queue-support-01`) 
   - Type: STANDARD, Channel: VOICE
   - Customer support queue metrics

3. **Monitoring Voice Queue** (`queue-support-999`)
   - Type: AGENT, Channel: VOICE
   - Internal monitoring queue

4. **Monitoring Chat Queue** (`queue-chat-999`)
   - Type: AGENT, Channel: CHAT
   - Chat-based monitoring queue

#### Available Metrics
- `CONTACTS_IN_QUEUE` (COUNT)
- `OLDEST_CONTACT_AGE` (SECONDS)
- `AGENTS_ON_CONTACT` (COUNT)
- `AGENTS_NON_PRODUCTIVE` (COUNT)
- `SLOTS_AVAILABLE` (COUNT)
- `SLOTS_ACTIVE` (COUNT)
- `AGENTS_AVAILABLE` (COUNT)
- `AGENTS_STAFFED` (COUNT)
- `AGENTS_AFTER_CONTACT_WORK` (COUNT)
- `CONTACTS_SCHEDULED` (COUNT)

### Error Simulation

The mock API simulates realistic AWS service errors:

#### Error Types
- `ThrottlingException`
- `ServiceUnavailableException`
- `InternalServiceErrorException`
- `ResourceNotFoundException`
- `InvalidParameterException`
- `ValidationException`

#### Configuring Error Rates

```bash
# Set 10% error rate
curl -X POST http://localhost:3000/admin/errorRate \
  -H 'Content-Type: application/json' \
  -d '{"rate": 0.1}'

# Set 50% error rate for testing resilience
curl -X POST http://localhost:3000/admin/errorRate \
  -H 'Content-Type: application/json' \
  -d '{"rate": 0.5}'

# Reset to low error rate
curl -X POST http://localhost:3000/admin/errorRate \
  -H 'Content-Type: application/json' \
  -d '{"rate": 0.05}'
```

## Development Configuration

### config.development.yml

```yaml
# AWS Configuration (for development with mock API)
aws:
  region: "eu-west-2"
  instance_arn: "arn:aws:connect:eu-west-2:123456789012:instance/f604177c-b2ab-4c76-9033-635d195b2772"

# Faster collection intervals for development
collectors:
  realtime_queue:
    enabled: true
    interval: 30  # 30 seconds for faster feedback
  agent_status:
    enabled: true
    interval: 60  # 1 minute for faster feedback

# Debug logging
log_level: debug
```

### Environment Detection

The exporter automatically detects development mode through:

1. **Environment Variables**:
   ```bash
   ENV=development
   NODE_ENV=development
   AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000
   ```

2. **Automatic Endpoint Configuration**: When in development mode, the AWS Connect client automatically points to the mock endpoint.

## Development Workflow

### 1. Test Driven Development

```bash
# Terminal 1: Start mock API
cd docs/tools/aws-mock-api-main && node server.js

# Terminal 2: Run tests with mock data
export ENV=development
go test ./collectors/...

# Terminal 3: Run exporter with live reload
export ENV=development
./aws-connect-exporter --config.file=config.development.yml
```

### 2. Error Resilience Testing

```bash
# Test high error rate scenarios
curl -X POST http://localhost:3000/admin/errorRate -d '{"rate": 0.8}'

# Watch exporter logs for retry behavior
tail -f exporter.log

# Reset to normal operation
curl -X POST http://localhost:3000/admin/errorRate -d '{"rate": 0.1}'
```

### 3. Metric Validation

```bash
# Check metrics endpoint
curl http://localhost:9090/metrics | grep aws_connect

# Validate specific metrics
curl -s http://localhost:9090/metrics | grep "aws_connect_realtime_queue_contacts_in_queue"
curl -s http://localhost:9090/metrics | grep "aws_connect_agent_status_count"
```

## Testing Scenarios

### Scenario 1: Normal Operation

```bash
# Set low error rate (5%)
curl -X POST http://localhost:3000/admin/errorRate -d '{"rate": 0.05}'

# Observe normal metric collection
curl http://localhost:9090/metrics | grep aws_connect_realtime_queue
```

### Scenario 2: High Error Rate

```bash
# Set high error rate (80%)
curl -X POST http://localhost:3000/admin/errorRate -d '{"rate": 0.8}'

# Verify retry logic and partial failure handling
tail -f exporter.log | grep "retry\|error"
```

### Scenario 3: Service Unavailable

```bash
# Stop mock API server to simulate service outage
# Observe exporter behavior and error handling
```

### Scenario 4: Pagination Testing

The mock API returns 2 items per page by default, allowing testing of pagination logic with the 4 predefined queues.

## Debugging

### Mock API Logs

The mock API server provides detailed logging:

```bash
# Start with verbose logging
DEBUG=* node server.js
```

### Exporter Debug Logs

```bash
# Enable debug logging
./aws-connect-exporter --log.level=debug --config.file=config.development.yml
```

### Network Traffic Analysis

```bash
# Monitor HTTP traffic to mock API
tcpdump -A -s 0 'tcp port 3000'

# Or use curl to test endpoints directly
curl -v http://localhost:3000/queues-summary/f604177c-b2ab-4c76-9033-635d195b2772
```

## Performance Testing

### Load Testing

```bash
# Test collector performance with faster intervals
# Edit config.development.yml intervals to 5-10 seconds
# Monitor memory and CPU usage
```

### Memory Profiling

```bash
# Enable Go profiling
go build -race -o aws-connect-exporter ./cmd/connect-exporter_exporter
./aws-connect-exporter --config.file=config.development.yml

# Access profiling endpoints
curl http://localhost:9090/debug/pprof/heap
```

## Migration to Production

When ready for production deployment:

### 1. Update Configuration

```bash
# Copy production template
cp config.aws-connect.yml config.production.yml

# Update with real AWS Instance ARN
vim config.production.yml
```

### 2. Update Environment

```bash
# Remove development environment variables
unset ENV
unset NODE_ENV
unset AWS_CONNECT_MOCK_ENDPOINT

# Set production AWS credentials
export AWS_REGION=us-east-1
export AWS_PROFILE=production
```

### 3. Test Production Connection

```bash
# Test with production credentials
./aws-connect-exporter --config.file=config.production.yml --log.level=debug
```

## Best Practices

1. **Always use mock API for initial development**
2. **Test error scenarios with configurable error rates**
3. **Validate metrics format before production deployment**
4. **Use realistic collection intervals during development**
5. **Monitor resource usage during development**
6. **Test production credentials in staging environment first**

## Troubleshooting

### Common Issues

1. **Mock API Not Starting**: Check Node.js installation and port availability
2. **Exporter Not Using Mock**: Verify environment variables and development mode detection
3. **Missing Metrics**: Check mock API data and endpoint availability
4. **High CPU Usage**: Reduce collection intervals in development config

### Debug Commands

```bash
# Check mock API status
curl http://localhost:3000/health || echo "Mock API not running"

# Verify development mode
./aws-connect-exporter --config.file=config.development.yml --log.level=debug 2>&1 | grep "development mode"

# Test mock endpoints directly
curl http://localhost:3000/queues-summary/f604177c-b2ab-4c76-9033-635d195b2772
```
#!/bin/bash
#
# Health check script for AWS Connect Exporter
#

set -euo pipefail

echo "🔍 AWS Connect Exporter Health Check"
echo

# Check if exporter is running
if ! pgrep -f "aws-connect-exporter" > /dev/null; then
    echo "❌ Exporter process not running"
    exit 1
fi

echo "✅ Exporter process is running"

# Check health endpoint
if curl -s http://localhost:9090/health > /dev/null; then
    HEALTH=$(curl -s http://localhost:9090/health)
    echo "✅ Health endpoint: $HEALTH"
else
    echo "❌ Health endpoint not responding"
    exit 1
fi

# Check metrics endpoint
METRIC_COUNT=$(curl -s http://localhost:9090/metrics | grep -c "aws_connect_exporter" || echo "0")
echo "✅ Metrics available: $METRIC_COUNT metric families"

# Check specific metric types
AGENT_METRICS=$(curl -s http://localhost:9090/metrics | grep -c "agent_realtime_status" || echo "0")
QUEUE_METRICS=$(curl -s http://localhost:9090/metrics | grep -c "generic_realtime" || echo "0") 
HIST_METRICS=$(curl -s http://localhost:9090/metrics | grep -c "generic_historical" || echo "0")

echo "   - Agent metrics: $AGENT_METRICS"
echo "   - Queue metrics: $QUEUE_METRICS" 
echo "   - Historical metrics: $HIST_METRICS"

if [[ $METRIC_COUNT -gt 0 ]]; then
    echo
    echo "🎉 AWS Connect Exporter is healthy and collecting metrics!"
else
    echo
    echo "⚠️  Exporter is running but no metrics found. Check logs and AWS credentials."
fi

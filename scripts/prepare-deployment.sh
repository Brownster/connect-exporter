#!/bin/bash
#
# AWS Connect Exporter - Deployment Package Creator
#
# Creates a minimal deployment package for CentOS lab testing
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$PROJECT_ROOT/deploy-package"

echo -e "${BLUE}📦 AWS Connect Exporter - Deployment Package Creator${NC}"
echo

cd "$PROJECT_ROOT"

# Ensure binary exists
if [[ ! -f "bin/aws-connect-exporter" ]]; then
    echo -e "${YELLOW}⚠️  Binary not found. Building first...${NC}"
    ./scripts/build.sh
fi

# Create deployment directory
rm -rf "$DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR"

echo -e "${YELLOW}📋 Creating deployment package...${NC}"

# Copy essential files
cp bin/aws-connect-exporter "$DEPLOY_DIR/"
cp config.aws-connect-validated.yml "$DEPLOY_DIR/"
cp DEPLOYMENT_README.md "$DEPLOY_DIR/"

# Create a quick start script for the lab
cat > "$DEPLOY_DIR/run-exporter.sh" << 'EOF'
#!/bin/bash
#
# Quick start script for CentOS lab
#

set -euo pipefail

# Check if config file exists
if [[ ! -f "config.aws-connect-validated.yml" ]]; then
    echo "❌ Configuration file not found!"
    echo "Please ensure config.aws-connect-validated.yml is in this directory"
    exit 1
fi

# Get hostname and IP for labels
HOSTNAME=$(hostname)
IP_ADDRESS=$(hostname -I | awk '{print $1}' || echo "127.0.0.1")

echo "🚀 Starting AWS Connect Exporter..."
echo "   Hostname: $HOSTNAME"
echo "   IP Address: $IP_ADDRESS"
echo "   Config: config.aws-connect-validated.yml"
echo

# Check AWS credentials
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo "⚠️  Warning: AWS credentials not configured or accessible"
    echo "   Make sure to set AWS_PROFILE, AWS_ACCESS_KEY_ID, or use IAM role"
    echo
fi

# Run the exporter
./aws-connect-exporter \
    --web.listen-port=9090 \
    --labels.fqdn="$HOSTNAME" \
    --labels.location="centos-lab" \
    --labels.country="US" \
    --labels.ip-address="$IP_ADDRESS" \
    --config.file=config.aws-connect-validated.yml \
    --no-advertise \
    --log.level=info
EOF

chmod +x "$DEPLOY_DIR/run-exporter.sh"

# Create a health check script
cat > "$DEPLOY_DIR/check-health.sh" << 'EOF'
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
EOF

chmod +x "$DEPLOY_DIR/check-health.sh"

# Show package contents
echo -e "${GREEN}✅ Deployment package created!${NC}"
echo
echo -e "${BLUE}📂 Package contents ($DEPLOY_DIR):${NC}"
ls -lah "$DEPLOY_DIR"

echo
echo -e "${YELLOW}📋 Deployment Instructions:${NC}"
echo "1. Copy the entire deploy-package/ directory to your CentOS lab"
echo "2. Update config.aws-connect-validated.yml with your AWS Connect instance ARN"
echo "3. Set up AWS credentials (AWS_PROFILE, keys, or IAM role)"
echo "4. Run: cd deploy-package && ./run-exporter.sh"
echo "5. Health check: ./check-health.sh"
echo
echo -e "${GREEN}📖 See DEPLOYMENT_README.md for detailed instructions${NC}"

# Calculate package size
PACKAGE_SIZE=$(du -sh "$DEPLOY_DIR" | cut -f1)
echo "📦 Package size: $PACKAGE_SIZE"
EOF
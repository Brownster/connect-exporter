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

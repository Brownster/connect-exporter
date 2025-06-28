#!/bin/bash

echo "🔧 Debug: Mock API Connection Test"
echo "=================================="

# Set all necessary environment variables for mock mode
export ENV=development
export AWS_ACCESS_KEY_ID=mock-access-key
export AWS_SECRET_ACCESS_KEY=mock-secret-key
export AWS_SESSION_TOKEN=mock-session-token
export AWS_REGION=eu-west-2

echo "📡 Testing mock API server..."
curl -s http://localhost:3000/health

echo -e "\n🚀 Starting exporter with debug logging..."
echo "Environment variables set:"
echo "  ENV=$ENV"
echo "  AWS_REGION=$AWS_REGION"
echo "  AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID"
echo ""

/home/mbrown/connect-exporter/connect-exporter-optimized \
  --config.file=/home/mbrown/connect-exporter/config.kensington-test.yml \
  --log.level=debug \
  --no-advertise \
  --web.listen-port=8542 \
  --labels.fqdn=localhost \
  --labels.location=dev \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1
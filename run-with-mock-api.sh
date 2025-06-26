#!/bin/bash

echo "🚀 Starting AWS Connect Exporter with Mock API"
echo "=============================================="

# Set development mode
export ENV=development

# Set mock AWS credentials to prevent SDK from trying to get real credentials
export AWS_ACCESS_KEY_ID=mock-access-key-id
export AWS_SECRET_ACCESS_KEY=mock-secret-access-key
export AWS_SESSION_TOKEN=mock-session-token
export AWS_REGION=eu-west-2

# Force AWS SDK to use HTTP instead of HTTPS for mock API
export AWS_USE_SSL=false
export AWS_ENDPOINT_URL_CONNECT=http://localhost:3000

# Optional: Set mock endpoint explicitly (though default should work)
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000

# Find the exporter binary
EXPORTER_PATH=""
if [ -f "./connect-exporter-optimized" ]; then
    EXPORTER_PATH="./connect-exporter-optimized"
elif [ -f "./connect-exporter" ]; then
    EXPORTER_PATH="./connect-exporter"
elif [ -f "/home/mbrown/connect-exporter/connect-exporter-optimized" ]; then
    EXPORTER_PATH="/home/mbrown/connect-exporter/connect-exporter-optimized"
elif [ -f "/home/mbrown/connect-exporter/connect-exporter" ]; then
    EXPORTER_PATH="/home/mbrown/connect-exporter/connect-exporter"
else
    echo "❌ Could not find exporter binary! Please build it first:"
    echo "   go build -o connect-exporter ./cmd/connect-exporter_exporter"
    exit 1
fi

echo "📡 Testing mock API connection..."
if curl -s http://localhost:3000/health >/dev/null; then
    echo "✅ Mock API is running at http://localhost:3000"
else
    echo "❌ Mock API is not running! Please start it first:"
    echo "   cd docs/tools/aws-mock-api-main && node server.js"
    exit 1
fi

echo ""
echo "🔧 Environment variables set:"
echo "  ENV=$ENV"
echo "  AWS_REGION=$AWS_REGION"
echo "  AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID"
echo "  AWS_SECRET_ACCESS_KEY=[HIDDEN]"
echo "  AWS_CONNECT_MOCK_ENDPOINT=$AWS_CONNECT_MOCK_ENDPOINT"
echo ""

# Check for config file
CONFIG_FILE=""
if [ -f "./config.kensington-test.yml" ]; then
    CONFIG_FILE="./config.kensington-test.yml"
elif [ -f "/home/mbrown/connect-exporter/config.kensington-test.yml" ]; then
    CONFIG_FILE="/home/mbrown/connect-exporter/config.kensington-test.yml"
else
    echo "❌ Could not find config.kensington-test.yml!"
    exit 1
fi

echo "🚀 Starting exporter..."
echo "📁 Using exporter: $EXPORTER_PATH"
echo "📄 Using config: $CONFIG_FILE"
echo ""

# CRITICAL: Must specify --region=eu-west-2 to override the us-east-1 default!
$EXPORTER_PATH \
  --config.file=$CONFIG_FILE \
  --region=eu-west-2 \
  --log.level=debug \
  --no-advertise \
  --web.listen-port=8542 \
  --labels.fqdn=localhost \
  --labels.location=dev \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1
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

# Optional: Set mock endpoint explicitly (though default should work)
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000

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

echo "🚀 Starting exporter..."
/home/mbrown/connect-exporter/connect-exporter-optimized \
  --config.file=/home/mbrown/connect-exporter/config.kensington-test.yml \
  --log.level=debug \
  --no-advertise \
  --web.listen-port=8542 \
  --labels.fqdn=localhost \
  --labels.location=dev \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1
#!/bin/bash

echo "🔧 Fixing Lab Server Paths"
echo "=========================="

# Lab server project directory
LAB_DIR="/home/mbrown/connect-exporter"

echo "📁 Working in: $LAB_DIR"
cd "$LAB_DIR" || exit 1

echo ""
echo "🔄 Step 1: Pull latest changes..."
git pull origin main

echo ""
echo "🏗️  Step 2: Build with latest HTTP fixes..."
rm -f connect-exporter-*
go build -o connect-exporter-lab ./cmd/connect-exporter_exporter

if [ ! -f "connect-exporter-lab" ]; then
    echo "❌ Build failed! Check Go installation"
    exit 1
fi

echo "✅ Built connect-exporter-lab with HTTP fixes"

echo ""
echo "🔧 Step 3: Create lab-specific run script..."
cat > run-lab-mock-api.sh << 'EOF'
#!/bin/bash

echo "🚀 Lab Server: AWS Connect Exporter with Mock API"
echo "================================================="

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
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000

echo "📡 Testing mock API connection..."
if curl -s http://localhost:3000/health >/dev/null; then
    echo "✅ Mock API is running at http://localhost:3000"
else
    echo "❌ Mock API is not running! Please start it first:"
    echo "   cd /home/mbrown/connect-exporter/docs/tools/aws-mock-api-main && node server.js &"
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
echo "📁 Using exporter: /home/mbrown/connect-exporter/connect-exporter-lab"
echo "📄 Using config: /home/mbrown/connect-exporter/config.kensington-test.yml"
echo ""

# Use absolute paths for lab server
/home/mbrown/connect-exporter/connect-exporter-lab \
  --config.file=/home/mbrown/connect-exporter/config.kensington-test.yml \
  --region=eu-west-2 \
  --log.level=debug \
  --no-advertise \
  --web.listen-port=8542 \
  --labels.fqdn=monitoring-rocky \
  --labels.location=lab \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1
EOF

chmod +x run-lab-mock-api.sh

echo "✅ Created run-lab-mock-api.sh with absolute paths"

echo ""
echo "📡 Step 4: Check mock API status..."
if curl -s http://localhost:3000/health >/dev/null; then
    echo "✅ Mock API is running"
else
    echo "⚠️  Starting mock API..."
    cd docs/tools/aws-mock-api-main
    MOCK_ERROR_RATE=0.0 node server.js &
    sleep 3
    cd ../../..
    echo "✅ Started mock API"
fi

echo ""
echo "🎯 Lab server is now ready!"
echo ""
echo "To run the exporter:"
echo "  cd /home/mbrown/connect-exporter"
echo "  ./run-lab-mock-api.sh"
echo ""
echo "The new binary includes the HTTP fixes for the 302 redirect issue."
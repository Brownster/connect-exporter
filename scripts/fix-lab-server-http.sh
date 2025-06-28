#!/bin/bash

echo "🔧 Lab Server HTTP Fix Script"
echo "============================="
echo ""

# Check if running on lab server
if [[ ! "$HOSTNAME" == *"monitoring-rocky"* ]] && [[ ! "$USER" == "mbrown" ]]; then
    echo "⚠️  This script should be run on the lab server (monitoring-rocky)"
    echo "   Copy this script to the lab server and run it there"
    exit 1
fi

LAB_DIR="/home/mbrown/connect-exporter"
echo "📁 Working directory: $LAB_DIR"

# Check we're in the right directory
if [ ! -f "$LAB_DIR/connectors/aws_connect_connector.go" ]; then
    echo "❌ Not in connect-exporter directory!"
    echo "   Please run: cd $LAB_DIR"
    exit 1
fi

cd "$LAB_DIR" || exit 1

echo ""
echo "🔍 Step 1: Check current binary status..."
if [ -f "connect-exporter-optimized" ]; then
    echo "📦 Found connect-exporter-optimized"
    ./connect-exporter-optimized --version 2>&1 | head -3
else
    echo "❌ No connect-exporter-optimized found"
fi

echo ""
echo "🔍 Step 2: Check for HTTP fixes in source code..."
if grep -q "UseFIPSEndpoint.*Disabled" connectors/aws_connect_connector.go; then
    echo "✅ HTTP fixes are present in source code"
else
    echo "❌ HTTP fixes missing in source code!"
    echo "   The copied folder may not have the latest fixes"
    exit 1
fi

echo ""
echo "🏗️  Step 3: Build new binary with HTTP fixes..."
echo "   Removing old binaries..."
rm -f connect-exporter connect-exporter-optimized connect-exporter-fixed connect-exporter-latest

echo "   Building connect-exporter-fixed..."
go build -o connect-exporter-fixed ./cmd/connect-exporter_exporter

if [ ! -f "connect-exporter-fixed" ]; then
    echo "❌ Build failed! Check Go installation"
    exit 1
fi

echo "✅ Built connect-exporter-fixed with HTTP fixes"

echo ""
echo "🔧 Step 4: Test the fix..."
echo "   Checking if mock API is running..."
if ! curl -s http://localhost:3000/health >/dev/null; then
    echo "   Starting mock API..."
    cd docs/tools/aws-mock-api-main
    MOCK_ERROR_RATE=0.0 node server.js &
    MOCK_PID=$!
    sleep 3
    cd ../../..
    echo "   Started mock API (PID: $MOCK_PID)"
else
    echo "   Mock API already running"
    MOCK_PID=""
fi

echo ""
echo "🧪 Step 5: Quick test (10 seconds)..."

# Set environment variables for test
export ENV=development
export AWS_ACCESS_KEY_ID=mock-access-key-id  
export AWS_SECRET_ACCESS_KEY=mock-secret-access-key
export AWS_SESSION_TOKEN=mock-session-token
export AWS_REGION=eu-west-2
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000

echo "   Starting test exporter..."
./connect-exporter-fixed \
  --config.file=./config.kensington-test.yml \
  --region=eu-west-2 \
  --log.level=info \
  --no-advertise \
  --web.listen-port=8543 &

TEST_PID=$!
echo "   Started test exporter (PID: $TEST_PID)"

# Wait for startup
sleep 12

echo ""
echo "🔍 Step 6: Check results..."
METRICS_COUNT=$(curl -s http://127.0.0.1:8543/metrics 2>/dev/null | grep -c "aws_connect_exporter_" || echo "0")
echo "   Metrics found: $METRICS_COUNT"

if [ "$METRICS_COUNT" -gt 0 ]; then
    echo "✅ SUCCESS! HTTP fixes are working!"
    curl -s http://127.0.0.1:8543/metrics | grep "aws_connect_exporter_" | head -3
    echo ""
    echo "🎯 Lab server is now fixed!"
    echo ""
    echo "To use the fixed binary:"
    echo "   1. Use ./connect-exporter-fixed instead of ./connect-exporter-optimized"
    echo "   2. Or replace the old binary: mv connect-exporter-fixed connect-exporter-optimized"
else
    echo "❌ Still getting errors. Check the logs above for details."
fi

echo ""
echo "🛑 Cleanup: Stopping test processes..."
kill $TEST_PID 2>/dev/null || true
if [ -n "$MOCK_PID" ]; then
    kill $MOCK_PID 2>/dev/null || true
fi

echo ""
echo "📋 Summary:"
echo "   - Built connect-exporter-fixed with HTTP fixes"
echo "   - Fixed binary should resolve 302 errors in development mode"
echo "   - Use ./connect-exporter-fixed for development work"
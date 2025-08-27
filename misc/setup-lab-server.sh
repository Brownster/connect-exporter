#!/bin/bash

echo "🔧 Lab Server Setup Script"
echo "=========================="
echo "This script will update the lab server with the latest fixes"
echo ""

LAB_PROJECT_DIR="/home/mbrown/connect-exporter"

echo "📁 Target directory: $LAB_PROJECT_DIR"
echo ""

# Check if we're running on the lab server
if [[ "$HOSTNAME" == *"monitoring-rocky"* ]] || [[ "$USER" == "mbrown" ]]; then
    echo "✅ Running on lab server"
    CURRENT_DIR="$LAB_PROJECT_DIR"
else
    echo "⚠️  Not on lab server, creating transfer script..."
    cat > transfer-to-lab.sh << 'EOF'
#!/bin/bash
echo "🚀 Transfer Latest Code to Lab Server"
echo "===================================="

# Replace with your actual lab server details
LAB_SERVER="mbrown@monitoring-rocky"
LAB_PROJECT_DIR="/home/mbrown/connect-exporter"

echo "📦 Syncing latest code to lab server..."
rsync -av --delete \
  --exclude='.git' \
  --exclude='*.log' \
  --exclude='node_modules' \
  --exclude='*-exporter' \
  ./ $LAB_SERVER:$LAB_PROJECT_DIR/

echo "🔗 SSH to lab server and run setup..."
ssh $LAB_SERVER "cd $LAB_PROJECT_DIR && ./setup-lab-server.sh"
EOF
    chmod +x transfer-to-lab.sh
    echo "✅ Created transfer-to-lab.sh script"
    echo "   Run this script to sync code to lab server"
    exit 0
fi

# We're on the lab server, proceed with setup
cd "$CURRENT_DIR" || exit 1

echo "🔄 Step 1: Pull latest changes from GitHub..."
git pull origin main

echo ""
echo "🔧 Step 2: Remove old binaries..."
rm -f connect-exporter connect-exporter-optimized connect-exporter-fixed

echo ""
echo "🏗️  Step 3: Build latest exporter with HTTP fixes..."
go build -o connect-exporter-latest ./cmd/connect-exporter_exporter

if [ ! -f "connect-exporter-latest" ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Built connect-exporter-latest successfully"

echo ""
echo "🔧 Step 4: Update run script to use latest binary..."
sed -i 's/connect-exporter-optimized/connect-exporter-latest/g' run-with-mock-api.sh
sed -i 's/connect-exporter/connect-exporter-latest/g' run-with-mock-api.sh

echo ""
echo "📡 Step 5: Verify mock API is running..."
if curl -s http://localhost:3000/health >/dev/null; then
    echo "✅ Mock API is running"
else
    echo "⚠️  Mock API not running, starting it..."
    cd docs/tools/aws-mock-api-main
    npm install >/dev/null 2>&1
    MOCK_ERROR_RATE=0.0 node server.js &
    sleep 3
    cd ../../..
    echo "✅ Started mock API"
fi

echo ""
echo "🧪 Step 6: Test the fix..."
echo "Running 30-second test..."

./run-with-mock-api.sh &
EXPORTER_PID=$!

echo "Started exporter PID: $EXPORTER_PID"
sleep 35

echo ""
echo "🔍 Checking for metrics..."
METRICS_COUNT=$(curl -s http://127.0.0.1:8542/metrics | grep -c "aws_connect_exporter")
echo "Metrics found: $METRICS_COUNT"

if [ "$METRICS_COUNT" -gt 0 ]; then
    echo "✅ SUCCESS! Mock API is working!"
    curl -s http://127.0.0.1:8542/metrics | grep "aws_connect_exporter" | head -3
else
    echo "❌ Still not working, checking logs..."
    echo "Check the exporter logs above for errors"
fi

echo ""
echo "🛑 Stopping test exporter..."
kill $EXPORTER_PID 2>/dev/null

echo ""
echo "🎯 Setup complete!"
echo ""
echo "To run the exporter:"
echo "  cd $LAB_PROJECT_DIR"
echo "  ./run-with-mock-api.sh"
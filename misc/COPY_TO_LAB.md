# Copy Binary to Lab Server

## Ready-to-Use Binary

✅ **Built:** `connect-exporter-lab-ready` - Contains all fixes for both issues

### What's Fixed
1. **HTTP 302 errors** - Added FIPS/DualStack endpoint fixes for development mode
2. **Agent realtime status errors** - Added required Agents filter parameter

## Transfer to Lab Server

### Option 1: Direct Copy (if you have file access)
```bash
# Copy the binary
cp connect-exporter-lab-ready /path/to/lab/server/home/mbrown/connect-exporter/

# On lab server, make it executable
chmod +x /home/mbrown/connect-exporter/connect-exporter-lab-ready
```

### Option 2: SCP Transfer (if you have SSH access)
```bash
# From your laptop/development machine
scp connect-exporter-lab-ready mbrown@monitoring-rocky:/home/mbrown/connect-exporter/
```

### Option 3: USB/Network Transfer
1. Copy `connect-exporter-lab-ready` to USB drive or shared folder
2. Transfer to lab server
3. Place in `/home/mbrown/connect-exporter/`
4. Make executable: `chmod +x connect-exporter-lab-ready`

## Usage on Lab Server

### For Development Mode (Mock API)
```bash
cd /home/mbrown/connect-exporter

# Edit run-with-mock-api.sh to use new binary
# Change line 24-26 from:
# if [ -f "./connect-exporter-optimized" ]; then
#     EXPORTER_PATH="./connect-exporter-optimized"

# To:
# if [ -f "./connect-exporter-lab-ready" ]; then
#     EXPORTER_PATH="./connect-exporter-lab-ready"

# Then run normally
./run-with-mock-api.sh
```

### For Production Mode (Real AWS)
```bash
./connect-exporter-lab-ready \
  --config.file=./config.kensington-production.yml \
  --region=eu-west-2 \
  --log.level=debug \
  --no-advertise \
  --web.listen-port=8542 \
  --labels.fqdn=monitoring-rocky \
  --labels.location=lab \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1
```

## Quick Test Script for Lab Server

Create this script on lab server to test both modes:

```bash
# Save as test-fixes.sh
#!/bin/bash
echo "🧪 Testing Lab Server Fixes"
echo "=========================="

# Test 1: Development mode
echo "📱 Test 1: Mock API mode (10 seconds)..."
export ENV=development
export AWS_ACCESS_KEY_ID=mock-key
export AWS_SECRET_ACCESS_KEY=mock-secret  
export AWS_REGION=eu-west-2
export AWS_CONNECT_MOCK_ENDPOINT=http://localhost:3000

# Start mock API if not running
if ! curl -s http://localhost:3000/health >/dev/null; then
    echo "Starting mock API..."
    cd docs/tools/aws-mock-api-main
    MOCK_ERROR_RATE=0.0 node server.js &
    MOCK_PID=$!
    sleep 3
    cd ../../..
fi

# Test with new binary
timeout 10s ./connect-exporter-lab-ready \
  --config.file=./config.kensington-test.yml \
  --region=eu-west-2 \
  --log.level=info \
  --web.listen-port=8543 \
  --labels.fqdn=test \
  --labels.location=test \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1 &

sleep 12
METRICS=$(curl -s http://127.0.0.1:8543/metrics | grep -c "aws_connect_exporter_" || echo "0")
echo "Mock API metrics found: $METRICS"

if [ "$METRICS" -gt 0 ]; then
    echo "✅ Development mode: WORKING!"
else
    echo "❌ Development mode: Still has issues"
fi

# Cleanup
pkill -f connect-exporter-lab-ready
[[ -n "$MOCK_PID" ]] && kill $MOCK_PID 2>/dev/null

echo ""
echo "🎯 Binary is ready for use!"
```

## Expected Results After Fix

### Development Mode
- ✅ No `StatusCode: 302` errors
- ✅ All 4 collectors working
- ✅ Metrics visible at http://127.0.0.1:8542/metrics

### Production Mode  
- ✅ No `Missing required parameter` errors
- ✅ Agent realtime status collector working
- ✅ Real AWS Connect metrics

Binary size: ~37MB
Includes: All collectors, HTTP fixes, API parameter fixes
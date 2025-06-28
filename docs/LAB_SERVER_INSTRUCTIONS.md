# Lab Server Instructions

## Quick Fix Summary

You have two separate issues on the lab server:

### 1. Development Mode (Mock API) - HTTP 302 Errors

**Problem**: Old binary missing HTTP fixes
**Solution**: Build new binary with HTTP fixes

```bash
# On lab server
cd /home/mbrown/connect-exporter
./fix-lab-server-http.sh
```

This script will:
- Check for HTTP fixes in source code
- Build `connect-exporter-fixed` with latest fixes  
- Test the fix automatically
- Provide clear success/failure status

**Then update your run script to use the fixed binary:**
```bash
# Edit run-with-mock-api.sh line 24-26:
# Change from:
if [ -f "./connect-exporter-optimized" ]; then
    EXPORTER_PATH="./connect-exporter-optimized"

# Change to:
if [ -f "./connect-exporter-fixed" ]; then
    EXPORTER_PATH="./connect-exporter-fixed"
```

### 2. Real AWS API - Agent Realtime Status Error  

**Problem**: Missing required parameter in GetCurrentUserData API call
**Solution**: The fix is already in the source code (added `Agents: []string{}` filter)

After rebuilding with the fix script above, this issue will also be resolved.

## Root Cause Analysis

### HTTP 302 Errors (Development Mode)
- Lab server binary: `connect-exporter-optimized` built before HTTP fixes
- Missing lines 405-406 in connector that disable FIPS/DualStack endpoints
- AWS SDK making HTTPS requests to HTTP-only mock API

### Agent Realtime Status Errors (Both Modes)
- AWS GetCurrentUserData API requires at least one filter parameter
- Previous code had empty filters: `Filters: &types.UserDataFilters{}`
- Fixed with: `Filters: &types.UserDataFilters{Agents: []string{}}`

## Step-by-Step Lab Server Fix

1. **Run the fix script:**
   ```bash
   cd /home/mbrown/connect-exporter
   ./fix-lab-server-http.sh
   ```

2. **Update run script (if needed):**
   ```bash
   # Replace line 24 in run-with-mock-api.sh
   sed -i 's/connect-exporter-optimized/connect-exporter-fixed/g' run-with-mock-api.sh
   ```

3. **Test development mode:**
   ```bash
   # Start mock API
   cd docs/tools/aws-mock-api-main
   MOCK_ERROR_RATE=0.0 node server.js &
   cd ../../..
   
   # Run exporter
   ./run-with-mock-api.sh
   ```

4. **Test real AWS (if credentials available):**
   ```bash
   ./connect-exporter-fixed \
     --config.file=./config.kensington-production.yml \
     --region=eu-west-2 \
     --log.level=debug \
     --no-advertise \
     --web.listen-port=8542
   ```

## Expected Results

### Development Mode (Mock API)
- ✅ No more `StatusCode: 302` errors
- ✅ All 4 collectors working:
  - `realtime_queue` 
  - `agent_status`
  - `agent_realtime_status` 
  - `historical_queue`
- ✅ Metrics appearing at http://127.0.0.1:8542/metrics

### Real AWS Mode  
- ✅ No more `Missing required parameter` errors
- ✅ Agent realtime status collector working
- ✅ Real metrics from AWS Connect instance

## Verification Commands

```bash
# Check metrics are being collected
curl -s http://127.0.0.1:8542/metrics | grep aws_connect_exporter | head -5

# Check for errors in logs
# Should see "Successfully retrieved" messages instead of errors

# Check binary version
./connect-exporter-fixed --version
```

## Files Modified
- `collectors/agent_realtime_status_collector.go` - Added required Agents filter
- `connectors/aws_connect_connector.go` - HTTP fixes for development mode  
- `fix-lab-server-http.sh` - Automated lab server fix script

The lab server should now work identically to your laptop for both development and production modes.
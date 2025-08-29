# Debug Guide: Mock API 404 Errors

## Issue Summary
The connect-exporter works locally but fails in company lab with:
- 404 errors for routing profile queues endpoints
- ValidationException errors for GetCurrentMetricData

## Root Cause Analysis

### 1. Routing Profile Endpoints (404 errors)
The errors show:
```
Unknown operation: /routing-profiles/f604177c-b2ab-4c76-9033-635d195b2772/rp-sales/queues
```

**Expected behavior**: Mock API should handle this URL pattern and return queue data for routing profile `rp-sales`.

### 2. GetCurrentMetricData (400 validation errors)  
The errors show:
```
ValidationException: Validation error occurred
```

**Expected behavior**: Mock API should validate metrics and return simulated data.

## Solutions

### Quick Fix: Verify Mock API Configuration

1. **Check if mock API is running correctly**:
   ```bash
   curl -v http://localhost:3000/health
   ```

2. **Test routing profile endpoint directly**:
   ```bash
   curl -X POST http://localhost:3000/ \
     -H "Content-Type: application/x-amz-json-1.1" \
     -H "X-Amz-Target: Connect_20170801.ListRoutingProfileQueues" \
     -d '{
       "InstanceId": "f604177c-b2ab-4c76-9033-635d195b2772",
       "RoutingProfileId": "rp-sales"
     }'
   ```

3. **Test GetCurrentMetricData endpoint**:
   ```bash
   curl -X POST http://localhost:3000/ \
     -H "Content-Type: application/x-amz-json-1.1" \
     -H "X-Amz-Target: Connect_20170801.GetCurrentMetricData" \
     -d '{
       "InstanceId": "f604177c-b2ab-4c76-9033-635d195b2772",
       "Filters": {
         "Queues": ["queue-sales-01"],
         "Channels": ["VOICE"]
       },
       "CurrentMetrics": [
         {"Name": "CONTACTS_IN_QUEUE"},
         {"Name": "AGENTS_STAFFED"}
       ]
     }'
   ```

### File Verification Checklist

Ensure these files were copied correctly to company lab:

1. **Core mock server**: `docs/tools/aws-mock-api-main/server.js`
2. **Metrics configuration**: `docs/tools/aws-mock-api-main/metrics-config.json`  
3. **Package dependencies**: `docs/tools/aws-mock-api-main/package.json`

### Instance ID Validation

The mock API expects instance ID: `f604177c-b2ab-4c76-9033-635d195b2772`

Verify your exporter configuration uses this exact instance ID.

### Mock API Startup Command

Ensure the mock API is started with:
```bash
cd docs/tools/aws-mock-api-main
node server.js
```

The server should show:
```
Loaded metric config from /path/to/metrics-config.json
Mock AWS Connect API Server running on port 3000
Instance ID: f604177c-b2ab-4c76-9033-635d195b2772
```

## Expected Working State

When working correctly, you should see logs like:
```
=== Processing Routing Profile Queues Request ===
📍 Extracted instanceId and rpId from AWS SDK path: f604177c-b2ab-4c76-9033-635d195b2772 rp-sales
✅ Found routing profile: rp-sales
```

```
=== Processing Metrics Request ===
📍 Using instanceId from body: f604177c-b2ab-4c76-9033-635d195b2772
✅ Metrics validation passed
✅ Filters validation passed
📊 Generating metrics...
```
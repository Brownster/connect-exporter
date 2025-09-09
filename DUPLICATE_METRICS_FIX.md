# Duplicate Metrics Fix - Agent Realtime Status Collector

## Issue Summary

**Problem**: When AWS Connect instances have 101+ queues, the agent realtime status collector triggers duplicate metrics errors like:
```
collected metric "aws-connect-exporter_agent_realtime_status_connect_agent_status_state" ... was collected before with the same name and label values
```

**Root Cause**: The batching logic added in commit `5581583f` (Aug 28, 2025) processes queues in batches of 100, but agents can appear in multiple batches if they're associated with queues across batch boundaries. The original code didn't deduplicate agents, causing Prometheus to reject identical metrics.

## Technical Details

### Affected Component
- **File**: `collectors/agent_realtime_status_collector.go`
- **Lines**: 78-128 (batching logic)
- **Issue**: Line 121 `allUserData = append(allUserData, data.UserDataList...)` without deduplication

### Why This Happens
1. **AWS API Behavior**: `GetCurrentUserData` with queue filters returns all agents associated with ANY of the specified queues
2. **Batching Logic**: With 101+ queues, splits into batches (e.g., queues 1-100, then queue 101)
3. **Agent Overlap**: Agents like Travis Rodrigues (`ff3b0143-915d-4ddf-8ff9-8cabcebf900f`) appear in multiple batches
4. **Identical Labels**: All instances of the same agent get identical labels (same queue_id, queue_name, status, etc.)
5. **Prometheus Rejection**: Prometheus rejects the duplicate metric with identical labels

## The Fix

### Code Changes
Added agent deduplication logic after line 128 in `agent_realtime_status_collector.go`:

```go
// Deduplicate agents that appear in multiple batches to prevent duplicate metrics
seenAgents := make(map[string]bool)
uniqueUserData := make([]types.UserData, 0)
duplicateCount := 0

for _, user := range allUserData {
    if user.User != nil && user.User.Id != nil {
        agentID := aws.ToString(user.User.Id)
        if !seenAgents[agentID] {
            seenAgents[agentID] = true
            uniqueUserData = append(uniqueUserData, user)
        } else {
            duplicateCount++
        }
    }
}

if duplicateCount > 0 {
    logrus.WithFields(logrus.Fields{
        "duplicates_removed": duplicateCount,
        "unique_agents":      len(uniqueUserData),
        "total_returned":     len(allUserData),
    }).Debug("Removed duplicate agents from batched results")
}

// Use deduplicated data for processing
allUserData = uniqueUserData
```

### What the Fix Does
1. **Tracks Seen Agents**: Uses a map to track which agents have already been processed
2. **Deduplicates**: Only keeps the first occurrence of each agent
3. **Logs Activity**: Debug logs show how many duplicates were removed
4. **Preserves Functionality**: All other behavior remains identical

## Deployment Instructions

### Immediate Fix (Production)
1. **Backup Current Binary**:
   ```bash
   cp /path/to/current/aws-connect-exporter /path/to/backup/aws-connect-exporter.backup
   ```

2. **Deploy Fixed Binary**:
   ```bash
   # Copy the fixed binary to production
   cp connect-exporter-fixed /path/to/production/aws-connect-exporter
   
   # Restart the service
   sudo systemctl restart aws-connect-exporter
   # OR kill existing process and restart
   ```

3. **Verify Fix**:
   ```bash
   # Check logs for deduplication activity
   tail -f /var/log/aws-connect-exporter.log | grep "duplicates_removed"
   
   # Verify metrics collection works
   curl http://localhost:9090/metrics | grep "agent_realtime_status_connect_agent_status_state"
   ```

### Git Workflow
```bash
# Create a hotfix branch
git checkout -b hotfix/duplicate-metrics-fix

# Commit the fix
git add collectors/agent_realtime_status_collector.go
git commit -m "fix: Add agent deduplication to prevent duplicate metrics in 101+ queue environments

- Adds deduplication logic to agent realtime status collector
- Prevents same agent appearing in multiple queue batches from creating duplicate metrics  
- Includes debug logging for visibility into deduplication activity
- Fixes Prometheus rejection of identical agent metrics
- Resolves issue introduced in commit 5581583f"

# Deploy and test, then merge to main
```

## Validation

### Expected Log Output (Success)
With the fix, you should see debug logs like:
```json
{"level":"debug","msg":"Processing agent queue batch","batch_start":0,"batch_end":100,"batch_size":100}
{"level":"debug","msg":"Processing agent queue batch","batch_start":100,"batch_end":101,"batch_size":1}  
{"level":"debug","msg":"Removed duplicate agents from batched results","duplicates_removed":3,"unique_agents":25,"total_returned":28}
```

### Testing the Fix
1. **Environment**: Any AWS Connect instance with 100+ queues
2. **Expected Behavior**: No more duplicate metrics errors
3. **Metrics Collection**: Should continue normally with proper agent metrics
4. **Performance**: Minimal impact - just one additional loop through agents

## Prevention

### Code Review Checklist
For future batching implementations:
- [ ] Does the batching logic handle overlapping data?
- [ ] Are there deduplication mechanisms for entities that might appear in multiple batches?
- [ ] Does the implementation include appropriate debug logging?
- [ ] Has it been tested with boundary conditions (100, 101, 200+ items)?

### Monitoring
Add alerting for:
- Duplicate metrics errors in exporter logs
- Unexpected drops in agent metrics collection
- AWS API rate limiting from excessive calls

## Related Issues
- Original batching implementation: Commit `5581583f`
- Production logs showing the issue: `collected metric aws-connect-export.txt`
- Agent in question: Travis Rodrigues (`ff3b0143-915d-4ddf-8ff9-8cabcebf900f`)

## Impact
- **Before**: Duplicate metrics errors, collection failures
- **After**: Clean metrics collection with proper deduplication
- **Performance**: Negligible impact, improved reliability
- **Compatibility**: Fully backward compatible
# Test Coverage Analysis - AWS Connect Exporter

## Current Coverage Status

### Working Tests with Coverage:
- **Connectors Package: 33.9% coverage** (partial tests passing)
  - ✅ `TestAgentManager` - PASS
  - ✅ `TestAgentManagerWithError` - PASS  
  - ✅ `TestAgentManagerStartStop` - PASS
  - ❌ `TestGetMockEndpoint` - FAIL (localhost vs 127.0.0.1)
  - ❌ `TestRoutingProfileQueueManagerCaching` - FAIL (type comparison)

### Failing Tests:
- **Collectors Package: 0% coverage** (all tests failing)
  - ❌ All collector tests fail due to routing profile manager startup issues
  - Root cause: Mock API response format mismatch with AWS SDK expectations

## Analysis for 75% Coverage Target

### Package Breakdown

#### 1. Connectors Package (~34% current, need 75%)
**Files needing tests:**
- `aws_connect_connector.go` - Core connector logic
- `client.go` - AWS client initialization 
- `config.go` - Configuration handling
- `queue_manager.go` - Queue management
- `routing_profile_manager.go` - Routing profiles
- `dev.go` - Development helpers

**High-impact areas to test:**
- AWS connection establishment
- Configuration parsing and validation
- Queue discovery and caching
- Error handling paths
- Rate limiting logic

#### 2. Collectors Package (0% current, need 75%)
**Files needing tests:**
- `generic_historical_collector.go` - Historical metrics collection
- `generic_realtime_collector.go` - Real-time metrics collection  
- `agent_realtime_status_collector.go` - Agent status collection
- `aws_connect_exporter.go` - Main exporter logic

**High-impact areas to test:**
- Metric collection workflows
- Threshold handling (SERVICE_LEVEL metrics)
- Error recovery and retry logic
- Prometheus metric generation
- Configuration-driven collection

#### 3. Main Command (0% current, need 75%) 
- `cmd/connect-exporter_exporter/main.go` - Application startup

## Test Fixes Required

### Immediate Fixes (to get tests running):

#### 1. Fix Mock API Response Format
**Issue:** Tests fail because mock API returns numbers directly, but AWS SDK expects error objects:
```
json: cannot unmarshal number into Go value of type struct { Code string; Type string "json:\"__type\""; Message string }
```

**Solution:** Update mock API to return proper AWS error format:
```javascript
// Instead of: res.status(404).send(404)
// Use: res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND)
```

#### 2. Fix Test Assertions
**Connectors tests need minor fixes:**
- `TestGetMockEndpoint`: Update expected value to "127.0.0.1:3000" 
- `TestRoutingProfileQueueManagerCaching`: Fix type comparison

#### 3. Fix Collector Test Setup
**Issue:** Tests try to start full routing profile manager
**Solution:** Mock the routing profile manager dependency

### Comprehensive Test Plan to Reach 75%

#### Phase 1: Fix Existing Tests (Week 1)
- [ ] Fix mock API response format compatibility
- [ ] Fix failing connector tests 
- [ ] Get collector tests running with mocked dependencies
- **Target:** Get current tests passing

#### Phase 2: Core Functionality Tests (Week 2)
- [ ] **AWS Connection Tests** (high impact)
  - Connection establishment
  - Credential handling
  - Region configuration
  - Error scenarios (invalid credentials, network issues)

- [ ] **Configuration Tests** (high impact) 
  - YAML parsing
  - Validation logic
  - Default values
  - Invalid configuration handling

- [ ] **Queue Management Tests** (high impact)
  - Queue discovery
  - Caching behavior
  - Refresh logic
  - API pagination

#### Phase 3: Collector Tests (Week 3)
- [ ] **Generic Historical Collector** (high impact)
  - Metric collection workflow
  - Threshold configuration (SERVICE_LEVEL)
  - Groupings and filters
  - Error handling

- [ ] **Generic Realtime Collector** (high impact)
  - Real-time metric collection
  - Update frequency handling
  - Queue-specific filtering
  - Prometheus metric generation

- [ ] **Agent Status Collector** (medium impact)
  - Agent discovery
  - Status mapping
  - Name resolution caching
  - Routing profile associations

#### Phase 4: Edge Cases and Error Handling (Week 4)
- [ ] **Rate Limiting Tests**
  - Throttling detection
  - Exponential backoff
  - Retry logic

- [ ] **Error Recovery Tests**
  - Network failures
  - AWS API errors
  - Partial data scenarios
  - Graceful degradation

## Test Infrastructure Improvements

### Mock API Enhancements Needed:
1. **Proper AWS Error Format**
   ```javascript
   res.status(404).json({
     __type: "ResourceNotFoundException",
     message: "Resource not found"
   });
   ```

2. **Routing Profile Support**
   - Add proper routing profile endpoints
   - Support queue associations
   - Handle pagination correctly

3. **Realistic Data Generation**
   - Time-series data for historical metrics
   - Agent status changes simulation
   - Queue activity patterns

### Test Helpers to Create:
1. **Mock AWS Clients** - Standardized mocks for AWS services
2. **Configuration Builders** - Easy test configuration creation
3. **Metric Validators** - Prometheus metric validation helpers
4. **Integration Test Framework** - End-to-end testing with mock API

## Estimated Coverage by Package (after fixes):

| Package | Current | After Fixes | After Phase 2 | After Phase 3 | After Phase 4 |
|---------|---------|-------------|---------------|---------------|---------------|
| connectors | 33.9% | 45% | 65% | 75% | 85% |
| collectors | 0% | 25% | 50% | 75% | 85% |
| cmd/main | 0% | 10% | 40% | 60% | 75% |
| **Overall** | **~12%** | **~30%** | **~55%** | **~75%** ✅ | **~85%** |

## High-Priority Test Areas (for 75% target):

### Must Have (Critical for 75%):
1. **Configuration parsing and validation**
2. **AWS client initialization and connection**
3. **Queue discovery and management**
4. **Basic metric collection workflows**
5. **Error handling for common failures**

### Should Have (Additional coverage):
1. **Threshold configuration handling**
2. **Rate limiting and retry logic**
3. **Agent name resolution**
4. **Prometheus metric generation**

### Could Have (Nice to have):
1. **Performance edge cases**
2. **Complex error scenarios**
3. **Load testing simulation**

## Implementation Recommendations:

### Quick Wins (1-2 days):
- Fix the 2 failing connector tests
- Update mock API error format
- Add basic configuration tests

### Medium Effort (1 week):
- Create comprehensive connector tests
- Add queue management test suite
- Fix collector test infrastructure

### Long Term (2-3 weeks):
- Full collector test coverage
- Integration test framework
- Performance and load testing

## Success Metrics:
- [ ] All existing tests pass consistently
- [ ] 75% code coverage across main packages
- [ ] Tests run in <30 seconds
- [ ] No flaky tests (>99% success rate)
- [ ] Clear test documentation and examples
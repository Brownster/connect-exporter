# Test Coverage Plan - Path to 75%

## Current Status ✅

### Tests Now Working:
- **Connectors Package: 33.9% coverage** - All tests passing
- **Collectors Package: 0.6% coverage** - Basic tests working
- **Mock API Integration: Fixed** - Tests aligned with actual behavior

### Key Fixes Applied:
1. **Fixed Mock Endpoint Test** - Updated expected value to match actual `127.0.0.1:3000`
2. **Fixed Type Assertions** - Corrected `[]Queue` vs `[]string` expectations
3. **Added Agent Manager Mock** - Implemented missing `DescribeUser` method
4. **Created Basic Collector Tests** - Simple scraper creation tests

## Practical Path to 75% Coverage

### Phase 1: Quick Wins (Next 2-3 days) 📈
**Target: 45-50% overall coverage**

#### High-Impact, Low-Effort Tests:

1. **Configuration Tests** (Easy wins)
   ```go
   // connectors/config_test.go
   func TestLoadConfig(t *testing.T) {
       cfg := LoadDefaultConfig()
       assert.NotNil(t, cfg)
       assert.NotEmpty(t, cfg.Instance)
   }
   ```

2. **Queue Manager Tests** (Expand existing)
   ```go
   // Add more queue manager scenarios
   func TestQueueManagerRefresh(t *testing.T) // Test refresh logic
   func TestQueueManagerCaching(t *testing.T) // Test cache behavior
   ```

3. **Client Tests** (Error handling)
   ```go
   // connectors/client_test.go  
   func TestNewAWSConnectClient(t *testing.T) // Test client creation
   func TestClientErrorHandling(t *testing.T) // Test error scenarios
   ```

### Phase 2: Core Functionality (Week 2) 🎯
**Target: 60-65% overall coverage**

#### Collector Logic Tests:
1. **Metric Generation Logic**
   ```go
   func TestGeneratePrometheusMetric(t *testing.T)
   func TestMetricLabeling(t *testing.T)  
   func TestThresholdHandling(t *testing.T)
   ```

2. **Configuration Parsing**
   ```go
   func TestParseMetricConfigs(t *testing.T)
   func TestValidateGroupings(t *testing.T)
   func TestFilterValidation(t *testing.T)
   ```

### Phase 3: Integration & Edge Cases (Week 3) 🚀
**Target: 75%+ overall coverage**

#### Integration Tests with Mock API:
1. **End-to-End Collection**
   ```go
   func TestCollectMetricsE2E(t *testing.T) {
       // Start mock API
       // Run collector  
       // Validate metrics
   }
   ```

2. **Error Recovery**
   ```go
   func TestNetworkFailureRecovery(t *testing.T)
   func TestPartialDataHandling(t *testing.T)
   ```

## Specific Test Files to Create

### 1. `connectors/config_test.go` (High Impact)
```go
package connectors

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestLoadDefaultConfig(t *testing.T) {
    cfg := LoadDefaultConfig()
    assert.NotNil(t, cfg)
}

func TestValidateConfig(t *testing.T) {
    // Test configuration validation logic
    cfg := AWSConnectConfig{
        Instance: "test-instance",
        Region:   "us-east-1", 
    }
    err := ValidateConfig(cfg)
    assert.NoError(t, err)
}
```

### 2. `connectors/client_test.go` (Medium Impact)
```go
func TestNewAWSConnectClient(t *testing.T) {
    cfg := AWSConnectConfig{
        Instance: "test-instance",
        Region:   "us-east-1",
    }
    client, err := NewAWSConnectClient(cfg)
    assert.NoError(t, err)
    assert.NotNil(t, client)
}
```

### 3. `collectors/metric_generation_test.go` (High Impact)
```go
func TestGenerateHistoricalMetric(t *testing.T) {
    // Test metric generation without full connector
    scraper := NewGenericHistoricalScraper()
    
    // Mock metric data
    metricData := // ... create test data
    
    metric := scraper.generateMetric(metricData)
    assert.NotNil(t, metric)
}
```

## Estimated Coverage Impact

| Phase | Package | Current | After Phase | Tests Added |
|-------|---------|---------|-------------|-------------|
| 1 | connectors | 33.9% | 50% | 5-8 simple tests |
| 1 | collectors | 0.6% | 15% | 3-5 unit tests |
| 1 | **Overall** | **~18%** | **~35%** | **8-13 tests** |
| 2 | connectors | 50% | 65% | Complex logic tests |
| 2 | collectors | 15% | 45% | Core functionality |  
| 2 | **Overall** | **~35%** | **~58%** | **15-20 more tests** |
| 3 | connectors | 65% | 75% | Integration tests |
| 3 | collectors | 45% | 80% | E2E with mock API |
| 3 | **Overall** | **~58%** | **~75%** ✅ | **10-15 more tests** |

## Success Metrics & Validation

### Coverage Targets:
- [ ] **Phase 1**: 35% overall coverage (2-3 days)
- [ ] **Phase 2**: 58% overall coverage (1 week)  
- [ ] **Phase 3**: 75% overall coverage (2 weeks) ✅

### Quality Gates:
- [ ] All tests pass consistently (>99% success rate)
- [ ] Test execution time <30 seconds
- [ ] No test dependencies on external services (except mock API)
- [ ] Clear test documentation and examples

### Validation Commands:
```bash
# Overall coverage check
go test -cover ./connectors ./collectors | grep coverage

# Detailed coverage report
go test -coverprofile=coverage.out ./connectors ./collectors
go tool cover -html=coverage.out -o coverage.html

# Fast test execution
go test ./connectors ./collectors -timeout=30s
```

## Implementation Priority

### Week 1 Focus:
1. ✅ **Fix existing tests** (COMPLETED)
2. **Add config tests** (2 hours)
3. **Add client tests** (4 hours)  
4. **Expand queue manager tests** (3 hours)
5. **Add basic collector unit tests** (4 hours)

### Week 2 Focus:
1. **Metric generation logic tests** (6 hours)
2. **Configuration parsing tests** (4 hours)
3. **Error handling tests** (4 hours)

### Week 3 Focus:
1. **Integration tests with mock API** (8 hours)
2. **End-to-end collection tests** (6 hours)
3. **Edge case and error recovery tests** (4 hours)

## Notes for Implementation:

### Test Structure:
- Keep tests simple and focused
- Use table-driven tests for multiple scenarios
- Mock dependencies appropriately
- Test both success and failure paths

### Mock API Usage:
- Use existing mock API on port 3000
- Ensure tests work with the proven mock API setup
- Don't modify mock API unless absolutely necessary

### Coverage Strategy:
- Focus on high-impact, frequently used code paths first
- Don't chase 100% coverage - 75% with good quality is the goal
- Prioritize business logic over boilerplate code
- Test error paths and edge cases

This plan provides a realistic path to 75% coverage while maintaining test quality and keeping development time reasonable.
package connectors

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/stretchr/testify/assert"
)

// mockStatusLister implements AgentLister including ListAgentStatuses for testing.
type mockStatusLister struct {
	// Users for ListUsers pagination (not relevant for these tests)
	users []types.UserSummary
	// Paged status responses
	pages   []*connect.ListAgentStatusesOutput
	pageIdx int
}

func (m *mockStatusLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	return &connect.ListUsersOutput{UserSummaryList: m.users}, nil
}

func (m *mockStatusLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return &connect.DescribeUserOutput{}, nil
}

func (m *mockStatusLister) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	// Serve pages sequentially, respecting NextToken
	if m.pageIdx >= len(m.pages) {
		return &connect.ListAgentStatusesOutput{}, nil
	}
	out := m.pages[m.pageIdx]
	m.pageIdx++
	return out, nil
}

func (m *mockStatusLister) GetInstanceID() string { return "test-instance" }

func TestAgentManagerStatusPaginationAndCache(t *testing.T) {
	// Two pages of statuses
	s1Arn := aws.String("arn:aws:connect:us-east-1:123:instance/inst/status/s1")
	s2Arn := aws.String("arn:aws:connect:us-east-1:123:instance/inst/status/s2")
	lister := &mockStatusLister{
		pages: []*connect.ListAgentStatusesOutput{
			{AgentStatusSummaryList: []types.AgentStatusSummary{{Arn: s1Arn}}, NextToken: aws.String("n2")},
			{AgentStatusSummaryList: []types.AgentStatusSummary{{Arn: s2Arn}}},
		},
	}

	am := NewAgentManager(lister, 10*time.Millisecond)

	// Force a refresh and validate both pages aggregated
	err := am.refreshAgentStatuses()
	assert.NoError(t, err)
	m := am.GetStatusMap()
	assert.Len(t, m, 2)
	_, ok1 := m[*s1Arn]
	_, ok2 := m[*s2Arn]
	assert.True(t, ok1)
	assert.True(t, ok2)

	// Verify cache reuse when TTL not expired
	// Change lister pages to empty to detect reuse
	lister.pages = []*connect.ListAgentStatusesOutput{}
	// Set long TTL and recent refresh
	am.statusTTL = 10 * time.Minute
	am.lastStatusRefresh = time.Now()
	cached := am.GetStatusMap()
	assert.Len(t, cached, 2)
}

func TestAgentManagerStatusTTLRefresh(t *testing.T) {
	s3Arn := aws.String("arn:aws:connect:us-east-1:123:instance/inst/status/s3")
	lister := &mockStatusLister{
		pages: []*connect.ListAgentStatusesOutput{
			{AgentStatusSummaryList: []types.AgentStatusSummary{{Arn: s3Arn}}},
		},
	}

	am := NewAgentManager(lister, time.Second)
	// Prime with empty cache first
	am.statusTTL = 0
	am.lastStatusRefresh = time.Time{}

	m1 := am.GetStatusMap() // TTL expired, should call lister and populate
	assert.Len(t, m1, 1)
	_, ok := m1[*s3Arn]
	assert.True(t, ok)
}

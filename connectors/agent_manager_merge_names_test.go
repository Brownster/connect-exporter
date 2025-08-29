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

type singleUserLister struct{}

func (s *singleUserLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	return &connect.ListUsersOutput{UserSummaryList: []types.UserSummary{{Id: aws.String("a1"), Username: aws.String("u1")}}}, nil
}
func (s *singleUserLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return &connect.DescribeUserOutput{}, nil
}
func (s *singleUserLister) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	return &connect.ListAgentStatusesOutput{}, nil
}
func (s *singleUserLister) GetInstanceID() string { return "inst" }

func TestAgentManagerMergesCachedNamesOnRefresh(t *testing.T) {
	m := NewAgentManager(&singleUserLister{}, time.Minute)
	// seed names cache
	m.namesMutex.Lock()
	m.agentNames["a1"] = AgentNames{FirstName: "Jane", LastName: "Doe", UpdatedAt: time.Now()}
	m.namesMutex.Unlock()

	m.Refresh()
	mp := m.GetAgentMap()
	ag, ok := mp["a1"]
	assert.True(t, ok)
	assert.Equal(t, "Jane", ag.FirstName)
	assert.Equal(t, "Doe", ag.LastName)
}

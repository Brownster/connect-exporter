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

// nameLister mocks DescribeUser to return identity info
type nameLister struct{}

func (n *nameLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	return &connect.ListUsersOutput{}, nil
}
func (n *nameLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return &connect.DescribeUserOutput{User: &types.User{IdentityInfo: &types.UserIdentityInfo{FirstName: aws.String("Jane"), LastName: aws.String("Doe")}}}, nil
}
func (n *nameLister) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	return &connect.ListAgentStatusesOutput{}, nil
}
func (n *nameLister) GetInstanceID() string { return "inst" }

func TestAgentManagerFetchAllAgentNames(t *testing.T) {
	l := &nameLister{}
	m := NewAgentManager(l, time.Minute)
	// Seed with one agent
	m.mu.Lock()
	m.agents = []Agent{{ID: "a1"}}
	m.mu.Unlock()

	m.fetchAllAgentNames()

	mp := m.GetAgentMap()
	a, ok := mp["a1"]
	assert.True(t, ok)
	assert.Equal(t, "Jane", a.FirstName)
	assert.Equal(t, "Doe", a.LastName)
}

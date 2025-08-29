package connectors

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/stretchr/testify/assert"
)

type nilUserLister struct{}

func (n *nilUserLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	return &connect.ListUsersOutput{}, nil
}
func (n *nilUserLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return &connect.DescribeUserOutput{User: nil}, nil
}
func (n *nilUserLister) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	return &connect.ListAgentStatusesOutput{}, nil
}
func (n *nilUserLister) GetInstanceID() string { return "inst" }

func TestAgentManagerFetchAgentNameNilUser(t *testing.T) {
	m := NewAgentManager(&nilUserLister{}, time.Minute)
	out := m.fetchAgentName("a1")
	assert.Nil(t, out)
}

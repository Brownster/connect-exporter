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

type pagedAgentLister struct {
	pages []*connect.ListUsersOutput
	idx   int
}

func (p *pagedAgentLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	if p.idx >= len(p.pages) {
		return &connect.ListUsersOutput{}, nil
	}
	out := p.pages[p.idx]
	p.idx++
	return out, nil
}
func (p *pagedAgentLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return &connect.DescribeUserOutput{}, nil
}
func (p *pagedAgentLister) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	return &connect.ListAgentStatusesOutput{}, nil
}
func (p *pagedAgentLister) GetInstanceID() string { return "inst" }

func TestAgentManagerRefreshPagination(t *testing.T) {
	l := &pagedAgentLister{pages: []*connect.ListUsersOutput{
		{UserSummaryList: []types.UserSummary{{Id: aws.String("a1"), Username: aws.String("u1")}}, NextToken: aws.String("n2")},
		{UserSummaryList: []types.UserSummary{{Id: aws.String("a2"), Username: aws.String("u2")}}},
	}}
	m := NewAgentManager(l, time.Minute)
	m.Refresh()
	got := m.GetAgents()
	assert.Len(t, got, 2)
}

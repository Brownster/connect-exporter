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

// pagedRPQLister simulates paginated ListRoutingProfileQueues responses.
type pagedRPQLister struct {
	pages []*connect.ListRoutingProfileQueuesOutput
	idx   int
}

func (p *pagedRPQLister) ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error) {
	if p.idx >= len(p.pages) {
		return &connect.ListRoutingProfileQueuesOutput{}, nil
	}
	out := p.pages[p.idx]
	p.idx++
	return out, nil
}
func (p *pagedRPQLister) GetInstanceID() string { return "test-instance" }

func TestRoutingProfileQueueManagerPagination(t *testing.T) {
	l := &pagedRPQLister{pages: []*connect.ListRoutingProfileQueuesOutput{
		{RoutingProfileQueueConfigSummaryList: []types.RoutingProfileQueueConfigSummary{{QueueId: aws.String("q1")}}, NextToken: aws.String("n2")},
		{RoutingProfileQueueConfigSummaryList: []types.RoutingProfileQueueConfigSummary{{QueueId: aws.String("q2")}}},
	}}

	mgr := NewRoutingProfileQueueManager(l, time.Minute)
	qs, err := mgr.GetQueuesForRoutingProfile("rpA")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []Queue{{ID: "q1", Name: "", Type: "STANDARD"}, {ID: "q2", Name: "", Type: "STANDARD"}}, qs)
}

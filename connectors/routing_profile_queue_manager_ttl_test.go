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

type countingRPQLister struct{ calls int }

func (c *countingRPQLister) ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error) {
	c.calls++
	return &connect.ListRoutingProfileQueuesOutput{RoutingProfileQueueConfigSummaryList: []types.RoutingProfileQueueConfigSummary{{QueueId: aws.String("q1")}}}, nil
}
func (c *countingRPQLister) GetInstanceID() string { return "inst" }

func TestRoutingProfileQueueManagerTTLRefresh(t *testing.T) {
	l := &countingRPQLister{}
	mgr := NewRoutingProfileQueueManager(l, 10*time.Millisecond)
	// first call populates cache
	_, err := mgr.GetQueuesForRoutingProfile("rp1")
	assert.NoError(t, err)
	assert.Equal(t, 1, l.calls)
	// immediate second call uses cache
	_, _ = mgr.GetQueuesForRoutingProfile("rp1")
	assert.Equal(t, 1, l.calls)
	// after TTL expires, it should refetch
	time.Sleep(20 * time.Millisecond)
	_, _ = mgr.GetQueuesForRoutingProfile("rp1")
	assert.Equal(t, 2, l.calls)
}

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

type mockRPQLister struct {
	output *connect.ListRoutingProfileQueuesOutput
	err    error
	calls  int
}

func (m *mockRPQLister) ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error) {
	m.calls++
	return m.output, m.err
}

func (m *mockRPQLister) GetInstanceID() string { return "test-instance" }

func TestRoutingProfileQueueManagerCaching(t *testing.T) {
	lister := &mockRPQLister{output: &connect.ListRoutingProfileQueuesOutput{
		RoutingProfileQueueConfigSummaryList: []types.RoutingProfileQueueConfigSummary{
			{QueueId: aws.String("q1")},
			{QueueId: aws.String("q2")},
		},
	}}
	mgr := NewRoutingProfileQueueManager(lister, 50*time.Millisecond)
	qs, err := mgr.GetQueuesForRoutingProfile("rp1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"q1", "q2"}, qs)
	assert.Equal(t, 1, lister.calls)

	qs, err = mgr.GetQueuesForRoutingProfile("rp1")
	assert.NoError(t, err)
	assert.Equal(t, 1, lister.calls)

	time.Sleep(60 * time.Millisecond)
	_, _ = mgr.GetQueuesForRoutingProfile("rp1")
	assert.Equal(t, 2, lister.calls)
}

func TestRoutingProfileQueueManagerError(t *testing.T) {
	lister := &mockRPQLister{err: assert.AnError}
	mgr := NewRoutingProfileQueueManager(lister, time.Minute)
	qs, err := mgr.GetQueuesForRoutingProfile("rp1")
	assert.Error(t, err)
	assert.Nil(t, qs)
}

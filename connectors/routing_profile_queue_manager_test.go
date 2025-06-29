// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file contains the unit tests for the RoutingProfileQueueManager. The tests
// use a mock implementation of the RoutingProfileQueueLister interface to isolate
// the manager from the AWS Connect API. This allows for fast, reliable, and
// focused testing of the manager's caching and error handling logic.
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

// mockRPQLister is a mock implementation of the RoutingProfileQueueLister
// interface used for testing the RoutingProfileQueueManager.
type mockRPQLister struct {
	output *connect.ListRoutingProfileQueuesOutput
	err    error
	calls  int
}

// ListRoutingProfileQueues is the mock implementation of the
// ListRoutingProfileQueues method. It returns the mock output or an error,
// depending on how it's configured.
func (m *mockRPQLister) ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error) {
	m.calls++
	return m.output, m.err
}

// GetInstanceID is the mock implementation of the GetInstanceID method.
func (m *mockRPQLister) GetInstanceID() string { return "test-instance" }

// TestRoutingProfileQueueManagerCaching tests the caching logic of the
// RoutingProfileQueueManager. It verifies that the manager caches the results
// of the ListRoutingProfileQueues API call and that the cache expires after the
// configured TTL.
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

// TestRoutingProfileQueueManagerError tests that the RoutingProfileQueueManager
// handles errors from the RoutingProfileQueueLister gracefully.
func TestRoutingProfileQueueManagerError(t *testing.T) {
	lister := &mockRPQLister{err: assert.AnError}
	mgr := NewRoutingProfileQueueManager(lister, time.Minute)
	qs, err := mgr.GetQueuesForRoutingProfile("rp1")
	assert.Error(t, err)
	assert.Nil(t, qs)
}

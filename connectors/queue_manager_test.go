// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file contains the unit tests for the QueueManager. The tests use a mock
// implementation of the QueueLister interface to isolate the QueueManager from
// the AWS Connect API. This allows for fast, reliable, and focused testing of
// the QueueManager's logic.
package connectors

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// fakeLister is a mock implementation of the QueueLister interface used for
// testing the QueueManager.
type fakeLister struct {
	queues []Queue
	calls  int
}

// ListQueues is the mock implementation of the ListQueues method. It returns the
// mock queues and increments the call counter.
func (f *fakeLister) ListQueues(ctx context.Context) ([]Queue, error) {
	f.calls++
	return f.queues, nil
}

// TestQueueManagerRefreshAndFilter tests that the QueueManager correctly
// refreshes the queue list and applies the include filter.
func TestQueueManagerRefreshAndFilter(t *testing.T) {
	lister := &fakeLister{queues: []Queue{{ID: "1", Name: "Sales", Type: "STANDARD"}, {ID: "2", Name: "Support", Type: "STANDARD"}}}
	qm, err := NewQueueManager(lister, time.Millisecond, "^Sales$")
	assert.NoError(t, err)
	qm.Refresh()
	qs := qm.GetQueues()
	if assert.Len(t, qs, 1) {
		assert.Equal(t, "Sales", qs[0].Name)
	}
	assert.Equal(t, 1, lister.calls)
}

// TestQueueManagerStartStops tests that the QueueManager can be started and
// stopped without any issues.
func TestQueueManagerStartStops(t *testing.T) {
	lister := &fakeLister{queues: []Queue{{ID: "1", Name: "Q1", Type: "STANDARD"}}}
	qm, err := NewQueueManager(lister, 10*time.Millisecond, "")
	assert.NoError(t, err)
	qm.Start()
	time.Sleep(25 * time.Millisecond)
	qm.Stop()
	// at least two calls to ListQueues should have occurred (initial + ticker)
	assert.GreaterOrEqual(t, lister.calls, 2)
}

// TestQueueManagerDiscoverQueues tests that the QueueManager correctly
// discovers and caches the list of queues.
func TestQueueManagerDiscoverQueues(t *testing.T) {
	expected := []Queue{
		{ID: "q-1", Name: "Sales_Queue_VOICE", Type: "STANDARD"},
		{ID: "q-2", Name: "Support_Queue_CHAT", Type: "STANDARD"},
		{ID: "q-3", Name: "Test_Queue_TASK", Type: "STANDARD"},
	}
	lister := &fakeLister{queues: expected}
	qm, err := NewQueueManager(lister, time.Millisecond, "")
	assert.NoError(t, err)
	qm.Refresh()
	qs := qm.GetQueues()
	assert.Equal(t, expected, qs)
}

// errorLister is a mock implementation of the QueueLister interface that always
// returns an error.
type errorLister struct{ err error }

// ListQueues is the mock implementation of the ListQueues method. It always
// returns an error.
func (e *errorLister) ListQueues(ctx context.Context) ([]Queue, error) { return nil, e.err }

// TestQueueManagerDiscoverQueuesError tests that the QueueManager handles errors
// from the QueueLister gracefully.
func TestQueueManagerDiscoverQueuesError(t *testing.T) {
	lister := &errorLister{err: fmt.Errorf("boom")}
	qm, err := NewQueueManager(lister, time.Millisecond, "")
	assert.NoError(t, err)

	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	qm.Refresh()
	assert.Contains(t, buf.String(), "failed to refresh queue list")
	assert.Empty(t, qm.GetQueues())
}

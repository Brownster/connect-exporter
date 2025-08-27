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

type fakeLister struct {
	queues []Queue
	calls  int
}

func (f *fakeLister) ListQueues(ctx context.Context) ([]Queue, error) {
	f.calls++
	return f.queues, nil
}

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

type errorLister struct{ err error }

func (e *errorLister) ListQueues(ctx context.Context) ([]Queue, error) { return nil, e.err }

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

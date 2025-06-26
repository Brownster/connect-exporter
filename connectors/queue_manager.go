package connectors

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// QueueLister defines the interface for listing queues
// implemented by AWSConnectConnector
// and useful for testing QueueManager independently.
type QueueLister interface {
	ListQueues(ctx context.Context) ([]Queue, error)
}

// QueueManager periodically refreshes the list of queues
// from AWS Connect and provides cached access for collectors.
type QueueManager struct {
	lister          QueueLister
	refreshInterval time.Duration
	includeRegex    *regexp.Regexp

	mu     sync.RWMutex
	queues []Queue
	stopCh chan struct{}
}

// NewQueueManager creates a new QueueManager. If pattern is non-empty,
// only queues whose name matches the regex will be retained.
func NewQueueManager(lister QueueLister, interval time.Duration, pattern string) (*QueueManager, error) {
	var re *regexp.Regexp
	var err error
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &QueueManager{
		lister:          lister,
		refreshInterval: interval,
		includeRegex:    re,
		stopCh:          make(chan struct{}),
	}, nil
}

// Start begins the periodic refresh loop.
func (m *QueueManager) Start() {
	m.Refresh()
	ticker := time.NewTicker(m.refreshInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.Refresh()
			case <-m.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop terminates the refresh loop.
func (m *QueueManager) Stop() {
	close(m.stopCh)
}

// Refresh immediately updates the cached queue list.
func (m *QueueManager) Refresh() {
	queues, err := m.lister.ListQueues(context.Background())
	if err != nil {
		logrus.WithError(err).Warn("failed to refresh queue list")
		return
	}
	if m.includeRegex != nil {
		filtered := make([]Queue, 0, len(queues))
		for _, q := range queues {
			if m.includeRegex.MatchString(q.Name) {
				filtered = append(filtered, q)
			}
		}
		queues = filtered
	}
	m.mu.Lock()
	m.queues = queues
	m.mu.Unlock()
}

// GetQueues returns the current cached queue list.
func (m *QueueManager) GetQueues() []Queue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Queue, len(m.queues))
	copy(out, m.queues)
	return out
}

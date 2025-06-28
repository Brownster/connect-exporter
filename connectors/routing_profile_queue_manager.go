package connectors

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/sirupsen/logrus"
)

// RoutingProfileQueueLister defines the interface for listing queues for a routing profile
// implemented by AWSConnectConnector
// and useful for testing RoutingProfileQueueManager independently.
type RoutingProfileQueueLister interface {
	ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error)
	GetInstanceID() string
}

type rpqEntry struct {
	queues  []string
	expires time.Time
}

// RoutingProfileQueueManager caches queue lists per routing profile with TTL.
type RoutingProfileQueueManager struct {
	lister RoutingProfileQueueLister
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]*rpqEntry
}

// NewRoutingProfileQueueManager creates a new manager with the given TTL.
func NewRoutingProfileQueueManager(lister RoutingProfileQueueLister, ttl time.Duration) *RoutingProfileQueueManager {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &RoutingProfileQueueManager{
		lister: lister,
		ttl:    ttl,
		cache:  make(map[string]*rpqEntry),
	}
}

// GetQueuesForRoutingProfile returns queue IDs for the given routing profile.
// Results are cached for the configured TTL.
func (m *RoutingProfileQueueManager) GetQueuesForRoutingProfile(rpID string) ([]string, error) {
	m.mu.RLock()
	entry, ok := m.cache[rpID]
	if ok && time.Now().Before(entry.expires) {
		out := make([]string, len(entry.queues))
		copy(out, entry.queues)
		m.mu.RUnlock()
		return out, nil
	}
	m.mu.RUnlock()

	queues, err := m.fetchQueues(rpID)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.cache[rpID] = &rpqEntry{queues: queues, expires: time.Now().Add(m.ttl)}
	m.mu.Unlock()
	return queues, nil
}

func (m *RoutingProfileQueueManager) fetchQueues(rpID string) ([]string, error) {
	instanceID := m.lister.GetInstanceID()
	var err error
	var queues []string
	for attempt := 0; attempt < 3; attempt++ {
		queues = queues[:0]
		var nextToken *string
		for {
			var out *connect.ListRoutingProfileQueuesOutput
			out, err = m.lister.ListRoutingProfileQueues(context.Background(), &connect.ListRoutingProfileQueuesInput{
				InstanceId:       &instanceID,
				RoutingProfileId: aws.String(rpID),
				NextToken:        nextToken,
			})
			if err != nil {
				break
			}
			for _, q := range out.RoutingProfileQueueConfigSummaryList {
				if q.QueueId != nil {
					queues = append(queues, aws.ToString(q.QueueId))
				}
			}
			if out.NextToken == nil {
				break
			}
			nextToken = out.NextToken
		}
		if err == nil {
			return queues, nil
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	logrus.WithError(err).Warn("failed to list routing profile queues")
	return nil, err
}

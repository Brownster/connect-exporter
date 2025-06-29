// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file defines the AgentManager, a component responsible for periodically
// refreshing the list of agents from AWS Connect and providing a cached, thread-safe
// way for collectors to access this data. This avoids redundant API calls and
// improves the overall performance of the exporter.
package connectors

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/sirupsen/logrus"
)

// Agent represents essential metadata about an AWS Connect agent.
type Agent struct {
	ID       string
	Username string
	Arn      string
}

// AgentLister defines the interface for listing agents. It is implemented by
// AWSConnectConnector and allows the AgentManager to be tested independently.
type AgentLister interface {
	ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error)
	GetInstanceID() string
}

// AgentManager periodically refreshes the list of agents/users from AWS Connect
// and provides cached access for collectors.
type AgentManager struct {
	lister          AgentLister
	refreshInterval time.Duration

	mu     sync.RWMutex
	agents []Agent
	stopCh chan struct{}
}

// NewAgentManager creates and initializes a new AgentManager.
func NewAgentManager(lister AgentLister, interval time.Duration) *AgentManager {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &AgentManager{
		lister:          lister,
		refreshInterval: interval,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the periodic refresh loop in a separate goroutine.
func (m *AgentManager) Start() {
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

// Stop terminates the background refresh loop.
func (m *AgentManager) Stop() {
	close(m.stopCh)
}

// Refresh immediately fetches the latest agent list from the AWS Connect API.
func (m *AgentManager) Refresh() {
	instanceID := m.lister.GetInstanceID()
	usersInput := &connect.ListUsersInput{
		InstanceId: &instanceID,
		MaxResults: aws.Int32(1000),
	}

	users, err := m.lister.ListUsers(context.Background(), usersInput)
	if err != nil {
		logrus.WithError(err).Warn("failed to refresh agent list")
		return
	}

	agents := make([]Agent, 0, len(users.UserSummaryList))
	for _, u := range users.UserSummaryList {
		if u.Id != nil {
			agent := Agent{
				ID:  aws.ToString(u.Id),
				Arn: aws.ToString(u.Arn),
			}
			if u.Username != nil {
				agent.Username = aws.ToString(u.Username)
			}
			agents = append(agents, agent)
		}
	}

	m.mu.Lock()
	m.agents = agents
	m.mu.Unlock()

	logrus.WithField("agent_count", len(agents)).Debug("Refreshed agent list")
}

// GetAgents returns a thread-safe copy of the current cached agent list.
func (m *AgentManager) GetAgents() []Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Agent, len(m.agents))
	copy(out, m.agents)
	return out
}

// GetAgentByID returns a specific agent by their ID from the cache.
func (m *AgentManager) GetAgentByID(id string) (Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, agent := range m.agents {
		if agent.ID == id {
			return agent, true
		}
	}
	return Agent{}, false
}

// GetAgentMap returns a map of agent IDs to agent metadata for efficient lookups.
func (m *AgentManager) GetAgentMap() map[string]Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agentMap := make(map[string]Agent, len(m.agents))
	for _, agent := range m.agents {
		agentMap[agent.ID] = agent
	}
	return agentMap
}

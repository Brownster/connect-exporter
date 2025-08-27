package connectors

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/sirupsen/logrus"
)

// Agent represents comprehensive agent metadata
type Agent struct {
	ID               string
	Username         string
	FirstName        string
	LastName         string
	Email            string
	Phone            string
	RoutingProfileID string
	HierarchyGroupID string
	SecurityProfileIDs []string
	Tags             map[string]string
}

// AgentLister defines the interface for listing agents
// implemented by AWSConnectConnector
type AgentLister interface {
	ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error)
	DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error)
	GetInstanceID() string
}

// AgentManager periodically refreshes the list of agents
// from AWS Connect and provides cached access for collectors.
type AgentManager struct {
	lister          AgentLister
	refreshInterval time.Duration

	mu     sync.RWMutex
	agents []Agent
	stopCh chan struct{}
}

// NewAgentManager creates a new AgentManager.
func NewAgentManager(lister AgentLister, interval time.Duration) (*AgentManager, error) {
	if interval <= 0 {
		interval = 12 * time.Hour
	}
	return &AgentManager{
		lister:          lister,
		refreshInterval: interval,
		stopCh:          make(chan struct{}),
	}, nil
}

// Start begins the periodic refresh loop.
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

// Stop terminates the refresh loop.
func (m *AgentManager) Stop() {
	close(m.stopCh)
}

// Refresh immediately updates the cached agent list.
func (m *AgentManager) Refresh() {
	ctx := context.Background()
	instanceID := m.lister.GetInstanceID()
	
	// Step 1: Get list of all users
	var allUserSummaries []struct {
		ID       string
		Username string
	}
	var nextToken *string

	for {
		input := &connect.ListUsersInput{
			InstanceId: &instanceID,
			MaxResults: aws.Int32(100),
			NextToken:  nextToken,
		}

		output, err := m.lister.ListUsers(ctx, input)
		if err != nil {
			logrus.WithError(err).Warn("failed to refresh agent list")
			return
		}

		for _, user := range output.UserSummaryList {
			allUserSummaries = append(allUserSummaries, struct {
				ID       string
				Username string
			}{
				ID:       *user.Id,
				Username: getStringValue(user.Username),
			})
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}

	// Step 2: Get detailed info for each user using DescribeUser
	var agents []Agent
	for _, userSummary := range allUserSummaries {
		input := &connect.DescribeUserInput{
			InstanceId: &instanceID,
			UserId:     &userSummary.ID,
		}

		output, err := m.lister.DescribeUser(ctx, input)
		if err != nil {
			logrus.WithError(err).WithField("user_id", userSummary.ID).Debug("failed to describe user, using partial data")
			// Use partial data if DescribeUser fails
			agents = append(agents, Agent{
				ID:       userSummary.ID,
				Username: userSummary.Username,
			})
			continue
		}

		if output.User == nil {
			continue
		}

		user := output.User
		agent := Agent{
			ID:               *user.Id,
			Username:         getStringValue(user.Username),
			RoutingProfileID: getStringValue(user.RoutingProfileId),
			HierarchyGroupID: getStringValue(user.HierarchyGroupId),
			Tags:             make(map[string]string),
		}

		// Extract first/last name from IdentityInfo
		if user.IdentityInfo != nil {
			agent.FirstName = getStringValue(user.IdentityInfo.FirstName)
			agent.LastName = getStringValue(user.IdentityInfo.LastName)
			agent.Email = getStringValue(user.IdentityInfo.Email)
			agent.Phone = getStringValue(user.IdentityInfo.Mobile)
		}

		// Convert security profile IDs
		if user.SecurityProfileIds != nil {
			for _, spId := range user.SecurityProfileIds {
				agent.SecurityProfileIDs = append(agent.SecurityProfileIDs, spId)
			}
		}

		// Convert tags
		if user.Tags != nil {
			for key, value := range user.Tags {
				agent.Tags[key] = value
			}
		}

		agents = append(agents, agent)
	}

	m.mu.Lock()
	m.agents = agents
	m.mu.Unlock()

	logrus.WithField("agent_count", len(agents)).Debug("refreshed agent cache")
}

// GetAgents returns the current cached agent list.
func (m *AgentManager) GetAgents() []Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Agent, len(m.agents))
	copy(out, m.agents)
	return out
}

// GetAgentByID returns a specific agent by ID from cache
func (m *AgentManager) GetAgentByID(id string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.agents {
		if m.agents[i].ID == id {
			return &m.agents[i]
		}
	}
	return nil
}

// GetAgentByUsername returns a specific agent by username from cache
func (m *AgentManager) GetAgentByUsername(username string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.agents {
		if m.agents[i].Username == username {
			return &m.agents[i]
		}
	}
	return nil
}
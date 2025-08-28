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
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// Agent represents essential metadata about an AWS Connect agent.
type Agent struct {
	ID        string
	Username  string
	Arn       string
	FirstName string
	LastName  string
}

// AgentNames represents cached first and last names for an agent.
type AgentNames struct {
	FirstName string
	LastName  string
	UpdatedAt time.Time
}

// AgentLister defines the interface for listing agents. It is implemented by
// AWSConnectConnector and allows the AgentManager to be tested independently.
type AgentLister interface {
	ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error)
	DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error)
	GetInstanceID() string
}

// AgentManager periodically refreshes the list of agents/users from AWS Connect
// and provides cached access for collectors.
type AgentManager struct {
	lister          AgentLister
	refreshInterval time.Duration

	// Agent list cache
	mu     sync.RWMutex
	agents []Agent
	stopCh chan struct{}

	// Agent names cache and configuration
	agentNames           map[string]AgentNames
	namesMutex           sync.RWMutex
	rateLimiter          *rate.Limiter
	nameRefreshInterval  time.Duration
	nameRefreshTicker    *time.Ticker
	lastNameRefresh      time.Time
	namesEnabled         bool
}

// NewAgentManager creates and initializes a new AgentManager.
func NewAgentManager(lister AgentLister, interval time.Duration) *AgentManager {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &AgentManager{
		lister:              lister,
		refreshInterval:     interval,
		stopCh:              make(chan struct{}),
		agentNames:          make(map[string]AgentNames),
		rateLimiter:         rate.NewLimiter(rate.Limit(2), 5), // 2 calls/sec with burst of 5
		nameRefreshInterval: 6 * time.Hour,                     // Default 6 hours
		namesEnabled:        true,                              // Enable by default
	}
}

// Start begins the periodic refresh loop in a separate goroutine.
func (m *AgentManager) Start() {
	m.Refresh()
	
	// Start periodic agent list refresh
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
	
	// Start initial agent names fetch and periodic refresh
	if m.namesEnabled {
		go m.fetchAllAgentNames()
		
		m.nameRefreshTicker = time.NewTicker(m.nameRefreshInterval)
		go func() {
			for {
				select {
				case <-m.nameRefreshTicker.C:
					go m.fetchAllAgentNames()
				case <-m.stopCh:
					if m.nameRefreshTicker != nil {
						m.nameRefreshTicker.Stop()
					}
					return
				}
			}
		}()
	}
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
			
			// Add cached names if available
			m.namesMutex.RLock()
			if names, exists := m.agentNames[agent.ID]; exists {
				agent.FirstName = names.FirstName
				agent.LastName = names.LastName
			}
			m.namesMutex.RUnlock()
			
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
	m.namesMutex.RLock()
	defer m.mu.RUnlock()
	defer m.namesMutex.RUnlock()
	
	agentMap := make(map[string]Agent, len(m.agents))
	for _, agent := range m.agents {
		// Merge agent names from the names cache if available
		if names, exists := m.agentNames[agent.ID]; exists {
			agent.FirstName = names.FirstName
			agent.LastName = names.LastName
		}
		agentMap[agent.ID] = agent
	}
	return agentMap
}

// fetchAllAgentNames fetches first and last names for all agents with smart throttling.
func (m *AgentManager) fetchAllAgentNames() {
	if !m.namesEnabled {
		return
	}
	
	logrus.Info("Starting agent names collection")
	startTime := time.Now()
	
	// Get current agent IDs
	m.mu.RLock()
	agentIDs := make([]string, 0, len(m.agents))
	for _, agent := range m.agents {
		agentIDs = append(agentIDs, agent.ID)
	}
	m.mu.RUnlock()
	
	if len(agentIDs) == 0 {
		logrus.Debug("No agents to fetch names for")
		return
	}
	
	successCount := 0
	errorCount := 0
	throttleEvents := 0
	
	for _, agentID := range agentIDs {
		// Check if we need to rate limit
		if !m.rateLimiter.Allow() {
			throttleEvents++
			logrus.WithField("agent_id", agentID).Debug("Rate limiting agent name fetch")
			
			// Wait for the rate limiter
			reservation := m.rateLimiter.Reserve()
			if !reservation.OK() {
				logrus.WithField("agent_id", agentID).Warn("Rate limiter reservation failed")
				continue
			}
			time.Sleep(reservation.Delay())
		}
		
		if names := m.fetchAgentName(agentID); names != nil {
			m.namesMutex.Lock()
			m.agentNames[agentID] = *names
			m.namesMutex.Unlock()
			successCount++
		} else {
			errorCount++
		}
	}
	
	m.lastNameRefresh = time.Now()
	duration := time.Since(startTime)
	
	logrus.WithFields(logrus.Fields{
		"total_agents":     len(agentIDs),
		"success_count":    successCount,
		"error_count":      errorCount,
		"throttle_events":  throttleEvents,
		"duration_seconds": duration.Seconds(),
		"rate_per_second":  float64(len(agentIDs)) / duration.Seconds(),
	}).Info("Agent names collection completed")
}

// fetchAgentName fetches the first and last name for a specific agent.
func (m *AgentManager) fetchAgentName(agentID string) *AgentNames {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	output, err := m.lister.DescribeUser(ctx, &connect.DescribeUserInput{
		InstanceId: aws.String(m.lister.GetInstanceID()),
		UserId:     aws.String(agentID),
	})
	
	if err != nil {
		if isRateLimitError(err) {
			logrus.WithFields(logrus.Fields{
				"agent_id": agentID,
				"error":    err.Error(),
			}).Debug("Rate limited on DescribeUser")
		} else {
			logrus.WithFields(logrus.Fields{
				"agent_id": agentID,
				"error":    err.Error(),
			}).Warn("Failed to describe user")
		}
		return nil
	}
	
	if output.User == nil {
		logrus.WithField("agent_id", agentID).Warn("DescribeUser returned nil user")
		return nil
	}
	
	var firstName, lastName string
	if output.User.IdentityInfo != nil {
		firstName = aws.ToString(output.User.IdentityInfo.FirstName)
		lastName = aws.ToString(output.User.IdentityInfo.LastName)
		logrus.WithFields(logrus.Fields{
			"agent_id": agentID,
			"identity_info_present": true,
			"raw_first_name": output.User.IdentityInfo.FirstName,
			"raw_last_name": output.User.IdentityInfo.LastName,
			"parsed_first_name": firstName,
			"parsed_last_name": lastName,
		}).Debug("DescribeUser IdentityInfo details")
	} else {
		logrus.WithFields(logrus.Fields{
			"agent_id": agentID,
			"identity_info_present": false,
			"user_present": output.User != nil,
		}).Debug("DescribeUser IdentityInfo missing")
	}
	
	logrus.WithFields(logrus.Fields{
		"agent_id":   agentID,
		"first_name": firstName,
		"last_name":  lastName,
	}).Debug("Successfully fetched agent names")
	
	return &AgentNames{
		FirstName: firstName,
		LastName:  lastName,
		UpdatedAt: time.Now(),
	}
}

// isRateLimitError checks if an error is due to API rate limiting.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	return errors.New(errStr).Error() == "Throttling" ||
		errors.New(errStr).Error() == "TooManyRequestsException" ||
		errors.New(errStr).Error() == "ThrottlingException"
}

// SetNamesEnabled allows enabling or disabling agent name collection.
func (m *AgentManager) SetNamesEnabled(enabled bool) {
	m.namesEnabled = enabled
	if !enabled {
		logrus.Info("Agent name collection disabled")
	} else {
		logrus.Info("Agent name collection enabled")
	}
}

// GetNamesCacheStats returns statistics about the agent names cache.
func (m *AgentManager) GetNamesCacheStats() map[string]interface{} {
	m.namesMutex.RLock()
	defer m.namesMutex.RUnlock()
	
	return map[string]interface{}{
		"cached_names_count": len(m.agentNames),
		"last_refresh":       m.lastNameRefresh,
		"names_enabled":      m.namesEnabled,
		"cache_age_hours":    time.Since(m.lastNameRefresh).Hours(),
	}
}

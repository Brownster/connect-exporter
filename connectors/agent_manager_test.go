// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file contains the unit tests for the AgentManager. The tests use a mock
// implementation of the AgentLister interface to isolate the AgentManager from
// the AWS Connect API. This allows for fast, reliable, and focused testing of
// the AgentManager's logic.
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

// mockAgentLister is a mock implementation of the AgentLister interface used
// for testing the AgentManager.
type mockAgentLister struct {
	agents []types.UserSummary
	err    error
}

// ListUsers is the mock implementation of the ListUsers method. It returns the
// mock agents or an error, depending on how it's configured.
func (m *mockAgentLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &connect.ListUsersOutput{
		UserSummaryList: m.agents,
	}, nil
}

// GetInstanceID is the mock implementation of the GetInstanceID method.
func (m *mockAgentLister) GetInstanceID() string {
	return "test-instance"
}

// TestAgentManager tests the basic functionality of the AgentManager, including
// refreshing the agent list and retrieving agents by ID and as a map.
func TestAgentManager(t *testing.T) {
	mockAgents := []types.UserSummary{
		{
			Id:       aws.String("agent-1"),
			Username: aws.String("john.doe"),
			Arn:      aws.String("arn:aws:connect:us-east-1:123456789012:instance/test/agent/agent-1"),
		},
		{
			Id:       aws.String("agent-2"),
			Username: aws.String("jane.smith"),
			Arn:      aws.String("arn:aws:connect:us-east-1:123456789012:instance/test/agent/agent-2"),
		},
	}

	lister := &mockAgentLister{agents: mockAgents}
	manager := NewAgentManager(lister, 10*time.Millisecond)

	// Test initial refresh
	manager.Refresh()
	agents := manager.GetAgents()

	assert.Len(t, agents, 2)
	assert.Equal(t, "agent-1", agents[0].ID)
	assert.Equal(t, "john.doe", agents[0].Username)
	assert.Equal(t, "agent-2", agents[1].ID)
	assert.Equal(t, "jane.smith", agents[1].Username)

	// Test GetAgentByID
	agent, found := manager.GetAgentByID("agent-1")
	assert.True(t, found)
	assert.Equal(t, "john.doe", agent.Username)

	_, found = manager.GetAgentByID("nonexistent")
	assert.False(t, found)

	// Test GetAgentMap
	agentMap := manager.GetAgentMap()
	assert.Len(t, agentMap, 2)
	assert.Equal(t, "john.doe", agentMap["agent-1"].Username)
	assert.Equal(t, "jane.smith", agentMap["agent-2"].Username)
}

// TestAgentManagerWithError tests that the AgentManager handles errors from the
// AgentLister gracefully.
func TestAgentManagerWithError(t *testing.T) {
	lister := &mockAgentLister{err: assert.AnError}
	manager := NewAgentManager(lister, 10*time.Millisecond)

	// Test refresh with error
	manager.Refresh()
	agents := manager.GetAgents()

	// Should return empty list when there's an error
	assert.Len(t, agents, 0)
}

// TestAgentManagerStartStop tests that the AgentManager can be started and
// stopped without any issues.
func TestAgentManagerStartStop(t *testing.T) {
	lister := &mockAgentLister{agents: []types.UserSummary{}}
	manager := NewAgentManager(lister, 10*time.Millisecond)

	// Test start and stop
	manager.Start()
	time.Sleep(20 * time.Millisecond) // Let it run briefly
	manager.Stop()

	// Should not panic or hang
	assert.True(t, true)
}

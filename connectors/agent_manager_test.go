package connectors

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakeAgentLister struct {
	users []Agent
	calls int
	describeCallCount int
}

func (f *fakeAgentLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	f.calls++
	users := make([]types.UserSummary, len(f.users))
	for i, agent := range f.users {
		users[i] = types.UserSummary{
			Id:       &agent.ID,
			Username: &agent.Username,
		}
	}
	return &connect.ListUsersOutput{UserSummaryList: users}, nil
}

func (f *fakeAgentLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	f.describeCallCount++
	userID := *input.UserId
	
	// Find the user
	for _, agent := range f.users {
		if agent.ID == userID {
			user := types.User{
				Id:               &agent.ID,
				Username:         &agent.Username,
				RoutingProfileId: &agent.RoutingProfileID,
				HierarchyGroupId: &agent.HierarchyGroupID,
				IdentityInfo: &types.UserIdentityInfo{
					FirstName: &agent.FirstName,
					LastName:  &agent.LastName,
					Email:     &agent.Email,
					Mobile:    &agent.Phone,
				},
				SecurityProfileIds: agent.SecurityProfileIDs,
				Tags:              agent.Tags,
			}
			return &connect.DescribeUserOutput{User: &user}, nil
		}
	}
	return nil, fmt.Errorf("user not found: %s", userID)
}

func (f *fakeAgentLister) GetInstanceID() string {
	return "test-instance-id"
}

func TestAgentManagerRefresh(t *testing.T) {
	expectedAgents := []Agent{
		{ID: "agent-1", Username: "alice.smith", FirstName: "Alice", LastName: "Smith", RoutingProfileID: "rp-sales"},
		{ID: "agent-2", Username: "bob.jones", FirstName: "Bob", LastName: "Jones", RoutingProfileID: "rp-support"},
	}
	
	lister := &fakeAgentLister{users: expectedAgents}
	am, err := NewAgentManager(lister, time.Millisecond)
	assert.NoError(t, err)
	
	am.Refresh()
	
	agents := am.GetAgents()
	assert.Len(t, agents, 2)
	assert.Equal(t, 1, lister.calls)
	assert.Equal(t, 2, lister.describeCallCount)
	
	// Check agent details
	alice := am.GetAgentByUsername("alice.smith")
	assert.NotNil(t, alice)
	assert.Equal(t, "Alice", alice.FirstName)
	assert.Equal(t, "Smith", alice.LastName)
	assert.Equal(t, "rp-sales", alice.RoutingProfileID)
	
	bob := am.GetAgentByID("agent-2")
	assert.NotNil(t, bob)
	assert.Equal(t, "Bob", bob.FirstName)
	assert.Equal(t, "bob.jones", bob.Username)
}

func TestAgentManagerStartStop(t *testing.T) {
	agents := []Agent{{ID: "agent-1", Username: "test.user", FirstName: "Test", LastName: "User"}}
	lister := &fakeAgentLister{users: agents}
	am, err := NewAgentManager(lister, 10*time.Millisecond)
	assert.NoError(t, err)
	
	am.Start()
	time.Sleep(25 * time.Millisecond)
	am.Stop()
	
	// At least two calls should have occurred (initial + ticker)
	assert.GreaterOrEqual(t, lister.calls, 2)
	assert.GreaterOrEqual(t, lister.describeCallCount, 2)
}

func TestAgentManagerGetAgentByID(t *testing.T) {
	agents := []Agent{
		{ID: "agent-1", Username: "alice.smith", FirstName: "Alice", LastName: "Smith"},
		{ID: "agent-2", Username: "bob.jones", FirstName: "Bob", LastName: "Jones"},
	}
	
	lister := &fakeAgentLister{users: agents}
	am, err := NewAgentManager(lister, time.Millisecond)
	assert.NoError(t, err)
	am.Refresh()
	
	// Test existing agent
	agent := am.GetAgentByID("agent-1")
	assert.NotNil(t, agent)
	assert.Equal(t, "Alice", agent.FirstName)
	assert.Equal(t, "alice.smith", agent.Username)
	
	// Test non-existing agent
	agent = am.GetAgentByID("non-existent")
	assert.Nil(t, agent)
}

func TestAgentManagerGetAgentByUsername(t *testing.T) {
	agents := []Agent{
		{ID: "agent-1", Username: "alice.smith", FirstName: "Alice", LastName: "Smith"},
		{ID: "agent-2", Username: "bob.jones", FirstName: "Bob", LastName: "Jones"},
	}
	
	lister := &fakeAgentLister{users: agents}
	am, err := NewAgentManager(lister, time.Millisecond)
	assert.NoError(t, err)
	am.Refresh()
	
	// Test existing agent
	agent := am.GetAgentByUsername("bob.jones")
	assert.NotNil(t, agent)
	assert.Equal(t, "agent-2", agent.ID)
	assert.Equal(t, "Bob", agent.FirstName)
	
	// Test non-existing agent
	agent = am.GetAgentByUsername("non.existent")
	assert.Nil(t, agent)
}

type errorAgentLister struct{ err error }

func (e *errorAgentLister) ListUsers(ctx context.Context, input *connect.ListUsersInput) (*connect.ListUsersOutput, error) {
	return nil, e.err
}

func (e *errorAgentLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return nil, e.err
}

func (e *errorAgentLister) GetInstanceID() string {
	return "test-instance-id"
}

func TestAgentManagerListUsersError(t *testing.T) {
	lister := &errorAgentLister{err: fmt.Errorf("listusers error")}
	am, err := NewAgentManager(lister, time.Millisecond)
	assert.NoError(t, err)

	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	am.Refresh()
	assert.Contains(t, buf.String(), "failed to refresh agent list")
	assert.Empty(t, am.GetAgents())
}

type describeErrorAgentLister struct {
	*fakeAgentLister
}

func (e *describeErrorAgentLister) DescribeUser(ctx context.Context, input *connect.DescribeUserInput) (*connect.DescribeUserOutput, error) {
	return nil, fmt.Errorf("describe error")
}

func TestAgentManagerDescribeUserError(t *testing.T) {
	// Create a lister that succeeds for ListUsers but fails for DescribeUser
	agents := []Agent{{ID: "agent-1", Username: "test.user"}}
	baseLister := &fakeAgentLister{users: agents}
	lister := &describeErrorAgentLister{fakeAgentLister: baseLister}
	
	am, err := NewAgentManager(lister, time.Millisecond)
	assert.NoError(t, err)

	// Set debug level and capture output
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	prevLevel := logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.DebugLevel)
	defer func() {
		logrus.SetOutput(prev)
		logrus.SetLevel(prevLevel)
	}()

	am.Refresh()
	
	// Should still have agents with partial data
	agents = am.GetAgents()
	assert.Len(t, agents, 1)
	assert.Equal(t, "agent-1", agents[0].ID)
	assert.Equal(t, "test.user", agents[0].Username)
	// FirstName should be empty due to DescribeUser failure
	assert.Equal(t, "", agents[0].FirstName)
	
	// Should log the error
	assert.Contains(t, buf.String(), "failed to describe user, using partial data")
}

func TestAgentManagerDefaultInterval(t *testing.T) {
	lister := &fakeAgentLister{}
	am, err := NewAgentManager(lister, 0) // Zero interval should default to 5 minutes
	assert.NoError(t, err)
	// Can't easily test the actual interval without waiting, but we can test it doesn't crash
	assert.NotNil(t, am)
}
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

// minimal lister to seed routing profiles
type seedRPLister struct{}

func (s *seedRPLister) ListRoutingProfiles(ctx context.Context, input *connect.ListRoutingProfilesInput) (*connect.ListRoutingProfilesOutput, error) {
	return &connect.ListRoutingProfilesOutput{RoutingProfileSummaryList: []types.RoutingProfileSummary{{Id: aws.String("rp1"), Name: aws.String("Sales")}}}, nil
}
func (s *seedRPLister) GetInstanceID() string { return "inst" }

func TestRoutingProfileManagerReturnsCopy(t *testing.T) {
	m := NewRoutingProfileManager(&seedRPLister{}, time.Minute)
	assert.NoError(t, m.Start())
	defer m.Stop()
	time.Sleep(10 * time.Millisecond)
	list := m.GetRoutingProfiles()
	assert.Len(t, list, 1)
	// modify returned slice
	list[0].Name = "Modified"
	// fetch again should be unaffected
	list2 := m.GetRoutingProfiles()
	assert.Equal(t, "Sales", list2[0].Name)
}

// fake queue lister to seed queues
type seedQueueLister struct{}

func (s *seedQueueLister) ListQueues(ctx context.Context) ([]Queue, error) {
	return []Queue{{ID: "q1", Name: "Sales", Type: "STANDARD"}}, nil
}

func TestQueueManagerReturnsCopy(t *testing.T) {
	qm, err := NewQueueManager(&seedQueueLister{}, time.Minute, "")
	assert.NoError(t, err)
	qm.Refresh()
	qs := qm.GetQueues()
	assert.Len(t, qs, 1)
	qs[0].Name = "Changed"
	qs2 := qm.GetQueues()
	assert.Equal(t, "Sales", qs2[0].Name)
}

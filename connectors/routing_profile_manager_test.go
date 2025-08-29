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

// mockRPLister implements RoutingProfileLister with paginated results.
type mockRPLister struct {
	pages []*connect.ListRoutingProfilesOutput
	idx   int
}

func (m *mockRPLister) ListRoutingProfiles(ctx context.Context, input *connect.ListRoutingProfilesInput) (*connect.ListRoutingProfilesOutput, error) {
	if m.idx >= len(m.pages) {
		return &connect.ListRoutingProfilesOutput{}, nil
	}
	out := m.pages[m.idx]
	m.idx++
	return out, nil
}
func (m *mockRPLister) GetInstanceID() string { return "test-instance" }

func TestRoutingProfileManagerRefreshAndMap(t *testing.T) {
	l := &mockRPLister{pages: []*connect.ListRoutingProfilesOutput{
		{RoutingProfileSummaryList: []types.RoutingProfileSummary{{Id: aws.String("rp1"), Name: aws.String("Sales")}}, NextToken: aws.String("n2")},
		{RoutingProfileSummaryList: []types.RoutingProfileSummary{{Id: aws.String("rp2"), Name: aws.String("Support")}}},
	}}
	m := NewRoutingProfileManager(l, time.Minute)
	assert.NoError(t, m.Start())
	defer m.Stop()

	// Wait a tick to allow initial refresh to complete in Start()
	time.Sleep(10 * time.Millisecond)

	list := m.GetRoutingProfiles()
	assert.Len(t, list, 2)
	mp := m.GetRoutingProfileMap()
	assert.Equal(t, "Sales", mp["rp1"].Name)
	assert.Equal(t, "Support", mp["rp2"].Name)
}

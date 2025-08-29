package connectors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/stretchr/testify/assert"
)

type errRPLister struct{}

func (e *errRPLister) ListRoutingProfiles(ctx context.Context, input *connect.ListRoutingProfilesInput) (*connect.ListRoutingProfilesOutput, error) {
	return nil, errors.New("boom")
}
func (e *errRPLister) GetInstanceID() string { return "inst" }

func TestRoutingProfileManagerStartError(t *testing.T) {
	m := NewRoutingProfileManager(&errRPLister{}, time.Minute)
	err := m.Start()
	assert.Error(t, err)
}

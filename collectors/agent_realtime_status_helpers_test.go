package collectors

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractStatusID(t *testing.T) {
	arn := "arn:aws:connect:us-east-1:123:instance/abc/agent-state/STATE_ID"
	assert.Equal(t, "STATE_ID", extractStatusID(arn))
	assert.Equal(t, "simple", extractStatusID("simple"))
	assert.Equal(t, "", extractStatusID(""))
}

func TestGetStatusDetailsFromCache(t *testing.T) {
	arn := "arn:aws:connect:us-east-1:123:instance/abc/agent-state/STATE_ID"
	cache := map[string]types.AgentStatusSummary{
		arn: {Arn: aws.String(arn), Name: aws.String("Available"), Type: types.AgentStatusTypeRoutable},
	}
	st, ok := getStatusDetailsFromCache(arn, cache)
	assert.True(t, ok)
	assert.Equal(t, "Available", aws.ToString(st.Name))

	_, ok = getStatusDetailsFromCache("missing", cache)
	assert.False(t, ok)
}

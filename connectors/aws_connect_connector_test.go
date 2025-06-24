package connectors

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/stretchr/testify/assert"
)

func TestIsDevelopmentMode(t *testing.T) {
	origEnv := os.Getenv("ENV")
	origNodeEnv := os.Getenv("NODE_ENV")
	origMock := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	defer func() {
		os.Setenv("ENV", origEnv)
		os.Setenv("NODE_ENV", origNodeEnv)
		os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", origMock)
	}()

	os.Unsetenv("ENV")
	os.Unsetenv("NODE_ENV")
	os.Unsetenv("AWS_CONNECT_MOCK_ENDPOINT")
	assert.False(t, isDevelopmentMode(), "no env vars should not trigger dev mode")

	os.Setenv("ENV", "development")
	assert.True(t, isDevelopmentMode(), "ENV=development should trigger dev mode")

	os.Setenv("ENV", "")
	os.Setenv("NODE_ENV", "dev")
	assert.True(t, isDevelopmentMode(), "NODE_ENV=dev should trigger dev mode")

	os.Unsetenv("NODE_ENV")
	os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", "http://example.com")
	assert.True(t, isDevelopmentMode(), "Mock endpoint set should trigger dev mode")
}

func TestGetMockEndpoint(t *testing.T) {
	orig := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	defer os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", orig)

	os.Unsetenv("AWS_CONNECT_MOCK_ENDPOINT")
	assert.Equal(t, "http://localhost:3000", getMockEndpoint(), "default endpoint when env not set")

	os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", "http://example.com")
	assert.Equal(t, "http://example.com", getMockEndpoint(), "env var should override")
}

func TestGetInstanceID(t *testing.T) {
	c := &AWSConnectConnector{instanceARN: "arn:aws:connect:us-east-1:123:instance/abcdef"}
	assert.Equal(t, "abcdef", c.GetInstanceID())

	c.instanceARN = "simple-id"
	assert.Equal(t, "simple-id", c.GetInstanceID())

	c.instanceARN = "arn:aws:connect:region:acct:instance/id/extra"
	assert.Equal(t, "extra", c.GetInstanceID())
}

func TestCreateCurrentMetricInput(t *testing.T) {
	input := CreateCurrentMetricInput("inst", []string{"q1", "q2"})

	assert.NotNil(t, input)
	assert.Equal(t, "inst", *input.InstanceId)
	assert.Equal(t, []string{"q1", "q2"}, input.Filters.Queues)
	if assert.Len(t, input.CurrentMetrics, 2) {
		assert.Equal(t, types.CurrentMetricNameContactsInQueue, input.CurrentMetrics[0].Name)
		assert.Equal(t, types.UnitCount, input.CurrentMetrics[0].Unit)
		assert.Equal(t, types.CurrentMetricNameOldestContactAge, input.CurrentMetrics[1].Name)
		assert.Equal(t, types.UnitSeconds, input.CurrentMetrics[1].Unit)
	}
}

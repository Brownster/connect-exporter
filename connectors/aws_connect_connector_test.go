// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file contains the unit tests for the AWSConnectConnector. The tests focus
// on the helper functions and the logic for handling development mode.
package connectors

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsDevelopmentMode tests the isDevelopmentMode helper function to ensure it
// correctly identifies when the exporter should run in development mode.
func TestIsDevelopmentMode(t *testing.T) {
	origEnv := os.Getenv("ENV")
	origNodeEnv := os.Getenv("NODE_ENV")
	origMock := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	t.Cleanup(func() {
		_ = os.Setenv("ENV", origEnv)
		_ = os.Setenv("NODE_ENV", origNodeEnv)
		_ = os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", origMock)
	})

	assert.NoError(t, os.Unsetenv("ENV"))
	assert.NoError(t, os.Unsetenv("NODE_ENV"))
	assert.NoError(t, os.Unsetenv("AWS_CONNECT_MOCK_ENDPOINT"))
	assert.False(t, isDevelopmentMode(), "no env vars should not trigger dev mode")

	assert.NoError(t, os.Setenv("ENV", "development"))
	assert.True(t, isDevelopmentMode(), "ENV=development should trigger dev mode")

	assert.NoError(t, os.Setenv("ENV", ""))
	assert.NoError(t, os.Setenv("NODE_ENV", "dev"))
	assert.True(t, isDevelopmentMode(), "NODE_ENV=dev should trigger dev mode")

	assert.NoError(t, os.Unsetenv("NODE_ENV"))
	assert.NoError(t, os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", "http://example.com"))
	assert.True(t, isDevelopmentMode(), "Mock endpoint set should trigger dev mode")
}

// TestGetMockEndpoint tests the getMockEndpoint helper function to ensure it
// returns the correct mock API endpoint.
func TestGetMockEndpoint(t *testing.T) {
	orig := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	t.Cleanup(func() { _ = os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", orig) })

	assert.NoError(t, os.Unsetenv("AWS_CONNECT_MOCK_ENDPOINT"))
	assert.Equal(t, "http://localhost:3000", getMockEndpoint(), "default endpoint when env not set")

	assert.NoError(t, os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", "http://example.com"))
	assert.Equal(t, "http://example.com", getMockEndpoint(), "env var should override")
}

// TestGetInstanceID tests the GetInstanceID helper function to ensure it
// correctly extracts the instance ID from an ARN.
func TestGetInstanceID(t *testing.T) {
	c := &AWSConnectConnector{instanceARN: "arn:aws:connect:us-east-1:123:instance/abcdef"}
	assert.Equal(t, "abcdef", c.GetInstanceID())

	c.instanceARN = "simple-id"
	assert.Equal(t, "simple-id", c.GetInstanceID())

	c.instanceARN = "arn:aws:connect:region:acct:instance/id/extra"
	assert.Equal(t, "extra", c.GetInstanceID())
}

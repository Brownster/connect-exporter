package connectors

import (
	"os"
	"strings"
)

// Development and mocking helpers extracted from aws_connect_connector.go

// isDevelopmentMode checks if we should use the mock API
func isDevelopmentMode() bool {
	// Check for development environment variables
	env := strings.ToLower(os.Getenv("ENV"))
	if env == "development" || env == "dev" {
		return true
	}

	nodeEnv := strings.ToLower(os.Getenv("NODE_ENV"))
	if nodeEnv == "development" || nodeEnv == "dev" {
		return true
	}

	// Check if mock endpoint is explicitly configured
	mockEndpoint := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	return mockEndpoint != ""
}

// getMockEndpoint returns the mock API endpoint
func getMockEndpoint() string {
	endpoint := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:3000"
	}
	return endpoint
}

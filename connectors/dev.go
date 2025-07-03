// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file contains helper functions for development and testing. These functions
// are used to enable the mock API and to configure the exporter to use it.
// By isolating this logic, the main connector code can remain focused on production
// functionality.
package connectors

import (
	"os"
	"strings"
)

// isDevelopmentMode checks if the exporter should run in development mode.
// Development mode is enabled by setting the ENV or NODE_ENV environment variable
// to "development" or "dev", or by setting the AWS_CONNECT_MOCK_ENDPOINT
// environment variable.
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

// getMockEndpoint returns the mock API endpoint. If the AWS_CONNECT_MOCK_ENDPOINT
// environment variable is set, it will be used. Otherwise, it defaults to
// http://localhost:3000.
func getMockEndpoint() string {
	endpoint := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:3000"
	}
	return endpoint
}

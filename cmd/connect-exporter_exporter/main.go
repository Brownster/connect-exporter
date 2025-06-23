package main

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sabio-engineering-product/connect-exporter/collectors"
	log "github.com/sirupsen/logrus"
)

const sentryTimeout = 5 * time.Second

func main() {
	// Create the AWS Connect exporter
	// The exporter will handle connector creation based on configuration
	exporter, err := collectors.NewAWSConnectExporter()
	if err != nil {
		log.Fatal(err)
	}

	// Start all scheduled scrapers
	exporter.Start()
	
	// Start HTTP server for metrics endpoint
	exporter.Serve()

	// Ensure Sentry events are sent before shutdown
	sentry.Flush(sentryTimeout)
}

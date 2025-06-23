package main

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sabio-engineering-product/connect-exporter/collectors"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	log "github.com/sirupsen/logrus"
)

const sentryTimeout = 5 * time.Second

func main() {
	// TEMPLATE: Choose the appropriate connector for your data source
	// Examples:
	// - For local/SSH commands: Use command.NewSSH() (production) or connectors.NewMockConnector() (testing)
	// - For HTTP APIs: connectors.NewTemplateHTTPConnector()
	// - For SNMP: Use snmp.NewSNMP() (production)
	
	// For this template example, we'll use HTTP connector
	connector := connectors.NewTemplateHTTPConnector()

	// Create the exporter using our template constructor
	exporter, err := collectors.NewConnectExporter(connector)
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

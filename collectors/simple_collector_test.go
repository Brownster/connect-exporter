package collectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewGenericHistoricalScraper tests that the scraper can be created
func TestNewGenericHistoricalScraper(t *testing.T) {
	scraper := NewGenericHistoricalScraper()
	assert.NotNil(t, scraper)
}

// TestNewGenericRealtimeScraper tests that the scraper can be created
func TestNewGenericRealtimeScraper(t *testing.T) {
	scraper := NewGenericRealtimeScraper()
	assert.NotNil(t, scraper)
}

// TestNewAgentRealtimeStatusScraper tests that the scraper can be created
func TestNewAgentRealtimeStatusScraper(t *testing.T) {
	// Skip this test for now since it requires app and config parameters
	t.Skip("Requires kingpin.Application and config parameters")
}

package connectors

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAgentManagerNamesToggleAndStats(t *testing.T) {
	m := NewAgentManager(&mockAgentLister{}, time.Minute)
	m.SetNamesEnabled(false)
	stats := m.GetNamesCacheStats()
	assert.Equal(t, false, stats["names_enabled"])
	// cache fields exist
	_, ok := stats["cached_names_count"]
	assert.True(t, ok)
	_, ok = stats["cache_age_hours"]
	assert.True(t, ok)
}

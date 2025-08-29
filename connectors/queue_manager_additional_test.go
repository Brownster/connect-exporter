package connectors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueueManagerInvalidRegex(t *testing.T) {
	qm, err := NewQueueManager(&fakeLister{}, time.Millisecond, "[invalid")
	assert.Nil(t, qm)
	assert.Error(t, err)
}

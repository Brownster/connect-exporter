package connectors

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsRateLimitError(t *testing.T) {
	assert.False(t, isRateLimitError(nil))
	assert.True(t, isRateLimitError(errors.New("Throttling")))
	assert.True(t, isRateLimitError(errors.New("TooManyRequestsException")))
	assert.True(t, isRateLimitError(errors.New("ThrottlingException")))
	assert.False(t, isRateLimitError(errors.New("SomeOtherError")))
}

package connectors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExecuteUnsupportedString(t *testing.T) {
	c := &AWSConnectConnector{}
	_, err := c.Execute("unsupported_command")
	assert.Error(t, err)
}

package connectors

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMockConnectorBasicResponses(t *testing.T) {
	m := NewMockConnector()
	m.AddResponse("ping", "pong")
	out, err := m.Execute("ping")
	assert.NoError(t, err)
	assert.Equal(t, "pong", out)

	m.AddError("fail", errors.New("boom"))
	_, err = m.Execute("fail")
	assert.Error(t, err)

	m.AddResponseWithDelay("slow", "ok", 1)
	start := time.Now()
	out, err = m.Execute("slow")
	assert.NoError(t, err)
	assert.Equal(t, "ok", out)
	assert.GreaterOrEqual(t, time.Since(start).Milliseconds(), int64(1))

	// whitespace trimming on AddResponseFromFile
	m.AddResponseFromFile("file", " value \n")
	out, err = m.Execute("file")
	assert.NoError(t, err)
	assert.Equal(t, "value", out)

	// call count
	assert.Equal(t, 1, m.GetCallCount("ping"))
}

func TestMockConnectorSystemCommandsAndClear(t *testing.T) {
	m := NewMockConnector()
	m.AddSystemCommandResponse()
	_, err := m.Execute("uname -a")
	assert.NoError(t, err)
	m.ClearResponses()
	// After clear, default response is returned
	out, err := m.Execute("uname -a")
	assert.NoError(t, err)
	assert.Contains(t, out.(string), "mock response for:")
}

func TestMockQueryString(t *testing.T) {
	q := MockQuery{Command: "cmd", Args: []string{"a", "b"}}
	assert.Equal(t, "cmd a b", q.String())
	q = MockQuery{Command: "cmd"}
	assert.Equal(t, "cmd", q.String())
}

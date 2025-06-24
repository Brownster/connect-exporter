package connectors

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockConnectorAddResponseAndCallCount(t *testing.T) {
	m := NewMockConnector()
	m.AddResponse("query", "value")

	resp, err := m.Execute("query")
	if assert.NoError(t, err) {
		assert.Equal(t, "value", resp)
	}
	assert.Equal(t, 1, m.GetCallCount("query"))

	// second call should increment call count
	resp, err = m.Execute("query")
	if assert.NoError(t, err) {
		assert.Equal(t, "value", resp)
	}
	assert.Equal(t, 2, m.GetCallCount("query"))
}

func TestMockConnectorAddError(t *testing.T) {
	m := NewMockConnector()
	expected := errors.New("boom")
	m.AddError("err", expected)

	resp, err := m.Execute("err")
	assert.Nil(t, resp)
	assert.Equal(t, expected, err)
	assert.Equal(t, 1, m.GetCallCount("err"))
}

func TestMockConnectorAddResponseWithDelay(t *testing.T) {
	m := NewMockConnector()
	m.AddResponseWithDelay("slow", "delayed", 100)

	done := make(chan interface{}, 1)
	go func() {
		r, _ := m.Execute("slow")
		done <- r
	}()

	time.Sleep(50 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("Execute returned too early")
	default:
	}

	time.Sleep(70 * time.Millisecond)
	select {
	case r := <-done:
		assert.Equal(t, "delayed", r)
	default:
		t.Fatal("Execute did not return after expected delay")
	}

	assert.Equal(t, 1, m.GetCallCount("slow"))
}

package connectors

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfigureSetsDevelopmentModeWhenMockEndpointProvided(t *testing.T) {
	orig := os.Getenv("AWS_CONNECT_MOCK_ENDPOINT")
	t.Cleanup(func() { _ = os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", orig) })
	_ = os.Setenv("AWS_CONNECT_MOCK_ENDPOINT", "http://127.0.0.1:3000")

	// temp config with required fields
	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.yml")
	assert.NoError(t, err)
	_, _ = tmp.WriteString("aws:\n  region: us-east-1\n  instance_arn: arn:aws:connect:us-east-1:123:instance/abc\n")
	_ = tmp.Close()

	app := kingpin.New("test", "")
	c, err := NewConfigurableAWSConnectConnector(app)
	assert.NoError(t, err)
	*c.configFileFlag = tmp.Name()

	err = c.configure()
	assert.NoError(t, err)
	// developmentMode should be true when mock endpoint env is set
	assert.True(t, c.developmentMode)
	assert.Equal(t, "http://127.0.0.1:3000", c.mockEndpoint)
}

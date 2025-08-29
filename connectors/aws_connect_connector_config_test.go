package connectors

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfigureMissingARNFails(t *testing.T) {
	app := kingpin.New("test", "")
	c, err := NewConfigurableAWSConnectConnector(app)
	assert.NoError(t, err)
	// config.file flag defaults to config.yml which won’t exist; missing ARN should error
	err = c.configure()
	assert.Error(t, err)
}

func TestConfigureWithConfigFileSucceeds(t *testing.T) {
	// Create temp config file
	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.yml")
	assert.NoError(t, err)
	_, _ = tmp.WriteString("aws:\n  region: us-east-1\n  instance_arn: arn:aws:connect:us-east-1:123:instance/abc\n")
	_ = tmp.Close()

	app := kingpin.New("test", "")
	c, err := NewConfigurableAWSConnectConnector(app)
	assert.NoError(t, err)
	// Point connector at our temp config path
	*c.configFileFlag = tmp.Name()

	err = c.configure()
	assert.NoError(t, err)
	cfg := c.GetConfig()
	assert.Equal(t, "us-east-1", cfg.AWS.Region)
	assert.Equal(t, "arn:aws:connect:us-east-1:123:instance/abc", cfg.AWS.InstanceARN)

	// defaults applied when not set in file
	assert.Equal(t, 30, cfg.Collectors.GenericRealtime.Interval)
	assert.Equal(t, 300, cfg.Collectors.GenericHistorical.Interval)
	// realtime queue defaults
	assert.Equal(t, 60, cfg.Collectors.RealtimeQueue.Interval)
}

func TestConfigureRegionFromEnv(t *testing.T) {
	orig := os.Getenv("AWS_REGION")
	t.Cleanup(func() { _ = os.Setenv("AWS_REGION", orig) })
	_ = os.Setenv("AWS_REGION", "eu-west-2")

	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.yml")
	assert.NoError(t, err)
	// omit region
	_, _ = tmp.WriteString("aws:\n  instance_arn: arn:aws:connect:eu-west-2:123:instance/abc\n")
	_ = tmp.Close()

	app := kingpin.New("test", "")
	c, err := NewConfigurableAWSConnectConnector(app)
	assert.NoError(t, err)
	*c.configFileFlag = tmp.Name()
	err = c.configure()
	assert.NoError(t, err)
	cfg := c.GetConfig()
	assert.Equal(t, "eu-west-2", cfg.AWS.Region)
}

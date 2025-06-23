package maas

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/alecthomas/kingpin/v2"
)

type LabelTestSuite struct {
	suite.Suite
	*kingpin.Application
}

func (s *LabelTestSuite) SetupTest() {
	s.Application = kingpin.New("test", "help")
}

func (s *LabelTestSuite) TestDefaultLabels() {
	d := NewDefaultLabels()
	s.Implements(new(Labeler), d)
	d.Flags(s.Application)

	_, err := s.Application.Parse([]string{
		"--labels.fqdn=maas.sabio.co.uk",
		"--labels.location=glasgow",
		"--labels.ip-address=127.0.0.1",
		"--labels.country=GB",
	})

	s.NoError(err)

	l := d.Labels()

	tests := []struct {
		key   string
		value string
	}{
		{key: "fqdn", value: "maas.sabio.co.uk"},
		{key: "exporter", value: "test"},
		{key: "environment", value: "production"},
		{key: "location", value: "glasgow"},
		{key: "country", value: "GB"},
		{key: "ip_address", value: "127.0.0.1"},
	}

	for _, tc := range tests {
		s.Contains(l, tc.key)
		s.Equal(tc.value, l[tc.key])
	}
}

func TestLabelTestSuite(t *testing.T) {
	suite.Run(t, new(LabelTestSuite))
}

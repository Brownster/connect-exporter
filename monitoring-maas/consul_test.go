package maas

import (
	"testing"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/suite"
	"github.com/alecthomas/kingpin/v2"
)

type ConsulTestSuite struct {
	suite.Suite
	*Consul
	*kingpin.Application
	Labeler
}

const expectedServices = 1
const expectedPort = 9100

func (s *ConsulTestSuite) SetupTest() {
	s.Application = kingpin.New("test", "help")
	s.Labeler = NewDefaultLabels()
	s.Labeler.Flags(s.Application)
}

func (s *ConsulTestSuite) TestRegisters() {
	srv, err := testutil.NewTestServerT(s.T())
	s.NoError((err))

	cfg := &consul.Config{Address: srv.HTTPAddr}

	c := NewConsul(WithConfig(cfg))

	c.Flags(s.Application)

	_, err = s.Application.Parse([]string{
		"--labels.fqdn=maas.sabio.co.uk",
		"--labels.location=glasgow",
		"--labels.ip-address=192.168.0.1",
		"--labels.country=GB",
	})

	s.NoError(err)

	err = c.Advertise("127.0.0.1", 9100, s.Labeler)

	s.NoError(err)

	cs, err := consul.NewClient(cfg)
	s.NoError(err)
	svcs, err := cs.Agent().Services()

	s.NoError(err)

	s.Equal(expectedServices, len(svcs))

	s.Contains(svcs, "127.0.0.1:9100")

	svc := svcs["127.0.0.1:9100"]
	s.Len(svc.Tags, 6)
	s.Equal("127.0.0.1", svc.Address)
	s.Equal(expectedPort, svc.Port)

	tests := []struct {
		key string
	}{
		{key: "fqdn=maas.sabio.co.uk"},
		{key: "exporter=test"},
		{key: "environment=production"},
		{key: "location=glasgow"},
		{key: "country=GB"},
		{key: "ip_address=192.168.0.1"},
	}

	for _, tc := range tests {
		s.Contains(svc.Tags, tc.key)
	}

	defer srv.Stop()
}

func TestConsultestSuite(t *testing.T) {
	suite.Run(t, new(ConsulTestSuite))
}

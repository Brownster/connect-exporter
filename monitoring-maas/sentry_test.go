package maas

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type SentryTestSuite struct {
	suite.Suite
}

func (s *SentryTestSuite) TestNoDSN() {
	err := RegisterSentryHook(NewDefaultLabels())

	s.EqualError(err, "no DSN provided")
	s.Len(log.StandardLogger().Hooks, 7)
}

func (s *SentryTestSuite) TestNoEnvironemnt() {
	SentryDSN = "https://DEADBEEF@sentry.maas.sabio.co.uk/8"
	err := RegisterSentryHook(&MockLabels{})

	s.EqualError(err, "no environment provided")
	s.Len(log.StandardLogger().Hooks, 7)
}

func (s *SentryTestSuite) TestVarDSN() {
	SentryDSN = "https://DEADBEEF@sentry.maas.sabio.co.uk/8"

	err := RegisterSentryHook(NewDefaultLabels())

	s.NoError(err)
	s.Len(log.StandardLogger().Hooks, 7)
}

func TestSentryTestSuite(t *testing.T) {
	suite.Run(t, new(SentryTestSuite))
}

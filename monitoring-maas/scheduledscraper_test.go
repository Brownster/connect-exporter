package maas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/alecthomas/kingpin/v2"
)

type ScheduledScraperTestSuite struct {
	suite.Suite
	*kingpin.Application
}

func (s *ScheduledScraperTestSuite) SetupTest() {
	s.Application = kingpin.New("app", "help")
}

func (s *ScheduledScraperTestSuite) TestFlags() {
	const expectedFrequency = 30 * time.Second

	const expectedTimeout = 5 * time.Second

	ss := NewScheduledScraper("test", &MockScraper{})

	ss.Flags(s.Application)

	_, err := s.Application.Parse([]string{
		"--test.frequency=30s",
		"--test.timeout=5s",
		"--no-test.enabled",
	})
	s.NoError(err)
	s.Equal(ss.schedule.frequency, expectedFrequency)
	s.Equal(ss.schedule.timeout, expectedTimeout)
	s.False(ss.schedule.isEnabled)
}

func TestScheduledScraperTestSuite(t *testing.T) {
	suite.Run(t, new(ScheduledScraperTestSuite))
}

package maas

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/alecthomas/kingpin/v2"
)

type ExporterTestSuite struct {
	suite.Suite
	*kingpin.Application
}

func (s *ExporterTestSuite) SetupTest() {
	s.Application = kingpin.New("app", "test app")
}

func (s *ExporterTestSuite) TestFailedConnect() {
	_, err := NewExporter(s.Application, &FailConnector{},
		WithLabels(&MockLabels{}),
		WithArgs([]string{
			"--web.listen-port=9100",
		}))
	s.EqualError(err, "unable to connect: down")
}

func (s *ExporterTestSuite) TestSuccessConnect() {
	_, err := NewExporter(s.Application, &SuccessConnector{},
		WithLabels(&MockLabels{}),
		WithArgs([]string{
			"--web.listen-port=9100",
		}))
	s.NoError(err)
}

func (s *ExporterTestSuite) TestSchedulesScrapes() {
	e, err := NewExporter(s.Application, &SuccessConnector{},
		WithArgs([]string{
			"--web.listen-port=9100",
		}),
		WithScheduler(NewMockScheduler()),
		WithLabels(&MockLabels{}),
		WithScheduledScrapers(
			NewScheduledScraper("mock", MockScraper{}),
		),
	)
	s.NoError(err)
	s.Len(e.scheduler.Entries(), 1)

	e.Start()
}

func TestExporterTestSuite(t *testing.T) {
	suite.Run(t, new(ExporterTestSuite))
}

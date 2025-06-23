package maas

import (
	"errors"

	"github.com/robfig/cron/v3"
	"github.com/alecthomas/kingpin/v2"
)

type MockScraper struct{}

func (s MockScraper) Scrape(c Connector) ([]Metric, error) {
	return nil, nil
}

type MockLabels struct{}

func (l *MockLabels) Labels() map[string]string {
	return nil
}
func (l *MockLabels) Flags(a *kingpin.Application) {}

type MockScheduler struct {
	*cron.Cron
}

func NewMockScheduler() *MockScheduler {
	return &MockScheduler{
		Cron: cron.New(cron.WithSeconds()),
	}
}

func (s *MockScheduler) Start() {
	for _, e := range s.Entries() {
		e.Job.Run()
	}
}

type FailConnector struct{}

func (c *FailConnector) Connect() error {
	return errors.New("down")
}

func (c *FailConnector) Flags(a *kingpin.Application) {}

func (c *FailConnector) Execute(interface{}) (interface{}, error) {
	return nil, nil
}

type SuccessConnector struct{}

func (c *SuccessConnector) Connect() error {
	return nil
}

func (c *SuccessConnector) Flags(a *kingpin.Application) {}

func (c *SuccessConnector) Execute(interface{}) (interface{}, error) {
	return nil, nil
}

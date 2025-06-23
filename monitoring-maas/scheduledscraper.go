package maas

import (
	"context"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"github.com/alecthomas/kingpin/v2"
)

type Scheduler interface {
	Entries() []cron.Entry
	Start()
	Stop() context.Context
	AddFunc(spec string, cmd func()) (cron.EntryID, error)
}

type ScheduledScraper struct {
	name         string
	schedule     *Schedule
	scraper      Scraper
	descriptions map[string]*prometheus.Desc
}

func NewScheduledScraper(name string, sc Scraper, options ...func(*ScheduledScraper)) *ScheduledScraper {
	s := &ScheduledScraper{
		name:         name,
		scraper:      sc,
		schedule:     NewSchedule(),
		descriptions: make(map[string]*prometheus.Desc),
	}

	s.apply(options)

	return s
}

func (s ScheduledScraper) Flags(a *kingpin.Application) {
	a.Flag(fmt.Sprintf("%s.frequency", s.name), fmt.Sprintf("Frequency to scrape %s", s.name)).
		Default(s.schedule.frequency.String()).
		DurationVar(&s.schedule.frequency)

	a.Flag(fmt.Sprintf("%s.timeout", s.name), fmt.Sprintf("Timeout for scraping %s", s.name)).
		Default(s.schedule.timeout.String()).
		DurationVar(&s.schedule.timeout)

	a.Flag(fmt.Sprintf("%s.enabled", s.name), fmt.Sprintf("Is scraper %s enabled", s.name)).
		Default(strconv.FormatBool(s.schedule.isEnabled)).
		BoolVar(&s.schedule.isEnabled)
}

func WithSchedule(sch *Schedule) func(*ScheduledScraper) {
	return func(s *ScheduledScraper) {
		s.schedule = sch
	}
}

func WithDescription(
	a *kingpin.Application,
	name string,
	help string,
	labels []string,
) func(*ScheduledScraper) {
	return func(s *ScheduledScraper) {
		s.descriptions[name] = prometheus.NewDesc(
			prometheus.BuildFQName(a.Name, s.name, name),
			help,
			labels,
			nil,
		)
	}
}

func (s *ScheduledScraper) ScheduleSpec() string {
	return fmt.Sprintf("@every %v", s.schedule.frequency)
}

func (s *ScheduledScraper) apply(options []func(*ScheduledScraper)) {
	for _, option := range options {
		option(s)
	}
}

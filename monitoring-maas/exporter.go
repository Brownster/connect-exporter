package maas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/ArthurHlt/logrusprom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"github.com/alecthomas/kingpin/v2"
)

type Exporter struct {
	application       *kingpin.Application
	scheduler         Scheduler
	connector         Connector
	args              []string
	logLevel          string
	listenAddress     string
	listenPort        int
	telemetryPath     string
	labels            Labeler
	shouldAdvertise   bool
	shouldDescribe    bool
	advertiser        Advertiser
	scrapeFrequency   *prometheus.GaugeVec
	scrapeSuccess     *prometheus.GaugeVec
	scrapeTotal       *prometheus.CounterVec
	scrapeFails       *prometheus.CounterVec
	scrapeTimeouts    *prometheus.CounterVec
	scrapeDuration    *prometheus.HistogramVec
	scrapeLastSuccess *prometheus.GaugeVec
	scheduledscrapers []*ScheduledScraper
	metrics           map[string]*Metrics
	registry          *prometheus.Registry
}

func NewExporter(a *kingpin.Application, c Connector, options ...func(*Exporter)) (*Exporter, error) {
	var metricLabels = []string{"exporter", "scraper"}

	e := &Exporter{
		application:       a,
		connector:         c,
		args:              os.Args[1:],
		scheduler:         cron.New(cron.WithSeconds()),
		metrics:           make(map[string]*Metrics),
		registry:          prometheus.NewRegistry(),
		labels:            NewDefaultLabels(),
		advertiser:        NewConsul(),
		scrapeFrequency:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "maas_scrape_frequency_seconds", Help: "Scrape frequency"}, metricLabels),
		scrapeSuccess:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "maas_scrape_success", Help: "Was the last scrape successful"}, metricLabels),
		scrapeLastSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "maas_scrape_last_success_seconds", Help: "When the last successful scrape was"}, metricLabels),
		scrapeTotal:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "maas_scrape_total", Help: "Total number of scrapes"}, metricLabels),
		scrapeTimeouts:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "maas_scrape_timeout_total", Help: "Total number of scrapes that have timed out"}, metricLabels),
		scrapeFails:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "maas_scrape_failed_total", Help: "Total number of failed scrapes"}, metricLabels),
		scrapeDuration:    prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "maas_scrape_duration", Help: "Scrape Duration"}, metricLabels),
	}
	e.apply(options)

	if err := e.flags(); err != nil {
		return nil, err
	}

	e.configureLogger()

	if err := c.Connect(); err != nil {
		return nil, UnableToConnectError{Err: err}
	}

	if err := e.schedule(); err != nil {
		return nil, err
	}

	log.Infof("Starting %s_exporter: %s\n", e.application.Name, version.Info())
	log.Infoln("Build context", version.BuildContext())

	e.registry.MustRegister(
		e,
		prometheus.NewBuildInfoCollector(),
		prometheus.NewGoCollector(),
		e.scrapeFrequency,
		e.scrapeSuccess,
		e.scrapeTotal,
		e.scrapeFails,
		e.scrapeLastSuccess,
		e.scrapeDuration,
		e.scrapeTimeouts,
	)

	return e, nil
}

func (e *Exporter) Start() {
	for _, s := range e.scheduler.Entries() {
		log.Debugf("Running Job %d", s.ID)
		s.Job.Run()
	}

	e.scheduler.Start()
}

func (e *Exporter) Serve() {
	err := e.advertise()

	if err != nil {
		log.Fatalf("unable to advertise exporter: %s", err)
	}

	socket := fmt.Sprintf("%s:%d", e.listenAddress, e.listenPort)

	http.Handle(e.telemetryPath, promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{}))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "PONG")
	})

	log.Infof("starting exporter on http://%s%s", socket, e.telemetryPath)

	err = http.ListenAndServe(socket, nil)

	if err != nil {
		log.Fatalf("couldn't start HTTP server: %s", err)
	}
}

func (e *Exporter) advertise() error {
	if !e.shouldAdvertise {
		return nil
	}
	if e.advertiser == nil {
		return errors.New("no valid advertister found")
	}

	return e.advertiser.Advertise(e.listenAddress, e.listenPort, e.labels)
}

func (e *Exporter) configureLogger() {
	level, err := log.ParseLevel(e.logLevel)

	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)
	log.SetFormatter(&log.JSONFormatter{})

	err = RegisterSentryHook(e.labels)

	if err != nil {
		log.Warnf("Unable to register sentry hook: %s", err)
	}

	e.registry.MustRegister(logrusprom.Collector())
}

func (e *Exporter) apply(options []func(*Exporter)) {
	for _, option := range options {
		option(e)
	}
}

func (e Exporter) describe() {
	fmt.Print(e.parseMetrics().ToMarkdown())
	os.Exit(0)
}

func (e Exporter) parseMetrics() Descriptions {
	descs := make(Descriptions)

	for _, s := range e.scheduledscrapers {
		iter := reflect.ValueOf(*s).FieldByName("descriptions").MapRange()

		for iter.Next() {
			metric := reflect.Indirect(iter.Value()).FieldByName("fqName").String()
			help := reflect.Indirect(iter.Value()).FieldByName("help").String()
			labels := fmt.Sprint(reflect.Indirect(iter.Value()).FieldByName("variableLabels"))
			labels = strings.ReplaceAll(labels, "[", "")
			labels = strings.ReplaceAll(labels, "]", "")
			labels = strings.ReplaceAll(labels, " ", ", ")

			descs[metric] = Description{
				Name:   metric,
				Help:   help,
				Labels: labels,
			}
		}
	}

	return descs
}

func (e *Exporter) flags() error {
	e.application.Flag(
		"advertise",
		"Should the exporter advertise itself",
	).Default(strconv.FormatBool(true)).BoolVar(&e.shouldAdvertise)

	e.application.Flag(
		"describe",
		"Produce documentation only",
	).Default(strconv.FormatBool(false)).BoolVar(&e.shouldDescribe)

	if e.advertiser != nil {
		e.advertiser.Flags(e.application)
	}

	e.connector.Flags(e.application)
	e.labels.Flags(e.application)
	e.webFlags()
	e.logflags()

	for _, s := range e.scheduledscrapers {
		s.Flags(e.application)
	}

	e.application.PreAction(func(c *kingpin.ParseContext) error {
		for _, el := range c.Elements {
			if f, ok := el.Clause.(*kingpin.FlagClause); ok {
				if f.Model().Name == "describe" {
					b, _ := strconv.ParseBool(f.Model().Value.String())
					if b {
						e.describe()
					}
				}
			}
		}
		return nil
	})

	_, err := e.application.Parse(e.args)

	return err
}

func (e *Exporter) webFlags() {
	e.application.Flag(
		"web.listen-address",
		"Address on which to expose metrics and web interface",
	).Default("127.0.0.1").StringVar(&e.listenAddress)

	e.application.Flag(
		"web.listen-port",
		"Port on which to expose metrics and web interface",
	).Required().IntVar(&e.listenPort)

	e.application.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics",
	).Default("/metrics").StringVar(&e.telemetryPath)
}

func (e *Exporter) logflags() {
	e.application.Flag(
		"log.level",
		"Only log messages with the given severity or above. Valid levels: [trace, debug, info, warn, error, fatal]",
	).Default("error").StringVar(&e.logLevel)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, m := range e.metrics {
		m.Collect(ch)
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(e, ch)
}

func (e *Exporter) schedule() error {
	for _, ss := range e.scheduledscrapers {
		s := ss
		if !s.schedule.isEnabled {
			log.Warnf("Scraper %s is not enabled", s.name)
			continue
		}

		e.metrics[s.name] = NewMetrics()
		e.scrapeFrequency.WithLabelValues(e.application.Name, s.name).Set(s.schedule.frequency.Seconds())
		e.scrapeFails.WithLabelValues(e.application.Name, s.name)
		e.scrapeTotal.WithLabelValues(e.application.Name, s.name)
		e.scrapeTimeouts.WithLabelValues(e.application.Name, s.name)
		id, err := e.scheduler.AddFunc(s.ScheduleSpec(), func() {
			ctx, cancel := context.WithTimeout(context.Background(), s.schedule.timeout)
			defer cancel()

			log.Tracef("Scraping %s", s.name)

			begin := time.Now()
			metrics, err := s.scraper.Scrape(e.connector)
			e.scrapeDuration.WithLabelValues(e.application.Name, s.name).Observe(time.Since(begin).Seconds())
			e.scrapeTotal.WithLabelValues(e.application.Name, s.name).Inc()
			if err != nil {
				log.Warnf("%s: scrape failed: %s", s.name, err)
				e.scrapeFails.WithLabelValues(e.application.Name, s.name).Inc()
				return
			}
			log.Tracef("%s: Received Metrics: %+v", s.name, metrics)
			e.metrics[s.name].Put(e.convertMetrics(s, metrics))
			select {
			case <-ctx.Done():
				e.scrapeTimeouts.WithLabelValues(e.application.Name, s.name).Inc()
				log.Warnf("%s: scrape timed out: %s", s.name, ctx.Err())
			default:
				e.scrapeLastSuccess.WithLabelValues(e.application.Name, s.name).SetToCurrentTime()
			}
		})

		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("scheduled %s with ID: %d every %s", ss.name, id, s.schedule.frequency))
	}

	return nil
}

func (e *Exporter) convertMetrics(s *ScheduledScraper, metrics []Metric) []prometheus.Metric {
	pm := make([]prometheus.Metric, 0, len(metrics))

	for _, m := range metrics {
		pm = append(pm, prometheus.MustNewConstMetric(s.descriptions[m.name], m.valueType, m.value, m.labels...))
	}

	return pm
}

func WithArgs(args []string) func(*Exporter) {
	return func(e *Exporter) {
		e.args = args
	}
}

func WithLabels(l Labeler) func(*Exporter) {
	return func(e *Exporter) {
		e.labels = l
	}
}

func WithScheduler(s Scheduler) func(*Exporter) {
	return func(e *Exporter) {
		e.scheduler = s
	}
}

func WithScheduledScrapers(ss ...*ScheduledScraper) func(*Exporter) {
	return func(e *Exporter) {
		e.scheduledscrapers = ss
	}
}

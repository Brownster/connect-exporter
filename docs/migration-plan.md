High-Level Plan

The migration will involve replacing the custom boilerplate (server, workers, metrics) with the monitoring-maas framework, while preserving the core business logic (feed fetching and parsing).
Component	Current Implementation (rss-exporter)	Target Implementation (Company Standard)
Project Structure	internal/exporter, internal/collectors	/collectors, /cmd
Main Entrypoint	main.go with signal handling, server setup	cmd/rss_exporter/main.go calling the maas framework
Worker Logic	Custom goroutine management in worker.go	maas.ScheduledScraper for each feed
Configuration	Custom flag parser in init()	kingpin via maas; config read in exporter.go
Metric Generation	Custom Prometheus collector in metrics.go	maas.NewMetric calls within a Scrape method
HTTP Fetching	Custom FetchFeedWithRetry in connectors	A new, custom maas.Connector for HTTP
Testing	httptest servers	Mock maas.Connector and testutil.CollectAndCompare
Step-by-Step Refactoring Guide
Step 1: Restructure the Project

Reorganize the files and directories to match the company standard.

    Create cmd/rss_exporter/: Move the existing main.go into this new directory.

    Rename internal/ to collectors/: This will be the new home for all scraping logic.

    Move Parser Logic: Move internal/collectors/scraper.go and aws_guid.go to collectors/parsers.go (or similar). The logic for selecting a provider scraper is valuable and should be kept.

    Remove Obsolete Files: Delete the following files, as their functionality will be replaced:

        internal/exporter/worker.go

        internal/exporter/metrics.go

        internal/exporter/exporter.go

    Consolidate Core Logic: The core scraping logic from internal/exporter/service.go will be moved into a new collectors/feed.go file in a later step.

Your new structure will look like this:

      
rss-exporter/
├── cmd/rss_exporter/
│   └── main.go
├── collectors/
│   ├── exporter.go         # (New file)
│   ├── feed.go             # (New file, contains logic from service.go)
│   ├── feed_test.go        # (New file)
│   ├── parsers.go          # (Contains logic from scraper.go)
│   └── testdata/           # (Existing test data)
├── Dockerfile
├── go.mod
└── ... (other files)

    

IGNORE_WHEN_COPYING_START
Use code with caution.
IGNORE_WHEN_COPYING_END


''Step 2: Create a Custom HTTP Connector

The monitoring-maas framework uses a maas.Connector interface. Since your exporter needs to make HTTP calls, you must create a new connector for this purpose.

    Create connectors/http.go:

          
    package connectors

    import (
        "github.com/4O4-Not-F0und/rss-exporter/internal/fetcher" // Reuse your existing logic
        "github.com/mmcdole/gofeed"
        "github.com/sirupsen/logrus"
    )

    // HTTPConnector implements maas.Connector for fetching RSS feeds.
    type HTTPConnector struct {
        Logger *logrus.Entry
    }

    // NewHTTPConnector creates a new HTTP connector.
    func NewHTTPConnector() *HTTPConnector {
        return &HTTPConnector{
            Logger: logrus.WithField("component", "http_connector"),
        }
    }

    // The query will be the URL string.
    type HTTPQuery struct {
        URL string
    }

    // Execute fetches the RSS feed.
    func (c *HTTPConnector) Execute(query interface{}) (interface{}, error) {
        httpQuery := query.(HTTPQuery)
        // Reuse your existing FetchFeedWithRetry logic here
        return connector.FetchFeedWithRetry(httpQuery.URL, c.Logger)
    }

        

    IGNORE_WHEN_COPYING_START

Use code with caution. Go
IGNORE_WHEN_COPYING_END

Update cmd/rss_exporter/main.go:

      
package main

import (
    "time"
    "github.com/getsentry/sentry-go"
    "github.com/4O4-Not-F0und/rss-exporter/collectors"
    "github.com/4O4-Not-F0und/rss-exporter/connectors" // Your new connector
    log "github.com/sirupsen/logrus"
)

func main() {
    // Instantiate your new HTTP connector
    e, err := collectors.NewRssExporter(connectors.NewHTTPConnector())

    if err != nil {
        log.Fatal(err)
    }

    e.Start()
    e.Serve()

    sentry.Flush(5 * time.Second)
}

    

IGNORE_WHEN_COPYING_START

    Use code with caution. Go
    IGNORE_WHEN_COPYING_END

Step 3: Refactor to Use maas.ScheduledScraper

This is the core of the refactoring. You will create one scheduled scraper for each service defined in config.yml.

    Create collectors/feed.go: This file will define a single, reusable collector for any feed.

          
    package collectors

    import (
        "time"
        "github.com/prometheus/client_golang/prometheus"
        maas "github.com/sabio-engineering-product/monitoring-maas"
        "gopkg.in/alecthomas/kingpin.v2"
        "github.com/mmcdole/gofeed"
        "github.com/4O4-Not-F0und/rss-exporter/connectors" // Your new connector
        "strings"
    )

    // NewFeedCollector creates a dynamically scheduled scraper for a single RSS feed.
    func NewFeedCollector(app *kingpin.Application, serviceConfig maas.ServiceFeed) *maas.ScheduledScraper {
        // Metric names are now simple suffixes. The service name becomes a prefix.
        // e.g., for service "aws", the metric will be "aws_service_status".
        // The service name is now a label.
        maas.WithDescription(app, "service_status", "Current service status", []string{"service", "customer", "state"})
        maas.WithDescription(app, "service_issue_info", "Details for active service issues", []string{"service", "customer", "service_name", "region", "title", "link", "guid"})
        
        return maas.NewScheduledScraper(
            serviceConfig.Name, // This name is used as a prefix for metrics
            NewFeedScraper(serviceConfig),
            maas.WithSchedule(maas.NewSchedule(
                maas.WithFrequency(time.Duration(serviceConfig.Interval)*time.Second),
            )),
        )
    }

    type FeedScraper struct {
        Config maas.ServiceFeed
        Parser Scraper // Re-use your parser interface from `parsers.go`
    }

    func NewFeedScraper(cfg maas.ServiceFeed) *FeedScraper {
        return &FeedScraper{
            Config: cfg,
            Parser: ScraperForService(cfg.Provider, cfg.Name),
        }
    }

    // Scrape contains the logic from your old `updateServiceStatus` function.
    func (s *FeedScraper) Scrape(c maas.Connector) ([]maas.Metric, error) {
        metrics := make([]maas.Metric, 0)
        
        feed, err := c.Execute(connectors.HTTPQuery{URL: s.Config.URL})
        if err != nil {
            // Let the framework handle logging of the error.
            // Just return it. maas will automatically increment a failure metric.
            return nil, err
        }
        
        gofeed := feed.(*gofeed.Feed)
        
        // ... (Your parsing logic from `updateServiceStatus` goes here) ...
        // state := "ok"
        // var activeItem *gofeed.Item
        // ...
        
        // After parsing, create metrics with maas.NewMetric
        for _, st := range []string{"ok", "service_issue", "outage"} {
            value := 0.0
            if state == st {
                value = 1.0
            }
            metrics = append(metrics, maas.NewMetric("service_status", prometheus.GaugeValue, value, 
                []string{s.Config.Name, s.Config.Customer, st}))
        }
        
        if activeItem != nil {
             // ... extract info ...
             metrics = append(metrics, maas.NewMetric("service_issue_info", prometheus.GaugeValue, 1, 
                []string{s.Config.Name, s.Config.Customer, svcName, region, title, link, guid}))
        }

        return metrics, nil
    }

        

    IGNORE_WHEN_COPYING_START

Use code with caution. Go
IGNORE_WHEN_COPYING_END

Create collectors/exporter.go: This file will now dynamically create collectors based on the config.

      
package collectors

import (
    maas "github.com/sabio-engineering-product/monitoring-maas"
    "gopkg.in/alecthomas/kingpin.v2"
)

// NewRssExporter is the main constructor.
func NewRssExporter(c maas.Connector, options ...func(*maas.Exporter)) (*maas.Exporter, error) {
    app := kingpin.New("rss_exporter", "Exporter for RSS/Atom status feeds.").DefaultEnvars()
    
    // The maas framework handles the -config.file flag. We can access the config here.
    config := maas.GetConfig() // Assuming maas provides a way to get the parsed config.
                               // If not, you may need to read it here manually.
                               
    scrapers := []*maas.ScheduledScraper{}
    for _, service := range config.Services {
        scrapers = append(scrapers, NewFeedCollector(app, service))
    }

    options = append(options, maas.WithScheduledScrapers(scrapers...))

    e, err := maas.NewExporter(app, c, options...)
    return e, err
}

    

IGNORE_WHEN_COPYING_START

    Use code with caution. Go
    IGNORE_WHEN_COPYING_END

    Note: The ability to dynamically add scrapers depends on the maas framework's flexibility. The approach above is a robust way to handle it.

Step 4: Update Testing

Refactor your tests to align with the new structure.

    Create connectors/http_mock.go:

          
    package connectors

    import "io"

    // MockHTTPConnector simulates HTTP requests for testing.
    type MockHTTPConnector struct {
        Responses map[string]string // Map URL to response content
    }

    func (c *MockHTTPConnector) Execute(query interface{}) (interface{}, error) {
        httpQuery := query.(HTTPQuery)
        content, ok := c.Responses[httpQuery.URL]
        if !ok {
            return nil, fmt.Errorf("no mock response for URL: %s", httpQuery.URL)
        }
        // Use the gofeed parser directly, just like the real connector.
        return gofeed.NewParser().Parse(strings.NewReader(content))
    }

        

    IGNORE_WHEN_COPYING_START

Use code with caution. Go
IGNORE_WHEN_COPYING_END

Update collectors/feed_test.go:

      
package collectors

import (
    "testing"
    "strings"
    "os"
    "github.com/stretchr/testify/suite"
    "github.com/4O4-Not-F0und/rss-exporter/connectors"
    maas "github.com/sabio-engineering-product/monitoring-maas"
    "github.com/prometheus/client_golang/prometheus/testutil"
)

type FeedTestSuite struct {
    suite.Suite
    Connector *connectors.MockHTTPConnector
    Exporter *maas.Exporter
}

func (s *FeedTestSuite) SetupTest() {
    s.Connector = &connectors.MockHTTPConnector{
        Responses: make(map[string]string),
    }

    // Load mock data
    awsData, _ := os.ReadFile("testdata/aws_outage.rss")
    s.Connector.Responses["http://mock.aws/feed"] = string(awsData)

    // Create exporter with mock connector and dynamic collectors
    app := kingpin.New("test", "")
    serviceCfg := maas.ServiceFeed{Name: "aws-test", URL: "http://mock.aws/feed", Interval: 300}
    
    // We test one collector at a time
    e, err := maas.NewExporter(app, s.Connector, 
        maas.WithScheduledScrapers(NewFeedCollector(app, serviceCfg)),
        maas.WithLabels(&maas.MockLabels{}),
    )
    s.NoError(err)
    s.Exporter = e
}

func (s *FeedTestSuite) TestAWSOutage() {
    s.Exporter.Start()
    
    expected := `
    # HELP aws-test_service_status Current service status
    # TYPE aws-test_service_status gauge
    aws-test_service_status{customer="",service="aws-test",state="ok"} 0
    aws-test_service_status{customer="",service="aws-test",state="outage"} 1
    aws-test_service_status{customer="",service="aws-test",state="service_issue"} 0
    `
    err := testutil.CollectAndCompare(s.Exporter, strings.NewReader(expected), "aws-test_service_status")
    s.NoError(err)
}

func TestFeedSuite(t *testing.T) {
    suite.Run(t, new(FeedTestSuite))
}

    

IGNORE_WHEN_COPYING_START

    Use code with caution. Go
    IGNORE_WHEN_COPYING_END

Step 5: Final Touches

    Update Dockerfile: Your Dockerfile is already using a multi-stage build, which is great. Update the final binary name to rss_exporter if needed, and ensure the config.yml is correctly copied. The company standard seems to be a single-stage build with a minimal image, so you might adjust to:

          
    # This aligns better with the linux-exporter example
    FROM registry-maas.maas.services.sabio.co.uk/docker/busybox-glibc:1.0.0
    COPY rss_exporter /
    # Note: config is now handled via a flag, not a file in the image
    CMD ["/rss_exporter", "-config.file=/config/config.yml"]

        

    IGNORE_WHEN_COPYING_START

    Use code with caution. Dockerfile
    IGNORE_WHEN_COPYING_END

    Update docker-compose.yml: Adjust the volume mount path for the config file.

    Update Documentation: Update the README.md and docs/ to reflect the new standardized command-line flag (-config.file) and the monitoring-maas framework.

By following this plan, you will successfully align your rss-exporter with the company's robust and standardized monitoring framework, making it a valuable and maintainable asset for the team.

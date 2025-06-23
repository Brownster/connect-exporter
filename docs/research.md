Thinking Thoughts (experimental)
Auto

Expand to view model thoughts
chevron_right
Thinking Thoughts (experimental)
Auto

Expand to view model thoughts
chevron_right

Of course. This is an excellent initiative. Migrating your personal rss-exporter project to the company's monitoring-maas framework will make it standardized, easier to maintain, and instantly familiar to other developers on your team.

Here is a detailed, step-by-step plan to refactor the rss-exporter.
Migration Plan: rss-exporter to Company Standard
Objective

Refactor the existing rss-exporter to use the monitoring-maas framework, aligning its structure, configuration, and testing with company standards as exemplified by monitoring-linux-exporter and monitoring-audiocodesbc-exporter.
Current State Analysis

The rss-exporter is a well-structured Go application.

    Strengths: Clear separation of concerns (connectors, collectors, exporter), robust provider logic, and good documentation.

    Deviations from Standard:

        It uses a custom worker/goroutine management system instead of maas.ScheduledScraper.

        It implements its own Prometheus collector (metrics.go) instead of using maas.NewMetric.

        It has its own configuration loader and flag parser instead of using kingpin via maas.

        The project structure (internal/) differs from the company's /collectors standard.


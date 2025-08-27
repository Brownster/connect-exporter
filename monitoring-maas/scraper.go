package maas

import "github.com/prometheus/client_golang/prometheus"

type Metric struct {
	name      string
	valueType prometheus.ValueType
	value     float64
	labels    []string
}

func NewMetric(n string, t prometheus.ValueType, v float64, l []string) Metric {
	return Metric{
		name:      n,
		valueType: t,
		value:     v,
		labels:    l,
	}
}

type Scraper interface {
	Scrape(c Connector) ([]Metric, error)
}

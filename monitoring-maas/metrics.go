package maas

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	sync.RWMutex
	metrics []prometheus.Metric
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Put(met []prometheus.Metric) {
	m.Lock()
	defer m.Unlock()
	m.metrics = met
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.RLock()
	defer m.RUnlock()

	for _, met := range m.metrics {
		ch <- met
	}
}

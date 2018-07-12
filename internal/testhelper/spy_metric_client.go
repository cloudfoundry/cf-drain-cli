package testhelper

import (
	"sync"

	"code.cloudfoundry.org/go-loggregator/pulseemitter"
)

type SpyMetricClient struct {
	metrics map[string]*SpyMetric
}

func NewMetricClient() *SpyMetricClient {
	return &SpyMetricClient{
		metrics: make(map[string]*SpyMetric),
	}
}

func (s *SpyMetricClient) NewCounterMetric(name string, opts ...pulseemitter.MetricOption) pulseemitter.CounterMetric {
	m := &SpyMetric{}
	s.metrics[name] = m

	return m
}

func (s *SpyMetricClient) NewGaugeMetric(name, unit string, opts ...pulseemitter.MetricOption) pulseemitter.GaugeMetric {
	m := &SpyMetric{}
	s.metrics[name] = m

	return m
}

func (s *SpyMetricClient) GetMetric(name string) *SpyMetric {
	return s.metrics[name]
}

type SpyMetric struct {
	mu         sync.Mutex
	delta      uint64
	gaugeValue float64
}

func (s *SpyMetric) Increment(c uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delta += c
}

func (s *SpyMetric) Emit(c pulseemitter.LogClient) {}

func (s *SpyMetric) Set(c float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gaugeValue = c
}

func (s *SpyMetric) Delta() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.delta
}

func (s *SpyMetric) GaugeValue() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gaugeValue
}

package testhelper

import (
	"sync"

	"code.cloudfoundry.org/go-loggregator/pulseemitter"
)

type TestMetric interface {
	Delta() uint64
	GaugeValue() float64
}

type SpyMetricClient struct {
	metrics map[string]TestMetric
}

func NewMetricClient() *SpyMetricClient {
	return &SpyMetricClient{
		metrics: make(map[string]TestMetric),
	}
}

func (s *SpyMetricClient) NewCounterMetric(
	name string,
	opts ...pulseemitter.MetricOption,
) pulseemitter.CounterMetric {
	m := &SpyMetric{}
	s.metrics[name] = m

	return m
}

func (s *SpyMetricClient) GetMetric(name string) TestMetric {
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

type SpyGaugeMetric struct {
	SpyMetric
}

func (s *SpyGaugeMetric) Increment(c float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gaugeValue += c
}

func (s *SpyGaugeMetric) Decrement(c float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gaugeValue -= c
}

func (s *SpyGaugeMetric) Set(c float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gaugeValue = c
}

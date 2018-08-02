package stream

import (
	"context"
	"log"
	"sync"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	streamaggregator "code.cloudfoundry.org/go-stream-aggregator"
)

// GatewayClient is the interface used to open a new stream with the logs
// provider gateway.
type GatewayClient interface {
	Stream(ctx context.Context, req *loggregator_v2.EgressBatchRequest) loggregator.EnvelopeStream
}

// Aggregator manages converging multiple streams into one.
type Aggregator struct {
	sync.Mutex
	client    GatewayClient
	agg       *streamaggregator.StreamAggregator
	streamIDs []string
	log       *log.Logger
	shardID   string
}

// NewAggregator configures and returns a new Aggregator.
func NewAggregator(c GatewayClient, shardID string, l *log.Logger) *Aggregator {
	return &Aggregator{
		client:  c,
		agg:     streamaggregator.New(streamaggregator.WithLogger(l)),
		log:     l,
		shardID: shardID,
	}
}

// Consume returns a channel from which a client can read from the aggregated
// stream.
func (a *Aggregator) Consume() <-chan interface{} {
	return a.agg.Consume(
		context.Background(),
		nil,
		streamaggregator.WithConsumeChannelLength(10000),
	)
}

// Add adds a new source ID to the aggregator. This is called by the
// orchestrator.
func (a *Aggregator) Add(id string) {
	a.log.Printf("adding producer for %s", id)

	a.Lock()
	defer a.Unlock()

	producer := &streamProducer{id, a.shardID, a.client, a.log}
	a.agg.AddProducer(id, producer)
	a.streamIDs = append(a.streamIDs, id)
}

// Remove removes an existing source ID from the aggregator. This is called by the
// orchestrator.
func (a *Aggregator) Remove(id string) {
	a.log.Printf("removing producer for %s", id)

	a.Lock()
	defer a.Unlock()
	a.agg.RemoveProducer(id)

	var newIDs []string
	for _, g := range a.streamIDs {
		if g == id {
			continue
		}
		newIDs = append(newIDs, g)
	}
	a.streamIDs = newIDs
}

// List returns the current list of stream IDs being aggregated. This is
// called by the orchestrator.
func (a *Aggregator) List() []interface{} {
	a.Lock()
	defer a.Unlock()

	var taskNames []interface{}
	for _, k := range a.streamIDs {
		taskNames = append(taskNames, k)
	}

	return taskNames
}

type streamProducer struct {
	guid    string
	shardID string
	client  GatewayClient
	log     *log.Logger
}

func (s *streamProducer) Produce(ctx context.Context, _ interface{}, c chan<- interface{}) {
	stream := s.client.Stream(ctx, &loggregator_v2.EgressBatchRequest{
		ShardId:   s.shardID,
		Selectors: selectorsForSource(s.guid),
	})

	for {
		envs := stream()
		if envs == nil {
			s.log.Printf("stream closed for %s", s.guid)
			return
		}

		for _, e := range envs {
			c <- e
		}
	}
}

func selectorsForSource(id string) []*loggregator_v2.Selector {
	return []*loggregator_v2.Selector{
		{SourceId: id, Message: &loggregator_v2.Selector_Log{Log: &loggregator_v2.LogSelector{}}},
		{SourceId: id, Message: &loggregator_v2.Selector_Gauge{Gauge: &loggregator_v2.GaugeSelector{}}},
		{SourceId: id, Message: &loggregator_v2.Selector_Counter{Counter: &loggregator_v2.CounterSelector{}}},
	}
}

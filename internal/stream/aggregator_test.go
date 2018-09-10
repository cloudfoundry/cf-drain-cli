package stream_test

import (
	"context"
	"log"
	"sync"

	"code.cloudfoundry.org/cf-drain-cli/internal/stream"
	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregator", func() {
	var (
		gatewayClient *spyGatewayClient
		logger        *log.Logger
	)

	BeforeEach(func() {
		gatewayClient = newSpyGatewayClient()
		logger = log.New(GinkgoWriter, "", log.LstdFlags)
	})

	It("adds a stream to be consumed", func() {
		agg := stream.NewAggregator(gatewayClient, "shard-id", logger)
		agg.Add(stream.Resource{
			GUID: "source-id-1",
			Name: "source-1",
		})

		_ = agg.Consume()

		Eventually(gatewayClient.streamReqs).Should(HaveLen(1))

		r := gatewayClient.streamReqs()[0]
		Expect(r.req).To(Equal(&loggregator_v2.EgressBatchRequest{
			ShardId: "shard-id",
			Selectors: []*loggregator_v2.Selector{
				{SourceId: "source-id-1", Message: &loggregator_v2.Selector_Log{Log: &loggregator_v2.LogSelector{}}},
				{SourceId: "source-id-1", Message: &loggregator_v2.Selector_Gauge{Gauge: &loggregator_v2.GaugeSelector{}}},
				{SourceId: "source-id-1", Message: &loggregator_v2.Selector_Counter{Counter: &loggregator_v2.CounterSelector{}}},
			},
		}))
	})

	It("removes a stream already being consumed", func() {
		agg := stream.NewAggregator(gatewayClient, "shard-id", logger)
		agg.Add(stream.Resource{
			GUID: "source-id-1",
			Name: "source-1",
		})
		_ = agg.Consume()
		Eventually(gatewayClient.streamReqs).Should(HaveLen(1))

		agg.Remove("source-id-1")

		r := gatewayClient.streamReqs()[0]
		Eventually(r.ctx.Done).Should(BeClosed())
	})

	It("lists source IDs being aggregated", func() {
		agg := stream.NewAggregator(gatewayClient, "shard-id", logger)
		agg.Add(stream.Resource{
			GUID: "source-id-1",
			Name: "source-1",
		})
		agg.Add(stream.Resource{
			GUID: "source-id-2",
			Name: "source-2",
		})

		Expect(agg.List()).To(Equal([]interface{}{
			stream.Resource{
				GUID: "source-id-1",
				Name: "source-1",
			},
			stream.Resource{
				GUID: "source-id-2",
				Name: "source-2",
			},
		}))
	})

	It("forwards produced logs to the consumer", func() {
		agg := stream.NewAggregator(gatewayClient, "shard-id", logger)
		agg.Add(stream.Resource{
			GUID: "source-id-1",
			Name: "source-1",
		})

		c := agg.Consume()
		Eventually(c).Should(Receive())
	})
})

type streamReq struct {
	ctx context.Context
	req *loggregator_v2.EgressBatchRequest
}

type spyGatewayClient struct {
	mu          sync.Mutex
	_streamReqs []streamReq
}

func newSpyGatewayClient() *spyGatewayClient {
	return &spyGatewayClient{}
}

func (s *spyGatewayClient) Stream(ctx context.Context, req *loggregator_v2.EgressBatchRequest) loggregator.EnvelopeStream {
	s.mu.Lock()
	defer s.mu.Unlock()

	s._streamReqs = append(s._streamReqs, streamReq{
		ctx: ctx,
		req: req,
	})

	return loggregator.EnvelopeStream(func() []*loggregator_v2.Envelope {
		return []*loggregator_v2.Envelope{
			{SourceId: "soruce-id-1"},
		}
	})
}

func (s *spyGatewayClient) streamReqs() []streamReq {
	s.mu.Lock()
	defer s.mu.Unlock()

	reqs := make([]streamReq, len(s._streamReqs))
	copy(reqs, s._streamReqs)

	return reqs
}

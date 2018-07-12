package groupmanager_test

import (
	"context"
	"sync"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/groupmanager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		stubGroupProvider *stubGroupProvider
		spyGroupUpdater   *spyGroupUpdater
		spyMetrics        *spyMetrics
		t                 chan time.Time
	)

	BeforeEach(func() {
		spyGroupUpdater = newSpyGroupUpdater()
		spyMetrics = newSpyMetrics()
		stubGroupProvider = newStubGroupProvider()
		t = make(chan time.Time, 1)

		stubGroupProvider.sourceIDs = []string{
			"source-id-1",
			"source-id-2",
		}

		go groupmanager.Start(
			"group-name",
			t,
			stubGroupProvider,
			spyGroupUpdater,
			groupmanager.WithMetrics(spyMetrics),
		)
	})

	It("fetches meta and adds to the group", func() {
		t <- time.Now()

		Eventually(spyGroupUpdater.AddRequests).Should(ConsistOf(
			addRequest{name: "group-name", sourceIDs: []string{"source-id-1"}},
			addRequest{name: "group-name", sourceIDs: []string{"source-id-2"}},
			addRequest{name: "group-name", sourceIDs: []string{"source-id-1"}},
			addRequest{name: "group-name", sourceIDs: []string{"source-id-2"}},
		))
	})

	It("immediately fetches", func() {
		Eventually(spyGroupUpdater.AddRequests).Should(ConsistOf(
			addRequest{name: "group-name", sourceIDs: []string{"source-id-1"}},
			addRequest{name: "group-name", sourceIDs: []string{"source-id-2"}},
		))
	})

	It("sets the source id len as a metric", func() {
		Eventually(spyMetrics.value).Should(BeEquivalentTo(2))
	})
})

type spyMetrics struct {
	mu     sync.Mutex
	value_ float64
}

func newSpyMetrics() *spyMetrics {
	return &spyMetrics{}
}

func (s *spyMetrics) NewGauge(name string) func(float64) {
	return func(v float64) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.value_ = v
	}
}

func (s *spyMetrics) value() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value_
}

func newStubGroupProvider() *stubGroupProvider {
	return &stubGroupProvider{}
}

type stubGroupProvider struct {
	sourceIDs []string
}

func (s *stubGroupProvider) SourceIDs() []string {
	return s.sourceIDs
}

func newSpyGroupUpdater() *spyGroupUpdater {
	return &spyGroupUpdater{}
}

type spyGroupUpdater struct {
	mu          sync.Mutex
	addRequests []addRequest
}

func (s *spyGroupUpdater) SetShardGroup(ctx context.Context, name string, sourceIDs ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addRequests = append(s.addRequests, addRequest{name: name, sourceIDs: sourceIDs})
	return nil
}

func (s *spyGroupUpdater) AddRequests() []addRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := make([]addRequest, len(s.addRequests))
	copy(r, s.addRequests)

	return r
}

type addRequest struct {
	name      string
	sourceIDs []string
}

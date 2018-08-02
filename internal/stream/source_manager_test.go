package stream_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/go-orchestrator"

	"code.cloudfoundry.org/cf-drain-cli/internal/stream"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Source Manager", func() {
	var (
		o *spyOrchestrator
		s *stubSourceProvider
	)

	BeforeEach(func() {
		o = newFakeOrchestrator()
		s = &stubSourceProvider{}
	})

	It("Adds sourceIDs to the orchestrator", func() {
		s.sourceIDs = []string{"source-id"}
		sm := stream.NewSourceManager(s, o, time.Second)

		go sm.Start()

		var tasks []orchestrator.Task
		Eventually(o.tasks).Should(Receive(&tasks))

		Expect(tasks[0].Name).To(Equal("source-id"))
		Expect(tasks[0].Instances).To(Equal(1))

		Eventually(o.nextTerm).Should(Receive())
	})

	It("updates the tasks after a given interval", func() {
		s.sourceIDs = []string{"source-id"}
		sm := stream.NewSourceManager(s, o, 250*time.Millisecond)

		go sm.Start()

		var tasks []orchestrator.Task
		Eventually(o.tasks).Should(Receive(&tasks))

		Expect(tasks[0].Name).To(Equal("source-id"))
		Expect(tasks[0].Instances).To(Equal(1))
		Eventually(o.nextTerm).Should(Receive())

		Eventually(o.tasks).Should(Receive(&tasks))

		Expect(tasks[0].Name).To(Equal("source-id"))
		Expect(tasks[0].Instances).To(Equal(1))
		Eventually(o.nextTerm).Should(Receive())
	})

	It("does not updated if the sourceID provider returns an error", func() {
		s.sourceIDs = []string{"source-id"}
		sm := stream.NewSourceManager(s, o, 250*time.Millisecond)

		go sm.Start()

		var tasks []orchestrator.Task
		Eventually(o.tasks).Should(Receive(&tasks))

		Expect(tasks[0].Name).To(Equal("source-id"))
		Expect(tasks[0].Instances).To(Equal(1))
		Eventually(o.nextTerm).Should(Receive())

		s.sourceIDs = nil
		s.err = errors.New("source ID error")

		Consistently(o.nextTerm, .25).ShouldNot(Receive())
	})
})

type spyOrchestrator struct {
	tasks    chan []orchestrator.Task
	nextTerm chan bool
}

func newFakeOrchestrator() *spyOrchestrator {
	return &spyOrchestrator{
		nextTerm: make(chan bool, 100),
		tasks:    make(chan []orchestrator.Task, 100),
	}
}

func (o *spyOrchestrator) NextTerm(ctx context.Context) {
	o.nextTerm <- true
}

func (o *spyOrchestrator) UpdateTasks(t []orchestrator.Task) {
	o.tasks <- t
}

type stubSourceProvider struct {
	sourceIDs []string
	err       error
}

func (s *stubSourceProvider) SourceIDs() ([]string, error) {
	return s.sourceIDs, s.err
}

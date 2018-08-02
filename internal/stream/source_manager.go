package stream

import (
	"context"
	"time"

	orchestrator "code.cloudfoundry.org/go-orchestrator"
)

type SourceIDProvider interface {
	SourceIDs() ([]string, error)
}

type Orchestrator interface {
	NextTerm(ctx context.Context)
	UpdateTasks([]orchestrator.Task)
}

type SourceManager struct {
	s        SourceIDProvider
	o        Orchestrator
	interval time.Duration
}

func NewSourceManager(s SourceIDProvider, o Orchestrator, interval time.Duration) *SourceManager {
	return &SourceManager{
		s:        s,
		o:        o,
		interval: interval,
	}
}

func (s *SourceManager) Start() {
	s.updateSources()

	t := time.NewTicker(s.interval)
	for range t.C {
		s.updateSources()
	}
}

func (s *SourceManager) updateSources() {
	sourceIDs, err := s.s.SourceIDs()
	if err != nil {
		return
	}

	tasks := sourceIDsToTasks(sourceIDs)
	s.o.UpdateTasks(tasks)
	s.o.NextTerm(context.Background())
}

func sourceIDsToTasks(sids []string) []orchestrator.Task {
	var tasks []orchestrator.Task
	for _, s := range sids {
		tasks = append(tasks, orchestrator.Task{Name: s, Instances: 1})
	}

	return tasks
}

type Communicator struct{}

func (a Communicator) List(ctx context.Context, worker interface{}) ([]interface{}, error) {
	sa := worker.(*Aggregator)
	return sa.List(), nil
}

func (a Communicator) Add(ctx context.Context, worker, task interface{}) error {
	sa := worker.(*Aggregator)
	guid := task.(string)
	sa.Add(guid)

	return nil
}

func (a Communicator) Remove(ctx context.Context, worker, task interface{}) error {
	sa := worker.(*Aggregator)
	if task != nil {
		sa.Remove(task.(string))
	}

	return nil
}

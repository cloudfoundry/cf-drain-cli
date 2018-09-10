package stream

import (
	"context"
	"time"

	orchestrator "code.cloudfoundry.org/go-orchestrator"
)

type SourceProvider interface {
	Resources() ([]Resource, error)
}

type Orchestrator interface {
	NextTerm(ctx context.Context)
	UpdateTasks([]orchestrator.Task)
}

type SourceManager struct {
	s        SourceProvider
	o        Orchestrator
	interval time.Duration
}

func NewSourceManager(s SourceProvider, o Orchestrator, interval time.Duration) *SourceManager {
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
	resources, err := s.s.Resources()
	if err != nil {
		return
	}

	tasks := resourcesToTasks(resources)
	s.o.UpdateTasks(tasks)
	s.o.NextTerm(context.Background())
}

func resourcesToTasks(resources []Resource) []orchestrator.Task {
	var tasks []orchestrator.Task
	for _, r := range resources {
		tasks = append(tasks, orchestrator.Task{Name: r, Instances: 1})
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
	r := task.(Resource)

	sa.Add(r)

	return nil
}

func (a Communicator) Remove(ctx context.Context, worker, task interface{}) error {
	sa := worker.(*Aggregator)
	if task != nil {
		sa.Remove(task.(Resource).GUID)
	}

	return nil
}

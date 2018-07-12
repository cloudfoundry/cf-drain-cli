package groupmanager

import (
	"context"
	"log"
	"time"
)

// Manager syncs the source IDs from the GroupProvider with the GroupUpdater.
// It does so at the configured interval.
type Manager struct {
	groupName      string
	ticker         <-chan time.Time
	gp             GroupProvider
	gu             GroupUpdater
	sourceIDMetric func(float64)
}

// Metrics registers Gauge metrics.
type metrics interface {
	// NewGauge returns a function to set the value for the given metric.
	NewGauge(name string) func(value float64)
}

// GroupProvider returns the desired SourceIDs.
type GroupProvider interface {
	SourceIDs() []string
}

// GroupProviderFunc upgrades a regular function into a GroupProvider.
type GroupProviderFunc func() []string

// SourceIDs implements GroupProvider.
func (f GroupProviderFunc) SourceIDs() []string {
	return f()
}

// GroupUpdater is used to add (and keep alive) the source IDs for a group.
type GroupUpdater interface {
	// SetShardGroup adds source IDs to the LogCache sub-groups.
	SetShardGroup(ctx context.Context, name string, sourceIDs ...string) error
}

// Start creates and starts a Manager.
func Start(
	groupName string,
	ticker <-chan time.Time,
	gp GroupProvider,
	gu GroupUpdater,
	opts ...GroupManagerOption,
) {
	m := &Manager{
		groupName:      groupName,
		ticker:         ticker,
		gp:             gp,
		gu:             gu,
		sourceIDMetric: func(float64) {},
	}

	for _, o := range opts {
		o(m)
	}

	m.run()
}

type GroupManagerOption func(*Manager)

func WithMetrics(metrics metrics) GroupManagerOption {
	return func(m *Manager) {
		m.sourceIDMetric = metrics.NewGauge("SourceIDs")
	}
}

func (m *Manager) run() {
	m.updateSourceIDs(m.gp.SourceIDs())

	for range m.ticker {
		m.updateSourceIDs(m.gp.SourceIDs())
	}
}

func (m *Manager) updateSourceIDs(sourceIDs []string) {
	start := time.Now()
	for _, sid := range sourceIDs {
		if err := m.gu.SetShardGroup(context.Background(), m.groupName, sid); err != nil {
			log.Printf("failed to set shard group: %s", err)
		}
	}
	m.sourceIDMetric(float64(len(sourceIDs)))
	log.Printf("Setting %d source ids took %s", len(sourceIDs), time.Since(start).String())
}

package egress

import (
	"log"

	logcache "code.cloudfoundry.org/go-log-cache"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

type metrics interface {
	NewCounter(string) func(uint64)
}

func NewVisitor(w Writer, metrics metrics, log *log.Logger) logcache.Visitor {
	ingress := metrics.NewCounter("Ingress")
	syslog := metrics.NewCounter("Egress")
	dropped := metrics.NewCounter("Dropped")

	return func(envs []*loggregator_v2.Envelope) bool {
		var droppedCount int
		for _, e := range envs {
			if w.Write(e) != nil {
				log.Printf("failed to write envelope: %s", e)
				droppedCount++
			}
		}

		sentCount := len(envs) - droppedCount
		ingress(uint64(len(envs)))
		dropped(uint64(droppedCount))
		syslog(uint64(sentCount))

		log.Printf("Wrote %d envelopes to syslog!", sentCount)
		return true
	}
}

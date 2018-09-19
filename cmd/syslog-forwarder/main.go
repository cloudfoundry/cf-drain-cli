package main

import (
	"expvar"
	"log"
	"net/http"
	"os"

	"code.cloudfoundry.org/cf-drain-cli/internal/egress"
	"code.cloudfoundry.org/cf-drain-cli/internal/stream"
	envstruct "code.cloudfoundry.org/go-envstruct"
	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	orchestrator "code.cloudfoundry.org/go-orchestrator"
	"code.cloudfoundry.org/loggregator-tools/log-cache-forwarders/pkg/expvarfilter"
	"code.cloudfoundry.org/loggregator-tools/log-cache-forwarders/pkg/metrics"
)

func main() {
	l := log.New(os.Stderr, "[Syslog-Forwarder] ", log.LstdFlags)
	l.Println("Starting Syslog Forwarder...")
	defer l.Println("Closing Syslog Forwarder...")

	cfg := LoadConfig()
	envstruct.WriteReport(&cfg)

	m := startMetricsEmit(l)
	success := m.NewCounter("EgressSuccess")
	failure := m.NewCounter("EgressFailure")

	client := loggregator.NewRLPGatewayClient(cfg.Vcap.RLPAddr,
		loggregator.WithRLPGatewayClientLogger(l),
	)

	streamAggregator := stream.NewAggregator(client, cfg.ShardID, l)
	o := createOrchestrator(streamAggregator)

	excludeSelf := func(sourceID string) bool { return sourceID == cfg.Vcap.AppID }
	sm := stream.NewSourceManager(
		stream.NewSingleOrSpaceProvider(
			cfg.SourceID,
			cfg.Vcap.API,
			cfg.Vcap.SpaceGUID,
			cfg.IncludeServices,
			stream.WithSourceProviderSpaceExcludeFilter(excludeSelf),
		),
		o,
		cfg.UpdateInterval,
	)
	go sm.Start()

	envs := streamAggregator.Consume()
	w := createSyslogWriter(cfg, l)
	for e := range envs {
		err := w.Write(e.(*loggregator_v2.Envelope))
		if err != nil {
			l.Printf("error writing envelope to syslog: %s", err)
			failure(1)
			continue
		}
		success(1)
	}
}

func createOrchestrator(s *stream.Aggregator) *orchestrator.Orchestrator {
	o := orchestrator.New(
		stream.Communicator{},
	)
	o.AddWorker(s)
	return o
}

func createSyslogWriter(cfg Config, log *log.Logger) egress.WriteCloser {
	netConf := egress.NetworkConfig{
		Keepalive:      cfg.KeepAlive,
		DialTimeout:    cfg.DialTimeout,
		WriteTimeout:   cfg.IOTimeout,
		SkipCertVerify: cfg.SkipCertVerify,
	}
	return egress.NewWriter(cfg.SourceHostname, cfg.SyslogURL, netConf, log)
}

func startMetricsEmit(log *log.Logger) *metrics.Metrics {
	m := metrics.New(expvar.NewMap("SyslogForwarder"))

	mh := expvarfilter.NewHandler(expvar.Handler(), []string{"SyslogForwarder"})
	go func() {
		// health endpoints (expvar)
		log.Printf("Health: %s", http.ListenAndServe(":"+os.Getenv("PORT"), mh))
	}()

	return m
}

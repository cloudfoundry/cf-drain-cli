package main

import (
	"context"
	"expvar"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/egress"
	envstruct "code.cloudfoundry.org/go-envstruct"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/loggregator-tools/log-cache-forwarders/pkg/expvarfilter"
	"code.cloudfoundry.org/loggregator-tools/log-cache-forwarders/pkg/metrics"
)

func main() {
	log := log.New(os.Stderr, "[Syslog-Forwarder] ", log.LstdFlags)
	log.Println("Starting Syslog Forwarder...")
	defer log.Println("Closing Syslog Forwarder...")

	rand.Seed(time.Now().UnixNano())

	cfg := LoadConfig()
	envstruct.WriteReport(&cfg)

	m := startMetricsEmit(log)

	groupClient := logcache.NewShardGroupReaderClient(
		cfg.LogCacheHost,
	)

	logcache.Walk(
		context.Background(),
		cfg.GroupName,
		egress.NewVisitor(createSyslogWriter(cfg, log), m, log),
		groupClient.BuildReader(rand.Uint64()),
		logcache.WithWalkStartTime(time.Now()),
		logcache.WithWalkBackoff(logcache.NewAlwaysRetryBackoff(250*time.Millisecond)),
		logcache.WithWalkLimit(1000),
		logcache.WithWalkLogger(log),
	)
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

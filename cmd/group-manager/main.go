package main

import (
	"log"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/groupmanager"
	logcache "code.cloudfoundry.org/go-log-cache"
)

func main() {
	log.Println("Starting Group manager...")
	defer log.Println("Closing Group manager...")
	cfg := loadConfig()

	groupClient := logcache.NewShardGroupReaderClient(
		cfg.LogCacheHost,
	)

	var p groupmanager.GroupProviderFunc
	p = func() []string {
		return []string{cfg.SourceID}
	}

	t := time.NewTicker(cfg.UpdateInterval)
	groupmanager.Start(
		cfg.GroupName,
		t.C,
		p,
		groupClient,
	)
}

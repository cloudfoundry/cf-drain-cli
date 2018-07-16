package main

import (
	"log"
	"net/http"
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

	var p groupmanager.GroupProvider
	p = groupmanager.GroupProviderFunc(func() []string {
		return []string{cfg.SourceID}
	})

	if cfg.SourceID == "" {
		p = groupmanager.Space(
			http.DefaultClient,
			cfg.VCap.API,
			cfg.VCap.SpaceGUID,
		)
	}

	t := time.NewTicker(cfg.UpdateInterval)
	groupmanager.Start(
		cfg.GroupName,
		t.C,
		p,
		groupClient,
	)
}

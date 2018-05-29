package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"github.com/cloudfoundry-incubator/uaago"
)

func main() {
	log := log.New(os.Stderr, "", log.LstdFlags)
	log.Printf("starting space drain...")
	defer log.Printf("space drain closing...")

	cfg := loadConfig()

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipCertVerify,
			},
		},
	}

	uaaClient, err := uaago.NewClient(cfg.UAAAddr)
	if err != nil {
		log.Fatalf("Failed to create UAA Client: %s", err)
	}

	var restager *cloudcontroller.Restager

	saveAndRestager := cloudcontroller.SaveAndRestagerFunc(func(rt string) {
		restager.SaveAndRestage(rt)
	})

	tokenManager := cloudcontroller.NewTokenManager(
		uaaClient,
		cfg.ClientID,
		cfg.RefreshToken,
		cfg.VCAPApplication.ID,
		cfg.SkipCertVerify,
		log,
	)

	curler := cloudcontroller.NewHTTPCurlClient(cfg.APIAddr, httpClient, tokenManager, saveAndRestager)
	restager = cloudcontroller.NewRestager(
		cfg.VCAPApplication.ID,
		curler,
		log,
	)

	drainLister := drain.NewServiceDrainLister(curler)
	drainCreator := cloudcontroller.NewCreateDrainClient(curler)
	drainBinder := cloudcontroller.NewBindDrainClient(curler)
	appLister := cloudcontroller.NewAppListerClient(curler)

	createAndBind(drainLister, drainCreator, drainBinder, appLister, cfg)
	for range time.Tick(time.Minute) {
		createAndBind(drainLister, drainCreator, drainBinder, appLister, cfg)
	}
}

func createAndBind(
	drainLister *drain.ServiceDrainLister,
	drainCreator *cloudcontroller.CreateDrainClient,
	drainBinder *cloudcontroller.BindDrainClient,
	appLister *cloudcontroller.AppListerClient,
	cfg Config,
) {
	drains, err := drainLister.Drains(cfg.SpaceID)
	if err != nil {
		log.Printf("failed to fetch drains: %s", err)
		return
	}

	drain, ok := hasDrain(cfg.DrainName, drains)
	if !ok {
		log.Printf("creating %s drain...", cfg.DrainName)
		if err := drainCreator.CreateDrain(
			cfg.DrainName,
			cfg.DrainURL,
			cfg.SpaceID,
			cfg.DrainType,
		); err != nil {
			log.Printf("failed to create drain: %s", err)
			return
		}
		log.Printf("created %s drain", cfg.DrainName)

		// go again so that ListDrains can find it and get its guid.
		createAndBind(drainLister, drainCreator, drainBinder, appLister, cfg)
		return
	}

	apps, err := appLister.ListApps(cfg.SpaceID)
	if err != nil {
		log.Printf("failed to list apps: %s", err)
		return
	}

	log.Printf("binding %d apps to drain...", len(apps))
	for _, app := range apps {
		if containsApp(app.Guid, drain.AppGuids) {
			return
		}

		if err := drainBinder.BindDrain(app.Guid, drain.Guid); err != nil {
			log.Printf("failed to bind %s to drain: %s", app.Guid, err)
			return
		}
		drain.AppGuids = append(drain.AppGuids, app.Guid)
	}
	log.Printf("done binding apps to drain.")
}

func containsApp(appGuid string, guids []string) bool {
	for _, g := range guids {
		if g == appGuid {
			return true
		}
	}

	return false
}

func hasDrain(name string, drains []drain.Drain) (drain.Drain, bool) {
	for _, drain := range drains {
		if drain.Name == name {
			return drain, true
		}
	}

	return drain.Drain{}, false
}

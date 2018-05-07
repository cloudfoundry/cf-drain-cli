package drain

import (
	"log"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

type ApplicationDrainLister struct {
	appLister   AppLister
	envProvider EnvProvider
}

type AppLister interface {
	ListApps(spaceGuid string) ([]cloudcontroller.App, error)
}

type EnvProvider interface {
	EnvVars(appGuid string) (map[string]string, error)
}

func NewApplicationDrainLister(appLister AppLister, envProvider EnvProvider) ApplicationDrainLister {
	return ApplicationDrainLister{
		appLister:   appLister,
		envProvider: envProvider,
	}
}

func (dl ApplicationDrainLister) Drains(spaceGUID string) ([]Drain, error) {
	spaceApps, _ := dl.appLister.ListApps(spaceGUID)
	guids, names, apps := dl.appMetadata(spaceApps)

	var drains []Drain
	for _, app := range spaceApps {
		envs, err := dl.envProvider.EnvVars(app.Guid)
		if err != nil {
			return nil, err
		}

		drainScope, ok := envs["DRAIN_SCOPE"]
		if !ok {
			continue
		}

		switch drainScope {
		case "space":
			drains = append(drains, Drain{
				Name:        app.Name,
				Guid:        app.Guid,
				Apps:        names,
				AppGuids:    guids,
				Type:        envs["DRAIN_TYPE"],
				DrainURL:    envs["DRAIN_URL"],
				AdapterType: "application",
			})
		case "single":
			sourceID, ok := envs["SOURCE_ID"]
			if !ok {
				log.Printf("failed to fetch environment variable SOURCE_ID for %s", app.Name)
				continue
			}
			name, ok := apps[sourceID]
			if !ok {
				log.Printf("something went very wrong: failed to retrieve app name for %s", sourceID)
				continue
			}

			drains = append(drains, Drain{
				Name:        app.Name,
				Guid:        app.Guid,
				Apps:        []string{name},
				AppGuids:    []string{sourceID},
				Type:        envs["DRAIN_TYPE"],
				DrainURL:    envs["SYSLOG_URL"],
				AdapterType: "application",
			})
		}
	}
	return drains, nil
}

func (dl ApplicationDrainLister) appMetadata(apps []cloudcontroller.App) (guids []string, names []string, spaceApps map[string]string) {
	spaceApps = make(map[string]string)
	for _, app := range apps {
		spaceApps[app.Guid] = app.Name
		guids = append(guids, app.Guid)
		names = append(names, app.Name)
	}
	return guids, names, spaceApps
}

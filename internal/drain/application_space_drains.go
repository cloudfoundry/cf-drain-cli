package drain

import (
	"regexp"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var spaceDrainPattern = "space-forwarder-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"

type ApplicationDrainLister struct {
	appLister    AppLister
	envProvider  EnvProvider
	appNameRegex *regexp.Regexp
}

type AppLister interface {
	ListApps(spaceGuid string) ([]cloudcontroller.App, error)
}

type EnvProvider interface {
	EnvVars(appGuid string) (map[string]string, error)
}

func NewApplicationDrainLister(appLister AppLister, envProvider EnvProvider) ApplicationDrainLister {
	return ApplicationDrainLister{
		appLister:    appLister,
		envProvider:  envProvider,
		appNameRegex: regexp.MustCompile(spaceDrainPattern),
	}
}

func (dl ApplicationDrainLister) Drains(spaceGUID string) ([]Drain, error) {
	apps, _ := dl.appLister.ListApps(spaceGUID)
	guids, names := dl.appMetadata(apps)

	var drains []Drain
	for _, app := range apps {
		if dl.appNameRegex.MatchString(app.Name) {
			// Do I need to be public?
			envs, err := dl.envProvider.EnvVars(app.Guid)
			if err != nil {
				return nil, err
			}

			drains = append(drains, Drain{
				Apps:        names,
				AppGuids:    guids,
				Type:        envs["DRAIN_TYPE"],
				DrainURL:    envs["DRAIN_URL"],
				AdapterType: "application",
			})
		}
	}
	return drains, nil
}

func (dl ApplicationDrainLister) appMetadata(apps []cloudcontroller.App) (guids []string, names []string) {
	for _, app := range apps {
		guids = append(guids, app.Guid)
		names = append(names, app.Name)
	}
	return guids, names
}

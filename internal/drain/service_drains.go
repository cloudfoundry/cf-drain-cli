package drain

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

type ServiceDrainLister struct {
	c           cloudcontroller.Curler
	appLister   AppLister
	envProvider EnvProvider
}

func NewServiceDrainLister(c cloudcontroller.Curler, appLister AppLister, envProvider EnvProvider) *ServiceDrainLister {
	return &ServiceDrainLister{
		c:           c,
		appLister:   appLister,
		envProvider: envProvider,
	}
}

type Drain struct {
	Name        string
	Guid        string
	Apps        []string
	AppGuids    []string
	Type        string
	DrainURL    string
	AdapterType string
	Scope       string
}

func (c *ServiceDrainLister) DeleteDrainAndUser(spaceGuid, drainName string) (bool, error) {
	drains, err := c.fetchSpaceDrains(spaceGuid)
	if err != nil {
		return false, err
	}
	sdrains, err := c.fetchServiceDrains(spaceGuid)
	if err != nil {
		return false, err
	}
	drains = append(drains, sdrains...)

	if err != nil {
		return false, fmt.Errorf("Failed to fetch drains: %s", err)
	}

	d, ok := c.findDrain(drains, drainName)
	if ok {
		if d.Scope == "space" {
			c.deleteDrain(d)
			c.deleteUser(fmt.Sprintf("space-drain-%s", d.Guid))
			return true, nil
		}
		c.unbindService(d)
		c.deleteService(d)
		return true, nil
	}

	return false, fmt.Errorf("Failed to find drain %s in space %s", drainName, spaceGuid)
}

func (c *ServiceDrainLister) Drains(spaceGuid string) ([]Drain, error) {
	return c.fetchServiceDrains(spaceGuid)
}

func (c *ServiceDrainLister) fetchServiceDrains(spaceGuid string) ([]Drain, error) {
	var url string
	url = fmt.Sprintf("/v2/user_provided_service_instances?q=space_guid:%s", spaceGuid)
	instances, err := c.fetchServiceInstances(url)
	if err != nil {
		return nil, err
	}

	var appGuids []string
	var drains []Drain
	for _, s := range instances {
		if s.Entity.SyslogDrainURL == "" {
			continue
		}

		apps, err := c.fetchApps(s.Entity.ServiceBindingsURL)
		if err != nil {
			return nil, err
		}
		appGuids = append(appGuids, apps...)

		drainType, err := c.TypeFromDrainURL(s.Entity.SyslogDrainURL)
		if err != nil {
			return nil, err
		}

		drain := Drain{
			Name:        s.Entity.Name,
			Guid:        s.MetaData.Guid,
			Apps:        apps,
			Type:        drainType,
			DrainURL:    s.Entity.SyslogDrainURL,
			AdapterType: "service",
			Scope:       "single",
		}

		drains = append(drains, drain)
	}

	appNames, err := c.fetchAppNames(appGuids)
	if err != nil {
		return nil, err
	}

	var namedDrains []Drain
	for _, d := range drains {
		var names []string
		var guids []string
		for _, guid := range d.Apps {
			names = append(names, appNames[guid])
			guids = append(guids, guid)
		}
		d.Apps = names
		d.AppGuids = guids
		namedDrains = append(namedDrains, d)
	}

	return namedDrains, nil
}

func (c *ServiceDrainLister) fetchSpaceDrains(spaceGuid string) ([]Drain, error) {
	spaceApps, _ := c.appLister.ListApps(spaceGuid)
	guids, names, _ := c.appMetadata(spaceApps)

	var drains []Drain
	for _, app := range spaceApps {
		envs, err := c.envProvider.EnvVars(app.Guid)
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
				Name:     app.Name,
				Guid:     app.Guid,
				Apps:     names,
				AppGuids: guids,
				Type:     envs["DRAIN_TYPE"],
				DrainURL: envs["DRAIN_URL"],
			})
		}
	}
	return drains, nil
}

func (c *ServiceDrainLister) fetchServiceInstances(url string) ([]userProvidedServiceInstance, error) {
	instances := []userProvidedServiceInstance{}
	for url != "" {
		resp, err := c.c.Curl(url, "GET", "")
		if err != nil {
			return nil, err
		}

		var services userProvidedServiceInstancesResponse
		err = json.Unmarshal(resp, &services)
		if err != nil {
			return nil, err
		}

		instances = append(instances, services.Resources...)

		url = services.NextURL
	}
	return instances, nil
}

func (c *ServiceDrainLister) fetchApps(url string) ([]string, error) {
	var apps []string
	for url != "" {
		resp, err := c.c.Curl(url, "GET", "")
		if err != nil {
			return nil, err
		}

		var serviceBindingsResponse serviceBindingsResponse
		err = json.Unmarshal(resp, &serviceBindingsResponse)
		if err != nil {
			return nil, err
		}

		for _, r := range serviceBindingsResponse.Resources {
			apps = append(apps, r.Entity.AppGuid)
		}

		url = serviceBindingsResponse.NextURL
	}

	return apps, nil
}

func (c *ServiceDrainLister) fetchAppNames(guids []string) (map[string]string, error) {
	if len(guids) == 0 {
		return nil, nil
	}

	allGuids := strings.Join(guids, ",")
	apps := make(map[string]string)

	url := fmt.Sprintf("/v3/apps?guids=%s", allGuids)
	for url != "" {
		resp, err := c.c.Curl(url, "GET", "")
		if err != nil {
			return nil, err
		}

		var appsResp appsResponse
		err = json.Unmarshal(resp, &appsResp)
		if err != nil {
			return nil, err
		}

		for _, a := range appsResp.Apps {
			apps[a.Guid] = a.Name
		}
		url = appsResp.Pagination.Next
	}

	return apps, nil
}

func (c *ServiceDrainLister) TypeFromDrainURL(URL string) (string, error) {
	uri, err := url.Parse(URL)
	if err != nil {
		return "", err
	}

	drainTypes := uri.Query()["drain-type"]
	if len(drainTypes) == 0 {
		return "logs", nil
	} else {
		return drainTypes[0], nil
	}
}

func (c *ServiceDrainLister) findDrain(ds []Drain, drainName string) (Drain, bool) {
	var drains []Drain
	for _, drain := range ds {
		if drain.Name == drainName {
			drains = append(drains, drain)
		}
	}

	if len(drains) == 0 {
		return Drain{}, false
	}

	if len(drains) > 1 {
		// can this ever happen?
		log.Printf("more than one drain found with name: %s", drainName)
		return drains[0], true
	}

	return drains[0], true
}

func (c *ServiceDrainLister) deleteDrain(drain Drain) {
	// command := []string{"delete", drain.Name, "-f"}
	// _, err := cli.CliCommand(command...)
	// if err != nil {
	// 	log.Fatalf("%s", err)
	// }
}

func (c *ServiceDrainLister) deleteUser(username string) {
	// command := []string{"delete-user", username, "-f"}
	// _, err := cli.CliCommand(command...)
	// if err != nil {
	// 	log.Fatalf("%s", err)
	// }
}

func (c *ServiceDrainLister) unbindService(drain Drain) {
	// services, err := cli.GetServices()
	// if err != nil {
	// 	log.Fatalf("%s", err)
	// }
	//
	// var namedService *plugin_models.GetServices_Model
	// for _, s := range services {
	// 	if s.Name == drainName {
	// 		namedService = &s
	// 		break
	// 	}
	// }
	//
	// if namedService == nil {
	// 	log.Fatalf("Unable to find service %s.", drainName)
	// }
	//
	// for _, app := range namedService.ApplicationNames {
	// 	command := []string{"unbind-service", app, drainName}
	// 	_, err := cli.CliCommand(command...)
	// 	if err != nil {
	// 		log.Fatalf("%s", err)
	// 	}
	// }

}

func (c *ServiceDrainLister) deleteService(drain Drain) {
	// command := []string{"delete-service", drainName, "-f"}
	// _, err = cli.CliCommand(command...)
	// if err != nil {
	// 	log.Fatalf("%s", err)
	// }
}

func (c *ServiceDrainLister) appMetadata(apps []cloudcontroller.App) (guids []string, names []string, spaceApps map[string]string) {
	spaceApps = make(map[string]string)
	for _, app := range apps {
		spaceApps[app.Guid] = app.Name
		guids = append(guids, app.Guid)
		names = append(names, app.Name)
	}
	return guids, names, spaceApps
}

type userProvidedServiceInstancesResponse struct {
	NextURL   string                        `json:"next_url"`
	Resources []userProvidedServiceInstance `json:"resources"`
}

type userProvidedServiceInstance struct {
	MetaData struct {
		Guid string `json:"guid"`
	} `json:"metadata"`
	Entity struct {
		Name               string `json:"name"`
		ServiceBindingsURL string `json:"service_bindings_url"`
		SyslogDrainURL     string `json:"syslog_drain_url"`
	} `json:"entity"`
}

type serviceBindingsResponse struct {
	NextURL   string           `json:"next_url"`
	Resources []serviceBinding `json:"resources"`
}

type serviceBinding struct {
	Entity struct {
		AppGuid string `json:"app_guid"`
		AppUrl  string `json:"app_url"`
	} `json:"entity"`
}

type appsResponse struct {
	Apps       []appData `json:"resources"`
	Pagination struct {
		Next string `json:"next"`
	} `json:pagination`
}

type appData struct {
	Name string `json:"name"`
	Guid string `json:"guid"`
}

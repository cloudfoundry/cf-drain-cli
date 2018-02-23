package cloudcontroller

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type ListDrainsClient struct {
	c Curler
}

type Curler interface {
	Curl(URL string) ([]byte, error)
}

func NewListDrainsClient(c Curler) *ListDrainsClient {
	return &ListDrainsClient{
		c: c,
	}
}

type Drain struct {
	Name     string
	Apps     []string
	Type     string
	DrainURL string
}

func (c *ListDrainsClient) Drains(spaceGuid string) ([]Drain, error) {
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

		drain, err := c.buildDrain(apps, s.Entity.Name, drainType, s.Entity.SyslogDrainURL)
		if err != nil {
			return nil, err
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
		for _, guid := range d.Apps {
			names = append(names, appNames[guid])
		}
		d.Apps = names
		namedDrains = append(namedDrains, d)
	}

	return namedDrains, nil
}

func (c *ListDrainsClient) fetchServiceInstances(url string) ([]userProvidedServiceInstance, error) {
	instances := []userProvidedServiceInstance{}
	for url != "" {
		resp, err := c.c.Curl(url)
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

func (c *ListDrainsClient) fetchApps(url string) ([]string, error) {
	var apps []string
	for url != "" {
		resp, err := c.c.Curl(url)
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

func (c *ListDrainsClient) fetchAppNames(guids []string) (map[string]string, error) {
	if len(guids) == 0 {
		return nil, nil
	}

	allGuids := strings.Join(guids, ",")
	apps := make(map[string]string)

	url := fmt.Sprintf("/v3/apps?guids=%s", allGuids)
	for url != "" {
		resp, err := c.c.Curl(url)
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

func (c *ListDrainsClient) TypeFromDrainURL(URL string) (string, error) {
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

func (c *ListDrainsClient) buildDrain(apps []string, name, drainType, drainURL string) (Drain, error) {
	return Drain{
		Name:     name,
		Apps:     apps,
		Type:     drainType,
		DrainURL: drainURL,
	}, nil
}

type userProvidedServiceInstancesResponse struct {
	NextURL   string                        `json:"next_url"`
	Resources []userProvidedServiceInstance `json:"resources"`
}

type userProvidedServiceInstance struct {
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

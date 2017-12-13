package cloudcontroller

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type DrainsClient struct {
	c Curler
}

type Curler interface {
	Curl(URL string) ([]byte, error)
}

func NewDrainsClient(c Curler) *DrainsClient {
	return &DrainsClient{
		c: c,
	}
}

type Drain struct {
	Name string
	Apps []string
	Type string
}

func (c *DrainsClient) Drains(spaceGuid string) ([]Drain, error) {
	var url string
	url = fmt.Sprintf("/v2/user_provided_service_instances?q=space_guid:%s", spaceGuid)
	instances, err := c.fetchServiceInstances(url)
	if err != nil {
		return nil, err
	}

	var drains []Drain
	for _, s := range instances {
		if s.Entity.SyslogDrainURL == "" {
			continue
		}

		apps, err := c.fetchApps(s.Entity.ServiceBindingsURL)
		if err != nil {
			return nil, err
		}
		drainName := s.Entity.Name
		drainType, err := c.TypeFromDrainURL(s.Entity.SyslogDrainURL)
		if err != nil {
			return nil, err
		}

		drain, err := c.buildDrain(apps, drainName, drainType)
		if err != nil {
			return nil, err
		}

		drains = append(drains, drain)
	}

	return drains, nil
}

func (c *DrainsClient) fetchServiceInstances(url string) ([]userProvidedServiceInstance, error) {
	instances := []userProvidedServiceInstance{}
	for url != "" {
		resp, err := c.c.Curl(
			url,
		)
		if err != nil {
			return nil, err
		}

		var services userProvidedServiceInstancesResponse
		if err := json.Unmarshal(resp, &services); err != nil {
			return nil, err
		}

		url = services.NextURL
		instances = append(instances, services.Resources...)
	}
	return instances, nil
}

func (c *DrainsClient) fetchApps(url string) ([]string, error) {
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

func (c *DrainsClient) TypeFromDrainURL(URL string) (string, error) {
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

func (c *DrainsClient) buildDrain(apps []string, name string, drainType string) (Drain, error) {
	return Drain{
		Name: name,
		Apps: apps,
		Type: drainType,
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

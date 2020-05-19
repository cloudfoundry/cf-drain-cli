package cloudcontroller

import (
	"encoding/json"
	"fmt"
)

type App struct {
	Name string
	Guid string
}

type Curler interface {
	Curl(URL, method, body string) ([]byte, error)
}

type AppListerClient struct {
	c Curler
}

func NewAppListerClient(c Curler) *AppListerClient {
	return &AppListerClient{
		c: c,
	}
}

func (c *AppListerClient) ListApps(spaceGuid string) ([]App, error) {
	resp, err := c.c.Curl(
		fmt.Sprintf("/v2/apps?q=space_guid:%s", spaceGuid),
		"GET",
		"",
	)

	if err != nil {
		return nil, err
	}

	var apps struct {
		Resources []struct {
			Metadata struct {
				Guid string
			}
			Entity struct {
				Name string
			}
		}
	}
	err = json.Unmarshal(resp, &apps)
	if err != nil {
		return nil, err
	}

	var a []App
	for _, r := range apps.Resources {
		a = append(a, App{r.Entity.Name, r.Metadata.Guid})
	}

	return a, nil
}

package cloudcontroller

import (
	"encoding/json"
	"fmt"
)

type AppListerClient struct {
	c Curler
}

func NewAppListerClient(c Curler) *AppListerClient {
	return &AppListerClient{
		c: c,
	}
}

func (c *AppListerClient) ListApps(spaceGuid string) ([]string, error) {
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
		}
	}
	err = json.Unmarshal(resp, &apps)
	if err != nil {
		return nil, err
	}

	var guids []string
	for _, r := range apps.Resources {
		guids = append(guids, r.Metadata.Guid)
	}

	return guids, nil
}

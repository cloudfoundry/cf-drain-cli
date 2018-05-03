package cloudcontroller

import (
	"encoding/json"
	"fmt"
)

type Client struct {
	c Curler
}

func NewClient(c Curler) *Client {
	return &Client{
		c: c,
	}
}

func (c *Client) EnvVars(appGUID string) (map[string]string, error) {
	resp, err := c.c.Curl(
		fmt.Sprintf("/v3/apps/%s/env", appGUID),
		"GET",
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app environment variables: %s", err)
	}

	var e env
	err = json.Unmarshal(resp, &e)
	if err != nil {
		return nil, err
	}

	return e.Vars, nil
}

type env struct {
	Vars map[string]string `json:"environment_variables"`
}

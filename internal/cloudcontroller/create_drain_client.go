package cloudcontroller

import (
	"fmt"
)

type CreateDrainClient struct {
	c Curler
}

func NewCreateDrainClient(c Curler) *CreateDrainClient {
	return &CreateDrainClient{
		c: c,
	}
}

func (c *CreateDrainClient) CreateDrain(name, url, spaceGuid, drainType string) error {
	if !validDrainType(drainType) {
		return fmt.Errorf("invalid drain type: %s", drainType)
	}

	url = fmt.Sprintf("%s?drain-type=%s", url, drainType)

	_, err := c.c.Curl(
		"/v2/user_provided_service_instances",
		"POST",
		c.buildRequestBody(name, url, spaceGuid),
	)

	return err
}

func (c *CreateDrainClient) buildRequestBody(name, url, spaceGuid string) string {
	return fmt.Sprintf(`
	{
	  "syslog_drain_url": %q,
	  "space_guid": %q,
	  "name": %q
	}`, url, spaceGuid, name)
}

func validDrainType(drainType string) bool {
	switch drainType {
	case "all", "metrics", "logs":
		return true
	default:
		return false
	}
}

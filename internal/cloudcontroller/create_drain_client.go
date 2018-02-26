package cloudcontroller

import "fmt"

type CreateDrainClient struct {
	c Curler
}

func NewCreateDrainClient(c Curler) *CreateDrainClient {
	return &CreateDrainClient{
		c: c,
	}
}

func (c *CreateDrainClient) CreateDrain(name, url, spaceGuid string) error {
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

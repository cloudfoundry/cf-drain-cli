package cloudcontroller

import "fmt"

type BindDrainClient struct {
	c Curler
}

func NewBindDrainClient(c Curler) *BindDrainClient {
	return &BindDrainClient{
		c: c,
	}
}

func (c *BindDrainClient) BindDrain(appGuid, serviceInstanceGuid string) error {
	_, err := c.c.Curl(
		"/v2/service_bindings",
		"POST",
		c.buildRequestBody(appGuid, serviceInstanceGuid),
	)
	return err
}

func (c *BindDrainClient) buildRequestBody(appGuid, serviceInstanceGuid string) string {
	return fmt.Sprintf(
		`{"service_instance_guid":%q, "app_guid":%q}`,
		serviceInstanceGuid,
		appGuid,
	)
}

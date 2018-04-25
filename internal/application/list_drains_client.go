package application

import (
	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

type ListDrainsClient struct {
	c Curler
}

type Curler interface {
	Curl(URL, method, body string) ([]byte, error)
}

func NewListDrainsClient(c Curler) *ListDrainsClient {
	return &ListDrainsClient{
		c: c,
	}
}

func (c *ListDrainsClient) Drains(spaceGuid string) ([]cloudcontroller.Drain, error) {
	// hit the cc to find out what apps are in the space
	// filter only apps that are syslog_forwarder apps
	panic("not implemented")
	return nil, nil
}

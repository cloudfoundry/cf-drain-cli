package cloudcontroller

import (
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

type CurlClient struct {
	conn plugin.CliConnection
}

func NewCurlClient(cli plugin.CliConnection) *CurlClient {
	return &CurlClient{conn: cli}
}

func (c *CurlClient) Curl(URL string) ([]byte, error) {
	resp, err := c.conn.CliCommandWithoutTerminalOutput(
		"curl",
		URL,
	)
	return []byte(strings.Join(resp, "\n")), err
}

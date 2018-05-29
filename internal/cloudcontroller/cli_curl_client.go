package cloudcontroller

import (
	"log"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

type CLICurlClient struct {
	conn plugin.CliConnection
}

func NewCLICurlClient(cli plugin.CliConnection) *CLICurlClient {
	return &CLICurlClient{conn: cli}
}

func (c *CLICurlClient) Curl(URL, method, body string) ([]byte, error) {
	if method != "GET" || body != "" {
		log.Panic("Request must be a GET with empty body")
	}
	resp, err := c.conn.CliCommandWithoutTerminalOutput(
		"curl",
		URL,
	)
	return []byte(strings.Join(resp, "\n")), err
}

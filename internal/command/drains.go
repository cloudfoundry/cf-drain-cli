package command

import (
	"strings"

	"code.cloudfoundry.org/cf-syslog-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cli/plugin"
)

type DrainFetcher interface {
	Drains(spaceGuid string) ([]cloudcontroller.Drain, error)
}

func Drains(
	cli plugin.CliConnection,
	fetcher DrainFetcher,
	args []string,
	log Logger) {
	if len(args) != 0 {
		log.Fatalf("Invalid arguments, expected 0, got %d.", len(args))
	}

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}
	_ = space

	// Header
	log.Printf("name\tbound apps")

	drains, err := fetcher.Drains(space.Guid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	for _, d := range drains {
		drain := []string{d.Name, strings.Join(d.Apps, ", ")}
		log.Printf(strings.Join(drain, "\t"))
	}
}

// curl /v2/user_provided_service_instances filtered by space guid
// filter out service instances with syslog drain url
// curl service_bindings_url
// get drain type from syslog_drain_url

type UserProvidedServiceInstancesResponse struct {
	Resources []UserProvidedServiceInstance `json:"resources"`
}

type UserProvidedServiceInstance struct {
	Entity struct {
		Name               string `json:"name"`
		SyslogDrainURL     string `json:"syslog_drain_url"`
		ServiceBindingsURL string `json:"service_bindings_url"`
	} `json:"entity"`
}

type ServiceBindingsResponse struct {
	Resources []ServiceBinding `json:"resources"`
}

type ServiceBinding struct {
	Entity struct {
		AppGuid string `json:"app_guid"`
		AppUrl  string `json:"app_url"`
	} `json:"entity"`
}

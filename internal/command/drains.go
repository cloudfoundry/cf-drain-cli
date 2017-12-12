package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

type CloudControllerClient interface {
	Curl(URL string) ([]byte, error)
}

func Drains(
	cli plugin.CliConnection,
	ccClient CloudControllerClient,
	args []string,
	log Logger) {
	if len(args) != 0 {
		log.Fatalf("Invalid arguments, expected 0, got %d.", len(args))
	}

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	resp, err := ccClient.Curl(
		fmt.Sprintf("/v2/user_provided_service_instances?q=space_guid:%s", space.Guid),
	)
	if err != nil {
		log.Fatalf("%s", err)
	}

	var ups UserProvidedServiceInstancesResponse
	err = json.Unmarshal(resp, &ups)
	if err != nil {
		log.Fatalf("Failed to parse response body: %s", err)
	}

	// Header
	log.Printf("name\tbound apps")

	for _, u := range ups.Resources {
		resp, _ = ccClient.Curl(
			u.Entity.ServiceBindingsURL,
		)
		var serviceBindings ServiceBindingsResponse
		_ = json.Unmarshal(resp, &serviceBindings)

		appGuids := []string{}
		for _, sb := range serviceBindings.Resources {
			appGuids = append(
				appGuids,
				sb.Entity.AppGuid,
			)
		}
		tabularServices := []string{u.Entity.Name, strings.Join(appGuids, ", ")}
		log.Printf(strings.Join(tabularServices, "\t"))
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

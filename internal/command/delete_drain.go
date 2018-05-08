package command

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

func DeleteDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, fetcher DrainFetcher) {
	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1, got %d.", len(args))
	}

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	var appDrains []drain.Drain
	drains, err := fetcher.Drains(space.Guid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	for _, drain := range drains {
		if drain.AdapterType == "application" {
			appDrains = append(appDrains, drain)
		}
	}

	if len(appDrains) > 0 {
		deleteDrains(cli, appDrains, "application")
		return
	}

	serviceName := args[0]

	services, err := cli.GetServices()
	if err != nil {
		log.Fatalf("%s", err)
	}

	var namedService *plugin_models.GetServices_Model
	for _, s := range services {
		if s.Name == serviceName {
			namedService = &s
			break
		}
	}

	if namedService == nil {
		log.Fatalf("Unable to find service %s.", serviceName)
	}

	log.Print(fmt.Sprintf("Are you sure you want to unbind %s from %s and delete %s? [y/N] ",
		serviceName,
		strings.Join(namedService.ApplicationNames, ", "),
		serviceName,
	))

	reader := bufio.NewReader(in)
	confirm, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read user input: %s", err)
	}

	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		log.Printf("Delete cancelled")
		return
	}

	for _, app := range namedService.ApplicationNames {
		command := []string{"unbind-service", app, serviceName}
		_, err := cli.CliCommand(command...)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	command := []string{"delete-service", serviceName, "-f"}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}

	deleteDrains(cli, drains, "application")
}

func deleteDrains(cli plugin.CliConnection, drains []drain.Drain, adapterType string) {
	for _, drain := range drains {
		if drain.AdapterType == adapterType {
			command := []string{"delete", drain.Name, "-f"}
			_, err := cli.CliCommand(command...)
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
	}
}

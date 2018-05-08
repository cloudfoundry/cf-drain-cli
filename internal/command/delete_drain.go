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

func DeleteDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, serviceDrainFetcher DrainFetcher, appDrainFetcher DrainFetcher) {
	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1, got %d.", len(args))
	}

	serviceName := args[0]

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	serviceDrains, err := serviceDrainFetcher.Drains(space.Guid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	svcDrain, ok := findDrain(serviceDrains, serviceName)
	if ok && svcDrain.Scope == "space" {
		deleteDrain(cli, svcDrain)
		return
	}

	appDrains, err := appDrainFetcher.Drains(space.Guid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	appDrain, ok := findDrain(appDrains, serviceName)
	if ok {
		deleteDrain(cli, appDrain)
		return
	}

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
}

func findDrain(ds []drain.Drain, drainName string) (drain.Drain, bool) {
	var drains []drain.Drain
	for _, drain := range ds {
		if drain.Name == drainName {
			drains = append(drains, drain)
		}
	}

	if len(drains) == 0 {
		return drain.Drain{}, false
	}

	if len(drains) > 1 {
		// can this ever happen?
		log.Printf("more than one drain found with name: %s", drainName)
		return drains[0], true
	}

	return drains[0], true
}

func deleteDrain(cli plugin.CliConnection, drain drain.Drain) {
	command := []string{"delete", drain.Name, "-f"}
	_, err := cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

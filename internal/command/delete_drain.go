package command

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

func DeleteDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, serviceDrainFetcher DrainFetcher, appDrainFetcher DrainFetcher) {
	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1, got %d.", len(args))
	}

	drainName := args[0]

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	ok := serviceDrainFetcher.DeleteDrainAndUser(space.Guid, drainName)
	if ok {
		return
	}

	ok = appDrainFetcher.DeleteDrainAndUser(space.Guid, drainName)
	if ok {
		return
	}

	services, err := cli.GetServices()
	if err != nil {
		log.Fatalf("%s", err)
	}

	var namedService *plugin_models.GetServices_Model
	for _, s := range services {
		if s.Name == drainName {
			namedService = &s
			break
		}
	}

	if namedService == nil {
		log.Fatalf("Unable to find service %s.", drainName)
	}

	log.Print(fmt.Sprintf("Are you sure you want to unbind %s from %s and delete %s? [y/N] ",
		drainName,
		strings.Join(namedService.ApplicationNames, ", "),
		drainName,
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
		command := []string{"unbind-service", app, drainName}
		_, err := cli.CliCommand(command...)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	command := []string{"delete-service", drainName, "-f"}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}

}

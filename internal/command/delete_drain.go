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
	flags "github.com/jessevdk/go-flags"
)

type deleteDrainOpts struct {
	Force bool `long:"force" short:"f"`
}

func DeleteDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, serviceDrainFetcher DrainFetcher, appDrainFetcher DrainFetcher) {
	opts := deleteDrainOpts{}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	args, err := parser.ParseArgs(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1, got %d.", len(args))
	}

	drainName := args[0]

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	ok := deleteDrainAndUser(cli, serviceDrainFetcher, true, space.Guid, drainName)
	if ok {
		return
	}

	ok = deleteDrainAndUser(cli, appDrainFetcher, false, space.Guid, drainName)
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

	if !opts.Force {
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

func deleteDrainAndUser(cli plugin.CliConnection, df DrainFetcher, isService bool, spaceGuid, drainName string) bool {
	drains, err := df.Drains(spaceGuid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	d, ok := findDrain(drains, drainName)
	if ok {
		if isService && d.Scope == "space" {
			deleteDrain(cli, d)
			deleteUser(cli, fmt.Sprintf("space-drain-%s", d.Guid))
			return true
		}
		if !isService {
			deleteDrain(cli, d)
			deleteUser(cli, fmt.Sprintf("drain-%s", d.AppGuids[0]))
			return true
		}
	}

	return false
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

func deleteUser(cli plugin.CliConnection, username string) {
	command := []string{"delete-user", username, "-f"}
	_, err := cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

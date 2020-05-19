package command

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	flags "github.com/jessevdk/go-flags"
)

type deleteDrainOpts struct {
	Force bool `long:"force" short:"f"`
}

func DeleteDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, serviceDrainFetcher DrainFetcher) {
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

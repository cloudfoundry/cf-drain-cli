package command

import (
	"bufio"
	"io"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	flags "github.com/jessevdk/go-flags"
)

type DeleteDrainFunc func(plugin.CliConnection, []string, Logger, io.Reader, DrainFetcher)

func DeleteSpaceDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, df DrainFetcher, deleteDrain DeleteDrainFunc) {
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

	if !opts.Force {
		log.Print("Are you sure you want to delete the space drain? [y/N] ")

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

	command := []string{"delete", "space-drain", "-f"}
	cli.CliCommand(command...)

	deleteDrain(cli, []string{drainName, "--force"}, log, nil, df)
}

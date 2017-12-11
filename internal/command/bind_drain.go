package command

import "code.cloudfoundry.org/cli/plugin"

func BindDrain(cli plugin.CliConnection, args []string, log Logger) {
	if len(args) != 2 {
		log.Fatalf("Invalid arguments, expected 2, got %d.", len(args))
	}

	appName := args[0]
	drainName := args[1]

	_, err := cli.CliCommand("bind-service", appName, drainName)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

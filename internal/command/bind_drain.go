package command

import (
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin"
)

func BindDrain(cli plugin.CliConnection, df DrainFetcher, args []string, log Logger) {
	if len(args) != 2 {
		log.Fatalf("Invalid arguments, expected 2, got %d.", len(args))
	}

	appName := args[0]
	drainName := args[1]

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	drains, err := df.Drains(space.Guid)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if !containsDrain(drains, drainName) {
		log.Fatalf("%s is not a valid drain.", drainName)
	}

	_, err = cli.CliCommand("bind-service", appName, drainName)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func containsDrain(drains []drain.Drain, drainName string) bool {
	for _, d := range drains {
		if d.Name == drainName {
			return true
		}
	}
	return false
}

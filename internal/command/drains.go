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

	drains, err := fetcher.Drains(space.Guid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	// Header
	log.Printf("name\tbound apps\ttype")

	for _, d := range drains {
		drain := []string{d.Name, strings.Join(d.Apps, ", "), d.Type}
		log.Printf(strings.Join(drain, "\t"))
	}
}

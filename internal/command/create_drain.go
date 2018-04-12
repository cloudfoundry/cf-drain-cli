package command

import (
	"flag"
	"fmt"
	"log"
	"net/url"

	"code.cloudfoundry.org/cli/plugin"
	uuid "github.com/nu7hatch/gouuid"
)

// Logger is used for outputting log-cache results and errors
type Logger interface {
	Printf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Print(...interface{})
}

func CreateDrain(cli plugin.CliConnection, args []string, log Logger) {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	drainType := f.String("type", "", "")
	drainName := f.String("drain-name", "", "")
	err := f.Parse(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(f.Args()) != 2 {
		log.Fatalf("Invalid arguments, expected 2, got %d.", len(f.Args()))
	}

	appName := f.Args()[0]
	drainURL := f.Args()[1]
	serviceName := buildDrainName(*drainName)

	_, err = cli.GetApp(appName)
	if err != nil {
		log.Fatalf("%s", err)
	}

	u, err := url.Parse(drainURL)
	if err != nil {
		log.Fatalf("Invalid syslog drain URL: %s", err)
	}

	if *drainType != "" {
		if !validDrainType(*drainType) {
			log.Fatalf("Invalid type: %s", *drainType)
		}

		qValues := u.Query()
		qValues.Set("drain-type", *drainType)
		u.RawQuery = qValues.Encode()
	}

	createAndBindService(cli, u, appName, serviceName, log)
}

func createAndBindService(cli plugin.CliConnection, u *url.URL, appName, serviceName string, log Logger) {
	command := []string{"create-user-provided-service", serviceName, "-l", u.String()}
	_, err := cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}

	command = []string{"bind-service", appName, serviceName}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func validDrainType(drainType string) bool {
	switch drainType {
	case "logs", "metrics", "all":
		return true
	default:
		return false
	}
}

func buildDrainName(drainName string) string {
	if drainName != "" {
		return drainName
	}

	guid, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("%s", err)
	}

	return fmt.Sprint("cf-drain-", guid)
}

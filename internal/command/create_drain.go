package command

import (
	"fmt"
	"log"
	"net/url"

	"code.cloudfoundry.org/cli/plugin"
	flags "github.com/jessevdk/go-flags"
	uuid "github.com/nu7hatch/gouuid"
)

// Logger is used for outputting log-cache results and errors
type Logger interface {
	Printf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Print(...interface{})
}

type createDrainOpts struct {
	AppOrServiceName string
	DrainName        string `long:"drain-name"`
	DrainType        string `long:"type"`
	DrainURL         string
}

func (f *createDrainOpts) serviceName() string {
	if f.DrainName != "" {
		return f.DrainName
	}

	guid, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("%s", err)
	}

	return fmt.Sprint("cf-drain-", guid)
}

func CreateDrain(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	p PasswordReader,
	log Logger,
) {
	opts := createDrainOpts{}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	args, err := parser.ParseArgs(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) != 2 {
		log.Fatalf("Invalid arguments, expected 2, got %d.", len(args))
	}

	opts.AppOrServiceName = args[0]
	opts.DrainURL = args[1]

	u, err := url.Parse(opts.DrainURL)
	if err != nil {
		log.Fatalf("Invalid syslog drain URL: %s", err)
	}

	if opts.DrainType != "" {
		if !validDrainType(opts.DrainType) {
			log.Fatalf("Invalid type: %s", opts.DrainType)
		}

		qValues := u.Query()
		qValues.Set("drain-type", opts.DrainType)
		u.RawQuery = qValues.Encode()
	}

	createAndBindService(cli, u, opts.AppOrServiceName, opts.serviceName(), log)
}

func createAndBindService(
	cli plugin.CliConnection,
	u *url.URL,
	appName, serviceName string,
	log Logger,
) {
	_, err := cli.GetApp(appName)
	if err != nil {
		log.Fatalf("%s", err)
	}

	command := []string{"create-user-provided-service", serviceName, "-l", u.String()}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}

	command = []string{"bind-service", appName, serviceName}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func sourceID(cli plugin.CliConnection, appName string) (string, error) {
	app, err := cli.GetApp(appName)
	if err != nil {
		svc, err := cli.GetService(appName)
		if err != nil {
			return "", err
		}

		return svc.Guid, nil
	}

	return app.Guid, nil
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

package command

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	uuid "github.com/nu7hatch/gouuid"
)

// Logger is used for outputting log-cache results and errors
type Logger interface {
	Printf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Print(...interface{})
}

func CreateDrain(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	log Logger,
) {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	drainType := f.String("type", "", "")
	drainName := f.String("drain-name", "", "")
	adapterType := f.String("adapter-type", "service", "")
	username := f.String("username", "", "")
	password := f.String("password", "", "")

	err := f.Parse(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if *adapterType == "application" {
		if *username == "" {
			log.Fatalf("missing required flag: username")
		}
		if *password == "" {
			log.Fatalf("missing required flag: password")
		}
	}

	if len(f.Args()) != 2 {
		log.Fatalf("Invalid arguments, expected 2, got %d.", len(f.Args()))
	}

	appName := f.Args()[0]
	drainURL := f.Args()[1]
	serviceName := buildDrainName(*drainName)

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

	switch *adapterType {
	case "service":
		createAndBindService(cli, u, appName, serviceName, log)
	case "application":
		pushSyslogForwarder(
			cli,
			u,
			appName,
			serviceName,
			*username,
			*password,
			d,
			log,
		)
	default:
		log.Fatalf("unsupported adapter type, must be 'service' or 'application'")
	}
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

func pushSyslogForwarder(
	cli plugin.CliConnection,
	u *url.URL,
	appOrServiceName string,
	serviceName string,
	username string,
	password string,
	d Downloader,
	log Logger,
) {
	sourceID, err := sourceID(cli, appOrServiceName)
	if err != nil {
		log.Fatalf("unknown application or service %q", appOrServiceName)
	}

	org, err := cli.GetCurrentOrg()
	if err != nil {
		log.Fatalf("%s", err)
	}
	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}
	apiEndpoint, err := cli.ApiEndpoint()
	if err != nil {
		log.Fatalf("%s", err)
	}

	path := path.Dir(d.Download("syslog_forwarder"))

	command := []string{
		"push",
		serviceName,
		"-p", path,
		"-b", "binary_buildpack",
		"-c", "./syslog_forwarder",
		"--no-start",
	}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}

	hostName := fmt.Sprintf("%s.%s.%s", org.Name, space.Name, appOrServiceName)
	uaaAddr := strings.Replace(apiEndpoint, "api.", "uaa.", 1)
	logCacheAddr := strings.Replace(apiEndpoint, "api.", "log-cache.", 1)
	groupName, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("%s", err)
	}
	envCommands := [][]string{
		{"set-env", serviceName, "SOURCE_ID", sourceID},
		{"set-env", serviceName, "SOURCE_HOST_NAME", hostName},
		{"set-env", serviceName, "UAA_URL", uaaAddr},
		{"set-env", serviceName, "CLIENT_ID", "cf"},
		{"set-env", serviceName, "USERNAME", username},
		{"set-env", serviceName, "PASSWORD", password},
		{"set-env", serviceName, "LOG_CACHE_HTTP_ADDR", logCacheAddr},
		{"set-env", serviceName, "SYSLOG_URL", u.String()},
		{"set-env", serviceName, "GROUP_NAME", groupName.String()},
	}

	for _, cmd := range envCommands {
		_, err = cli.CliCommandWithoutTerminalOutput(cmd...)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	command = []string{"start", serviceName}
	_, err = cli.CliCommand(command...)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func sourceID(cli plugin.CliConnection, appOrServiceName string) (string, error) {
	app, err := cli.GetApp(appOrServiceName)
	if err != nil {
		svc, err := cli.GetService(appOrServiceName)
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

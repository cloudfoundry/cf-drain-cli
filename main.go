package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin"
)

type CFDrainCLI struct{}

func (c CFDrainCLI) Run(conn plugin.CliConnection, args []string) {
	log := log.New(os.Stderr, "", 0)
	if len(args) == 0 {
		log.Fatalf("Expected at least 1 argument, but got 0.")
	}

	ccCurler := cloudcontroller.NewCLICurlClient(conn)
	sdClient := drain.NewServiceDrainLister(ccCurler)
	logger := newLogger(os.Stdout)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	downloader := command.NewGithubReleaseDownloader(httpClient, logger)

	switch args[0] {
	case "drain":
		command.CreateDrain(conn, args[1:], downloader, logger)
	case "delete-drain":
		command.DeleteDrain(conn, args[1:], logger, os.Stdin, sdClient)
	case "bind-drain":
		command.BindDrain(conn, sdClient, args[1:], logger)
	case "drains":
		command.Drains(conn, nil, logger, os.Stdout, sdClient)
	case "drain-space":
		tokenFetcher := command.NewTokenFetcher(configPath(log))
		command.PushSpaceDrain(conn, os.Stdin, args[1:], downloader, tokenFetcher, logger)
	case "delete-drain-space":
		command.DeleteSpaceDrain(conn, args[1:], logger, os.Stdin, sdClient, command.DeleteDrain)
	}
}

// version is set via ldflags at compile time. It should be JSON encoded
// plugin.VersionType. If it does not unmarshal, the plugin version will be
// left empty.
var version string

func (c CFDrainCLI) GetMetadata() plugin.PluginMetadata {
	var v plugin.VersionType
	// Ignore the error. If this doesn't unmarshal, then we want the default
	// VersionType.
	_ = json.Unmarshal([]byte(version), &v)

	return plugin.PluginMetadata{
		Name:    "drains",
		Version: v,
		Commands: []plugin.Command{
			{
				Name:     "drains",
				HelpText: "Lists all services for syslog drains.",
				UsageDetails: plugin.Usage{
					Usage: "drains",
				},
			},
			{
				Name:     "drain",
				HelpText: "Creates a user provided service for syslog drains and binds it to a given application.",
				UsageDetails: plugin.Usage{
					Usage: "drain <app-name> <syslog-drain-url> [options]",
					Options: map[string]string{
						"-drain-name": "The name of the drain that will be created. If excluded, the drain name will be `cf-drain-UUID`.",
						"-type":       "The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`",
					},
				},
			},
			{
				Name:     "bind-drain",
				HelpText: "Binds an application to an existing syslog drain.",
				UsageDetails: plugin.Usage{
					Usage: "bind-drain <app-name> <drain-name>",
				},
			},
			{
				Name:     "delete-drain",
				HelpText: "Unbinds the service from applications and deletes the service.",
				UsageDetails: plugin.Usage{
					Usage: "delete-drain <drain-name>",
					Options: map[string]string{
						"-force": "Skip warning prompt. Default is false",
					},
				},
			},
			{
				Name:     "drain-space",
				HelpText: "Pushes app to bind all apps in the space to the configured syslog drain",
				UsageDetails: plugin.Usage{
					Usage: "drain-space [OPTIONS]",
					Options: map[string]string{
						"-path":       "Path to the space drain app to push. If omitted the latest release will be downloaded",
						"-drain-name": "Name for the space drain. Required",
						"-drain-url":  "Syslog endpoint for the space drain. Required",
						"-type":       "Which log type to filter on (logs, metrics, all). Default is all",
						"-username":   "Username to use when pushing the app. If not specified, a user will be created (requires admin permissions)",
						"-force":      "Skip warning prompt. Default is false",
					},
				},
			},
			{
				Name:     "delete-drain-space",
				HelpText: "Deletes space drain app and unbinds all the apps in the space from the configured syslog drain",
				UsageDetails: plugin.Usage{
					Usage: "delete-drain-space <drain-name> [OPTIONS]",
					Options: map[string]string{
						"-force": "Skip warning prompt. Default is false",
					},
				},
			},
		},
	}
}

func main() {
	plugin.Start(CFDrainCLI{})
}

type logger struct {
	*log.Logger

	w io.Writer
}

func newLogger(w io.Writer) *logger {
	return &logger{
		Logger: log.New(os.Stdout, "", 0),
		w:      w,
	}
}

func (l *logger) Print(a ...interface{}) {
	fmt.Fprint(os.Stdout, a...)
}

func configPath(log *log.Logger) string {
	if cfHome := os.Getenv("CF_HOME"); cfHome != "" {
		return path.Join(cfHome, ".cf", "config.json")
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(usr.HomeDir, ".cf", "config.json")
}

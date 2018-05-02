package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cli/plugin"
)

type CFDrainCLI struct{}

func (c CFDrainCLI) Run(conn plugin.CliConnection, args []string) {
	if len(args) == 0 {
		log.Fatalf("Expected at least 1 argument, but got 0.")
	}

	ccCurler := cloudcontroller.NewCLICurlClient(conn)
	dClient := cloudcontroller.NewListDrainsClient(ccCurler)
	logger := newLogger(os.Stdout)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	downloader := command.NewGithubReleaseDownloader(httpClient, logger)

	switch args[0] {
	case "drain":
		command.CreateDrain(conn, args[1:], downloader, terminal.ReadPassword, logger)
	case "delete-drain":
		command.DeleteDrain(conn, args[1:], logger, os.Stdin)
	case "bind-drain":
		command.BindDrain(conn, dClient, args[1:], logger)
	case "drains":
		command.Drains(conn, dClient, nil, logger, os.Stdout)
	case "drain-space":
		command.PushSpaceDrain(conn, os.Stdin, terminal.ReadPassword, args[1:], downloader, logger)
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
					Usage: "drain [options] <app | service> <syslog-drain-url>",
					Options: map[string]string{
						"-type":         "The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`",
						"-adapter-type": "Set the type of adapter. The adapter is responsible for forwarding messages to the syslog drain. Available options: `service` or `application`. Service will use a cf user provided service that reads from loggregator and forwards to the drain. Application will deploy a cf application that reads from log-cache and forwards to the drain. Default is `service`",
						"-drain-name":   "The name of the app that will be created to forward messages to your drain. Default is `cf-drain-UUID`",
						"-username":     "The username to use for authentication when the `adapter-type` is `application`. If `adapter-type` is `application` and no username is provided, a user will be created.",
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
				},
			},
			{
				Name:     "drain-space",
				HelpText: "Pushes app to bind all apps in the space to the configured syslog drain",
				UsageDetails: plugin.Usage{
					Usage: "drain-space [OPTIONS]",
					Options: map[string]string{
						"-path":                "Path to the space drain app to push. If omitted the latest release will be downloaded",
						"-drain-name":          "Name for the space drain. Required",
						"-drain-url":           "Syslog endpoint for the space drain. Required",
						"-type":                "Which log type to filter on (logs, metrics, all). Default is all",
						"-adapter-type":        "Set the type of adapter. The adapter is responsible for forwarding messages to the space drain. Available options: `service` or `application`. Service will use a cf user provided service that reads from loggregator and forwards to the drain. Application will deploy a cf application that reads from log-cache and forwards to the drain. Default is `service`",
						"-username":            "Username to use when pushing the app. If not specified, a user will be created (requires admin permissions)",
						"-skip-ssl-validation": "Whether to ignore certificate errors. Default is false",
						"-force":               "Skip warning prompt. Default is false",
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

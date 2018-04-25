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
	"code.cloudfoundry.org/cf-drain-cli/internal/service"
	"code.cloudfoundry.org/cli/plugin"
)

type CFDrainCLI struct{}

func (c CFDrainCLI) Run(conn plugin.CliConnection, args []string) {
	if len(args) == 0 {
		log.Fatalf("Expected atleast 1 argument, but got 0.")
	}

	ccCurler := cloudcontroller.NewCLICurlClient(conn)
	// TODO: consider allowing adapter type to control how this is created vs
	//       just using the service adapter type
	dClient := service.NewListDrainsClient(ccCurler)
	logger := newLogger(os.Stdout)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	downloader := command.NewGithubReleaseDownloader(httpClient, logger)

	switch args[0] {
	case "drain", "create-drain":
		command.CreateDrain(conn, args[1:], downloader, logger)
	case "delete-drain":
		command.DeleteDrain(conn, args[1:], logger, os.Stdin)
	case "bind-drain":
		// TODO: consider allowing adapter type to control how this is created vs
		//       just using the service adapter type
		command.BindDrain(conn, dClient, args[1:], logger)
	case "drains":
		// TODO: consider allowing adapter type to control how this is created vs
		//       just using the service adapter type
		command.Drains(conn, dClient, nil, logger, os.Stdout)
	case "drain-space", "push-space-drain":
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
						"type":         "The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`",
						"adapter-type": "Set the type of adapter. The adapter is responsible for forwarding messages to the syslog drain. Available options: `service` or `application`. Service will use a cf user provided service that reads from loggregator and forwards to the drain. Application will deploy a cf application that reads from log-cache and forwards to the drain. Default is `service`",
						"drain-name":   "The name of the app that will be created to forward messages to your drain. Default is `cf-drain-UUID`",
						"username":     "The username to use for authentication when the `adapter-type` is `application`. Required if `adapter-type` is `application`.",
						"password":     "The password to use for authentication when the `adapter-type` is `application`. Required if `adapter-type` is `application`.",
					},
				},
			},
			{
				Name:     "create-drain",
				HelpText: "Deprecated. See the drain command for details.",
				UsageDetails: plugin.Usage{
					Usage: "drain [options] <app | service> <syslog-drain-url>",
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
				Name:     "push-space-drain",
				HelpText: "Deprecated. See the drain-space command for details.",
				UsageDetails: plugin.Usage{
					Usage: "push-space-drain [OPTIONS]",
				},
			},
			{
				Name:     "drain-space",
				HelpText: "Pushes app to bind all apps in the space to the configured syslog drain",
				UsageDetails: plugin.Usage{
					Usage: "drain-space [OPTIONS]",
					Options: map[string]string{
						"path":                "Path to the space drain app to push. If omitted the latest release will be downloaded",
						"drain-name":          "Name for the space drain. Required",
						"drain-url":           "Syslog endpoint for the space drain. Required",
						"type":                "Which log type to filter on (logs, metrics, all). Default is all",
						"username":            "Username to use when pushing the app. If not specified, a user will be created (requires admin permissions)",
						"skip-ssl-validation": "Whether to ignore certificate errors. Default is false",
						"force":               "Skip warning prompt. Default is false",
						"adapter-type":        "Set the type of adapter. The adapter is responsible for forwarding messages to the syslog drains. Available options: `service` or `application`. Service will use a cf user provided service that reads from loggregator and forwards to the drains. Application will deploy cf applications that read from log-cache and forward to the drains. Default is `service`",
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

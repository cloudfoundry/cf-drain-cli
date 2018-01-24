package main

import (
	"encoding/json"
	"log"
	"os"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cli/plugin"
)

type CFDrainCLI struct{}

func (c CFDrainCLI) Run(conn plugin.CliConnection, args []string) {
	if len(args) == 0 {
		log.Fatalf("Expected atleast 1 argument, but got 0.")
	}

	ccCurler := cloudcontroller.NewCurlClient(conn)
	dClient := cloudcontroller.NewDrainsClient(ccCurler)
	logger := log.New(os.Stdout, "", 0)

	switch args[0] {
	case "create-drain":
		command.CreateDrain(conn, args[1:], logger)
	case "delete-drain":
		command.DeleteDrain(conn, args[1:], logger)
	case "bind-drain":
		command.BindDrain(conn, dClient, args[1:], logger)
	case "drains":
		command.Drains(conn, dClient, nil, logger, os.Stdout)
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
				Name:     "create-drain",
				HelpText: "Creates a user provided service for syslog drains and binds it to a given application.",
				UsageDetails: plugin.Usage{
					Usage: "create-drain [options] <app-name> <drain-name> <syslog-drain-url>",
					Options: map[string]string{
						"type": "The type of logs to be sent to the syslog drain. Available types: `logs`, `metrics`, and `all`. Default is `logs`",
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
		},
	}
}

func main() {
	plugin.Start(CFDrainCLI{})
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"path"
	"text/tabwriter"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin"
	uuid "github.com/nu7hatch/gouuid"
)

type CFDrainCLI struct{}

func (c CFDrainCLI) Run(conn plugin.CliConnection, args []string) {
	rand.Seed(time.Now().UnixNano())
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
		if len(args) < 3 {
			c.exitWithUsage("drain")
		}
		command.CreateDrain(conn, args[1:], downloader, logger)
	case "delete-drain":
		if len(args) < 2 {
			c.exitWithUsage("delete-drain")
		}
		command.DeleteDrain(conn, args[1:], logger, os.Stdin, sdClient)
	case "bind-drain":
		if len(args) < 3 {
			c.exitWithUsage("bind-drain")
		}
		command.BindDrain(conn, sdClient, args[1:], logger)
	case "drains":
		command.Drains(conn, nil, logger, os.Stdout, sdClient)
	case "drain-space":
		if len(args) < 2 {
			c.exitWithUsage("drain-space", "SYSLOG_DRAIN_URL required of the form syslog://destinaton.url:port")
		}
		tokenFetcher := command.NewTokenFetcher(configPath(log))
		command.PushSpaceDrain(conn, args[1:], downloader, tokenFetcher, logger)
	case "delete-drain-space":
		if len(args) < 2 {
			c.exitWithUsage("delete-drain-space")
		}
		command.DeleteSpaceDrain(conn, args[1:], logger, os.Stdin, sdClient, command.DeleteDrain)
	case "drain-service":
		if len(args) < 2 {
			c.exitWithUsage("drain-service")
		}
		tf := command.NewTokenFetcher(configPath(log))
		command.PushServiceDrain(conn, args[1:], tf, logger, groupProvider)
	case "drain-services-in-space":
		if len(args) < 3 {
			c.exitWithUsage("drain-services-in-space")
		}
		tf := command.NewTokenFetcher(configPath(log))
		command.PushSpaceServiceDrain(
			conn,
			os.Stdin,
			args[1:],
			tf,
			logger,
			groupProvider,
			guidProvider,
		)
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
					Usage: "drain APP_NAME SYSLOG_DRAIN_URL [--drain-name NAME] [--type TYPE]",
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
					Usage: "bind-drain APP_NAME DRAIN_NAME",
				},
			},
			{
				Name:     "delete-drain",
				HelpText: "Unbinds the service from applications and deletes the service.",
				UsageDetails: plugin.Usage{
					Usage: "delete-drain DRAIN_NAME [--force]",
					Options: map[string]string{
						"-force": "Skip warning prompt. Default is false",
					},
				},
			},
			{
				Name:     "drain-space",
				HelpText: "Pushes app to bind all apps in the space to the configured syslog drain",
				UsageDetails: plugin.Usage{
					Usage: "drain-space SYSLOG_DRAIN_URL [--drain-name NAME] [--path PATH] [--type TYPE] [--force]",
					Options: map[string]string{
						"-drain-name": "Name for the space drain",
						"-path":       "Path to the space drain app to push. If omitted the latest release will be downloaded",
						"-type":       "Which log type to filter on (logs, metrics, all). Default is all",
					},
				},
			},
			{
				Name:     "delete-drain-space",
				HelpText: "Deletes space drain app and unbinds all the apps in the space from the configured syslog drain",
				UsageDetails: plugin.Usage{
					Usage: "delete-drain-space DRAIN_NAME [--force]",
					Options: map[string]string{
						"-force": "Skip warning prompt. Default is false",
					},
				},
			},
			{
				Name:     "drain-service",
				HelpText: "Pushes app to drain a single service",
				UsageDetails: plugin.Usage{
					Usage: "drain-service SERVICE_NAME SYSLOG_DRAIN_URL --path PATH",
					Options: map[string]string{
						"-path": "Path to the service drain zip file.",
					},
				},
			},
			{
				Name:     "drain-services-in-space",
				HelpText: "Pushes app to drain all services in space",
				UsageDetails: plugin.Usage{
					Usage: "drain-services-in-space SYSLOG_DRAIN_URL --path PATH",
					Options: map[string]string{
						"-path":  "Path to the service drain zip file.",
						"-force": "Skip warning prompt. Default is false",
					},
				},
			},
		},
	}
}

func (c CFDrainCLI) exitWithUsage(cmdName string, helpMsg ...string) {
	i := c.indexOfCommand(cmdName)
	fmt.Printf("\nInvalid arguments passed to %s command.", cmdName)
	if len(helpMsg) != 0 {
		fmt.Printf(" %s", helpMsg[0])
	}
	fmt.Println()
	fmt.Println()
	c.printUsage(cmdName, i)
	c.printOptions(cmdName, i)
	fmt.Println()
	os.Exit(127)
}

func (c CFDrainCLI) printUsage(cmdName string, index int) {
	fmt.Println("USAGE:")
	fmt.Print("   " + c.GetMetadata().Commands[index].UsageDetails.Usage)
	fmt.Println()
}

func (c CFDrainCLI) printOptions(cmdName string, index int) {
	if c.GetMetadata().Commands[index].UsageDetails.Options != nil {
		fmt.Println()
		fmt.Println("OPTIONS:")
	}
	tw := tabwriter.NewWriter(os.Stdout, 3, 2, 2, ' ', 0)
	for k, v := range c.GetMetadata().Commands[index].UsageDetails.Options {
		fmt.Fprintf(
			tw,
			"\t%s\t%s\n",
			"-"+k,
			v,
		)
	}
	tw.Flush()
}

func (c CFDrainCLI) indexOfCommand(name string) int {
	for i, cmd := range c.GetMetadata().Commands {
		if cmd.Name == name {
			return i
		}
	}
	return -1
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

func groupProvider() string {
	return fmt.Sprintf("%d", rand.Uint64())
}

func guidProvider() string {
	u, err := uuid.NewV4()
	if err != nil {
		log.Panicf("failed to generate UUID: %s", err)
	}

	return u.String()
}

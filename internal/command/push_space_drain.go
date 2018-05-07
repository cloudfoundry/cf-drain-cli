package command

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"path"
	"strconv"
	"strings"

	flags "github.com/jessevdk/go-flags"
	uuid "github.com/nu7hatch/gouuid"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

type Downloader interface {
	Download(assetName string) string
}

type pushSpaceDrainOpts struct {
	AdapterType string `long:"adapter-type"`
	DrainName   string `long:"drain-name" required:"true"`
	DrainURL    string `long:"drain-url" required:"true"`
	Username    string `long:"username"`
	Path        string `long:"path"`
	DrainType   string `long:"type"`
	Force       bool   `long:"force"`
	Password    string
}

type passwordReader func(int) ([]byte, error)

func PushSpaceDrain(
	cli plugin.CliConnection,
	reader io.Reader,
	pw passwordReader,
	args []string,
	d Downloader,
	log Logger,
) {
	opts := pushSpaceDrainOpts{
		AdapterType: "service",
		DrainType:   "all",
		Force:       false,
	}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	args, err := parser.ParseArgs(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) > 0 {
		log.Fatalf("Invalid arguments, expected 0, got %d.", len(args))
	}

	if opts.Username != "" {
		log.Printf("Enter a password for %s: ", opts.Username)
		bytePassword, err := pw(0)
		if err != nil {
			log.Fatalf("%s", err)
		}

		if string(bytePassword) == "" {
			log.Fatalf("Password cannot be blank.")
		}
		opts.Password = string(bytePassword)
	}

	if !opts.Force {
		log.Print(
			"The space drain functionality is an experimental feature. ",
			"See https://github.com/cloudfoundry/cf-drain-cli#space-drain-experimental for more details.\n",
			"Do you wish to proceed? [y/N] ",
		)

		buf := bufio.NewReader(reader)
		resp, err := buf.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read user input: %s", err)
		}
		if strings.TrimSpace(strings.ToLower(resp)) != "y" {
			log.Fatalf("OK, exiting.")
		}
	}

	switch opts.AdapterType {
	case "application":
		pushApplicationSpaceDrain(opts, cli, d, log)
	case "service":
		pushServiceSpaceDrain(opts, cli, d, log)
	default:
		log.Fatalf("Invalid value for flag `--adapter-type`: %s", opts.AdapterType)
	}
}

func pushApplicationSpaceDrain(opts pushSpaceDrainOpts, cli plugin.CliConnection, d Downloader, log Logger) {
	org := currentOrg(cli, log)
	space := currentSpace(cli, log)
	api := apiEndpoint(cli, log)
	appName := fmt.Sprintf("space-forwarder-%s", guid())

	envs := [][]string{
		{"LOG_CACHE_HTTP_ADDR", strings.Replace(api, "api", "log-cache", 1)},
		{"SOURCE_HOST_NAME", fmt.Sprintf("%s.%s.%s", org.Name, space.Name, appName)},
		{"GROUP_NAME", guid()},
	}

	pushDrain(cli, appName, "space_syslog", envs, opts, d, log)
}

func pushServiceSpaceDrain(opts pushSpaceDrainOpts, cli plugin.CliConnection, d Downloader, log Logger) {
	pushDrain(cli, "space-drain", "space_drain", nil, opts, d, log)
}

func pushDrain(cli plugin.CliConnection, appName, command string, extraEnvs [][]string, opts pushSpaceDrainOpts, d Downloader, log Logger) {
	if opts.Path == "" {
		log.Printf("Downloading latest space drain from github...")
		opts.Path = path.Dir(d.Download(command))
		log.Printf("Done downloading space drain from github.")
	}

	_, err := cli.CliCommand(
		"push", appName,
		"-p", opts.Path,
		"-b", "binary_buildpack",
		"-c", fmt.Sprint("./", command),
		"--health-check-type", "process",
		"--no-start",
		"--no-route",
	)
	if err != nil {
		log.Fatalf("%s", err)
	}

	space := currentSpace(cli, log)
	api := apiEndpoint(cli, log)

	if opts.Username == "" {
		app, err := cli.GetApp(appName)
		if err != nil {
			log.Fatalf("%s", err)
		}
		opts.Username = fmt.Sprintf("space-drain-%s", app.Guid)
		opts.Password = createUser(cli, opts.Username, log)
	}

	skipCertVerify, err := cli.IsSSLDisabled()
	if err != nil {
		log.Fatalf("%s", err)
	}

	sharedEnvs := [][]string{
		{"SPACE_ID", space.Guid},
		{"DRAIN_NAME", opts.DrainName},
		{"DRAIN_URL", opts.DrainURL},
		{"DRAIN_TYPE", opts.DrainType},
		{"API_ADDR", api},
		{"UAA_ADDR", strings.Replace(api, "api", "uaa", 1)},
		{"CLIENT_ID", "cf"},
		{"USERNAME", opts.Username},
		{"PASSWORD", opts.Password},
		{"SKIP_CERT_VERIFY", strconv.FormatBool(skipCertVerify)},
	}

	envs := append(sharedEnvs, extraEnvs...)
	for _, env := range envs {
		_, err := cli.CliCommandWithoutTerminalOutput("set-env", appName, env[0], env[1])
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	cli.CliCommand("start", appName)
}

func guid() string {
	u, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("failed to generate unique identifier: %s", err)
	}
	return u.String()
}

func currentOrg(cli plugin.CliConnection, log Logger) plugin_models.Organization {
	org, err := cli.GetCurrentOrg()
	if err != nil {
		log.Fatalf("%s", err)
	}
	return org
}

func currentSpace(cli plugin.CliConnection, log Logger) plugin_models.Space {
	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}
	return space
}

func apiEndpoint(cli plugin.CliConnection, log Logger) string {
	api, err := cli.ApiEndpoint()
	if err != nil {
		log.Fatalf("%s", err)
	}
	return api
}

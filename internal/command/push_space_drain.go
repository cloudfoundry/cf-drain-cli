package command

import (
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"

	flags "github.com/jessevdk/go-flags"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

type Downloader interface {
	Download(assetName string) string
}

type RefreshTokenFetcher interface {
	RefreshToken() (string, error)
}

type pushSpaceDrainOpts struct {
	DrainName        string `long:"drain-name"`
	DrainURL         string
	Path             string `long:"path"`
	DrainType        string `long:"type"`
	IncludeServices  bool   `long:"include-services"`
	ServiceDrainPath string `long:"path-to-service-drain-app"`
}

func PushSpaceDrain(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	f RefreshTokenFetcher,
	log Logger,
	guid GUIDProvider,
) {
	opts := pushSpaceDrainOpts{
		DrainType: "all",
		DrainName: "space-drain",
	}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	args, err := parser.ParseArgs(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1, got %d.", len(args))
	}

	opts.DrainURL = args[0]

	checkIfDrainExists(cli, opts.DrainName)

	refreshToken, err := f.RefreshToken()
	if err != nil {
		log.Fatalf("%s", err)
	}

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	org, err := cli.GetCurrentOrg()
	if err != nil {
		log.Fatalf("%s", err)
	}

	skipCertVerify, err := cli.IsSSLDisabled()
	if err != nil {
		log.Fatalf("%s", err)
	}

	pushDrain(cli, "space_drain", opts, d, log)
	setEnvVarsForSpaceDrain(cli, space, opts, refreshToken, skipCertVerify, log)
	cli.CliCommand("start", opts.DrainName)

	spaceServiceDrainAppName := fmt.Sprintf("space-services-forwarder-%s", guid())
	pushSpaceServiceDrain(cli, spaceServiceDrainAppName, opts, d, log)
	setEnvVarsForSpaceServiceDrain(
		cli,
		space,
		org,
		refreshToken,
		skipCertVerify,
		spaceServiceDrainAppName,
		opts.DrainURL,
		log,
	)

	cli.CliCommand("start", spaceServiceDrainAppName)
}

func checkIfDrainExists(cli plugin.CliConnection, appName string) {
	app, _ := cli.GetApp(appName)
	if app.Name == appName {
		log.Fatalf("A drain with that name already exists. Use --drain-name to create a drain with a different name.")
	}
}

func pushDrain(cli plugin.CliConnection, command string, opts pushSpaceDrainOpts, d Downloader, log Logger) {
	if opts.Path == "" {
		log.Printf("Downloading latest space drain from github...")
		opts.Path = path.Dir(d.Download(command))
		log.Printf("Done downloading space drain from github.")
	}

	_, err := cli.CliCommand(
		"push", opts.DrainName,
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
}

func setEnvVarsForSpaceDrain(cli plugin.CliConnection, space plugin_models.Space, opts pushSpaceDrainOpts, refreshToken string, skipCertVerify bool, log Logger) {
	api := apiEndpoint(cli, log)

	envs := [][]string{
		{"SPACE_ID", space.Guid},
		{"DRAIN_NAME", opts.DrainName},
		{"DRAIN_URL", opts.DrainURL},
		{"DRAIN_TYPE", opts.DrainType},
		{"API_ADDR", api},
		{"UAA_ADDR", strings.Replace(api, "api", "uaa", 1)},
		{"CLIENT_ID", "cf"},
		{"REFRESH_TOKEN", refreshToken},
		{"SKIP_CERT_VERIFY", strconv.FormatBool(skipCertVerify)},
		{"DRAIN_SCOPE", "space"},
	}

	for _, env := range envs {
		_, err := cli.CliCommandWithoutTerminalOutput("set-env", opts.DrainName, env[0], env[1])
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
}

func pushSpaceServiceDrain(cli plugin.CliConnection, appName string, opts pushSpaceDrainOpts, d Downloader, log Logger) {
	if opts.ServiceDrainPath == "" {
		log.Printf("Downloading latest space service drain from github...")
		opts.ServiceDrainPath = path.Dir(d.Download("forwarder.zip")) + "/forwarder.zip"
		log.Printf("Done downloading space service drain from github.")
	}
	_, err := cli.CliCommand(
		"push", appName,
		"-p", opts.ServiceDrainPath,
		"-i", "3",
		"-b", "binary_buildpack",
		"-c", "./run.sh",
		"--health-check-type", "process",
		"--no-start",
		"--no-route",
	)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func setEnvVarsForSpaceServiceDrain(
	cli plugin.CliConnection,
	space plugin_models.Space,
	org plugin_models.Organization,
	refreshToken string,
	skipCertVerify bool,
	appName string,
	drainURL string,
	log Logger) {
	envs := [][]string{
		{"SOURCE_HOSTNAME", fmt.Sprintf("%s.%s", org, space)},
		{"CLIENT_ID", "cf"},
		{"REFRESH_TOKEN", refreshToken},
		{"CACHE_SIZE", "0"},
		{"SKIP_CERT_VERIFY", fmt.Sprintf("%t", skipCertVerify)},
		{"SYSLOG_URL", drainURL},
	}

	for _, env := range envs {
		_, err := cli.CliCommandWithoutTerminalOutput("set-env", appName, env[0], env[1])
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
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

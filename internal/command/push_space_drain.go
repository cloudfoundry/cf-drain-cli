package command

import (
	"fmt"
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
	DrainName string `long:"drain-name"`
	DrainURL  string
	Path      string `long:"path"`
	DrainType string `long:"type"`
}

func PushSpaceDrain(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	f RefreshTokenFetcher,
	log Logger,
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

	app, _ := cli.GetApp(opts.DrainName)
	if app.Name == opts.DrainName {
		log.Fatalf("A drain with that name already exists. Use --drain-name to create a drain with a different name.")
	}

	pushDrain(cli, opts.DrainName, "space_drain", nil, opts, d, f, log)
}

func pushDrain(cli plugin.CliConnection, appName, command string, extraEnvs [][]string, opts pushSpaceDrainOpts, d Downloader, f RefreshTokenFetcher, log Logger) {
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
		"--no-start",
	)
	if err != nil {
		log.Fatalf("%s", err)
	}

	space := currentSpace(cli, log)
	api := apiEndpoint(cli, log)

	skipCertVerify, err := cli.IsSSLDisabled()
	if err != nil {
		log.Fatalf("%s", err)
	}

	refreshToken, err := f.RefreshToken()
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
		{"REFRESH_TOKEN", refreshToken},
		{"SKIP_CERT_VERIFY", strconv.FormatBool(skipCertVerify)},
		{"DRAIN_SCOPE", "space"},
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

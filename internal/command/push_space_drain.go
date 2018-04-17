package command

import (
	"bufio"
	"io"
	"path"
	"strconv"
	"strings"

	flags "github.com/jessevdk/go-flags"

	"code.cloudfoundry.org/cli/plugin"
)

type Downloader interface {
	Download(assetName string) string
}

type optionsFlags struct {
	Path           string `long:"path"`
	DrainName      string `long:"drain-name" required:"true"`
	DrainURL       string `long:"drain-url" required:"true"`
	DrainType      string `long:"type"`
	Username       string `long:"username" required:"true"`
	Password       string `long:"password" required:"true"`
	SkipCertVerify bool   `long:"skip-ssl-validation"`
	Force          bool   `long:"force"`
}

func PushSpaceDrain(cli plugin.CliConnection, reader io.Reader, args []string, d Downloader, log Logger) {
	opts := optionsFlags{
		DrainType:      "all",
		SkipCertVerify: false,
		Force:          false,
	}

	args, err := flags.ParseArgs(&opts, args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) > 0 {
		log.Fatalf("Invalid arguments, expected 0, got %d.", len(args))
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

	if opts.Path == "" {
		log.Printf("Downloading latest space drain from github...")
		opts.Path = path.Dir(d.Download("space_drain"))
		log.Printf("Done downloading space drain from github.")
	}

	_, err = cli.CliCommand(
		"push", "space-drain",
		"-p", opts.Path,
		"-b", "binary_buildpack",
		"-c", "./space_drain",
		"--health-check-type", "process",
		"--no-start",
		"--no-route",
	)
	if err != nil {
		log.Fatalf("%s", err)
	}

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	api, err := cli.ApiEndpoint()
	if err != nil {
		log.Fatalf("%s", err)
	}

	envs := map[string]string{
		"SPACE_ID":         space.Guid,
		"DRAIN_NAME":       opts.DrainName,
		"DRAIN_URL":        opts.DrainURL,
		"DRAIN_TYPE":       opts.DrainType,
		"API_ADDR":         api,
		"UAA_ADDR":         strings.Replace(api, "api", "uaa", 1),
		"CLIENT_ID":        "cf",
		"USERNAME":         opts.Username,
		"PASSWORD":         opts.Password,
		"SKIP_CERT_VERIFY": strconv.FormatBool(opts.SkipCertVerify),
	}

	for name, value := range envs {
		_, err := cli.CliCommandWithoutTerminalOutput(
			"set-env", "space-drain", name, value,
		)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	cli.CliCommand("start", "space-drain")
}

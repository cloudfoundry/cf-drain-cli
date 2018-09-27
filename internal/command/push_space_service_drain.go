package command

import (
	"fmt"

	"code.cloudfoundry.org/cli/plugin"
	flags "github.com/jessevdk/go-flags"
)

type pushSpaceServiceDrainOpts struct {
	DrainName string `long:"name"`
	DrainURL  string
	Path      string `long:"path"`
}

func PushSpaceServiceDrain(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	f RefreshTokenFetcher,
	log Logger,
	group GroupNameProvider,
	guid GUIDProvider,
) {
	var opts pushSpaceServiceDrainOpts

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	args, err := parser.ParseArgs(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1 got %d.", len(args))
	}

	skipCertVerify, err := cli.IsSSLDisabled()
	if err != nil {
		log.Fatalf("%s", err)
	}

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

	if opts.DrainName == "" {
		opts.DrainName = fmt.Sprintf("space-services-forwarder-%s", guid())
	}
	opts.DrainURL = args[0]

	if opts.Path == "" {
		log.Printf("Downloading latest space service drain from github...")
		opts.Path = d.Download("forwarder.zip")
		log.Printf("Done downloading space service drain from github.")
	}

	_, err = cli.CliCommand(
		"push", opts.DrainName,
		"-p", opts.Path,
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

	envs := [][]string{
		{"SOURCE_HOSTNAME", fmt.Sprintf("%s.%s", org.Name, space.Name)},
		{"INCLUDE_SERVICES", "true"},
		{"CLIENT_ID", "cf"},
		{"REFRESH_TOKEN", refreshToken},
		{"CACHE_SIZE", "0"},
		{"SKIP_CERT_VERIFY", fmt.Sprintf("%t", skipCertVerify)},
		{"GROUP_NAME", group()},
		{"SYSLOG_URL", opts.DrainURL},
	}

	for _, env := range envs {
		_, err := cli.CliCommandWithoutTerminalOutput("set-env", opts.DrainName, env[0], env[1])
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	cli.CliCommand("start", opts.DrainName)
}

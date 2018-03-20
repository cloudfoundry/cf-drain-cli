package command

import (
	"bufio"
	"flag"
	"io"
	"path"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

type Downloader interface {
	Download() string
}

func PushSpaceDrain(cli plugin.CliConnection, reader io.Reader, args []string, d Downloader, log Logger) {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	p := f.String("path", "", "")
	drainName := f.String("drain-name", "", "")
	drainURL := f.String("drain-url", "", "")
	drainType := f.String("type", "all", "")
	username := f.String("username", "", "")
	password := f.String("password", "", "")
	skipCertVerify := f.Bool("skip-ssl-validation", false, "")
	force := f.Bool("force", false, "")
	err := f.Parse(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	f.VisitAll(func(flag *flag.Flag) {
		if flag.Value.String() == "" && (flag.Name != "skip-ssl-validation" && flag.Name != "type" && flag.Name != "path") {
			log.Fatalf("required flag --%s missing", flag.Name)
		}
	})

	if !*force {
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

	if *p == "" {
		log.Printf("Downloading latest space drain from github...")
		*p = path.Dir(d.Download())
		log.Printf("Done downloading space drain from github.")
	}

	_, err = cli.CliCommand(
		"push", "space-drain",
		"-p", *p,
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
		"DRAIN_NAME":       *drainName,
		"DRAIN_URL":        *drainURL,
		"DRAIN_TYPE":       *drainType,
		"API_ADDR":         api,
		"UAA_ADDR":         strings.Replace(api, "api", "uaa", 1),
		"CLIENT_ID":        "cf",
		"USERNAME":         *username,
		"PASSWORD":         *password,
		"SKIP_CERT_VERIFY": strconv.FormatBool(*skipCertVerify),
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

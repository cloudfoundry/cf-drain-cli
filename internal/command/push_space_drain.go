package command

import (
	"flag"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

func PushSpaceDrain(cli plugin.CliConnection, args []string, log Logger) {

	f := flag.NewFlagSet("", flag.ContinueOnError)
	path := f.String("path", "", "")
	drainName := f.String("drain-name", "", "")
	drainURL := f.String("drain-url", "", "")
	drainType := f.String("type", "all", "")
	username := f.String("username", "", "")
	password := f.String("password", "", "")
	skipCertVerify := f.Bool("skip-ssl-validation", false, "")
	err := f.Parse(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	f.VisitAll(func(flag *flag.Flag) {
		if flag.Value.String() == "" && (flag.Name != "skip-ssl-validation" || flag.Name != "type") {
			log.Fatalf("required flag --%s missing", flag.Name)
		}
	})

	_, err = cli.CliCommand(
		"push", "space-drain",
		"-p", *path,
		"-b", "binary_buildpack",
		"-c", "./space_manager",
		"--health-check-type", "process",
		"--no-start",
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

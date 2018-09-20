package command

import (
	"fmt"
	"net/url"

	"code.cloudfoundry.org/cli/plugin"
	flags "github.com/jessevdk/go-flags"
)

type migrateSpaceDrainOpts struct {
	DrainName string `long:"drain-name"`
	DrainURL  string
	Path      string `long:"path"`
}

func MigrateSpaceDrain(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	f RefreshTokenFetcher,
	fetcher DrainFetcher,
	log Logger,
	guid GUIDProvider,
) {
	opts := migrateSpaceDrainOpts{
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

	opts.DrainURL = args[0]

	drainURL, err := url.Parse(opts.DrainURL)
	if err != nil {
		log.Fatalf("Invalid drain URL: %s", err)
	}

	if opts.Path == "" {
		log.Printf("Downloading latest syslog forwarder from github...")
		opts.Path = d.Download("forwarder.zip")
		log.Printf("Done downloading syslog forwarder from github.")
	}

	envs := [][]string{
		{"SOURCE_HOSTNAME", fmt.Sprintf("%s.%s.%s", org.Name, space.Name, opts.DrainName)},
		{"CLIENT_ID", "cf"},
		{"REFRESH_TOKEN", refreshToken},
		{"SKIP_CERT_VERIFY", fmt.Sprintf("%t", skipCertVerify)},
		{"SYSLOG_URL", opts.DrainURL},
	}
	pushSyslogForwarder(cli, log, opts.DrainName, opts.Path, envs)

	apps, err := cli.GetApps()
	if err != nil {
		log.Fatalf("%s", err)
	}

	for _, app := range apps {
		if app.Name == opts.DrainName {
			continue
		}

		a, err := cli.GetApp(app.Name)
		if err != nil {
			log.Fatalf("%s", err)
		}

		dURL, ok := a.EnvironmentVars["DRAIN_URL"].(string)
		if !ok {
			continue
		}

		u, err := url.Parse(dURL)
		if err != nil {
			continue
		}

		if drainURL.Scheme == u.Scheme && drainURL.Host == u.Host {
			_, err := cli.CliCommand("delete", a.Name, "-r", "-f")
			if err != nil {
				log.Fatalf("Failed to delete old space drain: %s", err)
			}
		}
	}

	drains, err := fetcher.Drains(space.Guid)
	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}

	for _, drain := range drains {
		u, err := url.Parse(drain.DrainURL)
		if err != nil {
			continue
		}

		if drainURL.Scheme == u.Scheme && drainURL.Host == u.Host {
			for _, app := range drain.Apps {
				_, err := cli.CliCommand("unbind-service", app, drain.Name)
				if err != nil {
					log.Fatalf("%s", err)
				}
			}

			_, err := cli.CliCommand("delete-service", drain.Name)
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
	}
}

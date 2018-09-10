package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MigrateSpaceDrain", func() {
	var (
		logger              *stubLogger
		cli                 *stubCliConnection
		downloader          *stubDownloader
		refreshTokenFetcher *stubRefreshTokenFetcher
		serviceDrainFetcher *stubDrainFetcher
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
		cli.currentSpaceName = "space"
		cli.currentOrgName = "org"
		cli.apiEndpoint = "https://api.something.com"
		cli.sslDisabled = true
		downloader = newStubDownloader()
		downloader.path = "/downloaded/temp/dir/space_drain"

		refreshTokenFetcher = newStubRefreshTokenFetcher()
		refreshTokenFetcher.token = "some-refresh-token"

		serviceDrainFetcher = newStubDrainFetcher()
	})

	It("deploys the syslog forwarder and removes existing CUPS services", func() {
		cli.getAppsApps = []plugin_models.GetAppsModel{
			{
				Name: "existing-space-drain",
			},
		}
		cli.getAppName = "existing-space-drain"
		cli.getAppEnvVars = map[string]interface{}{
			"DRAIN_URL": "syslog://my-drain.drain:4123?drain-type=all",
		}
		serviceDrainFetcher.drains = []drain.Drain{
			{Name: "drain-a", Apps: []string{"app-a", "app-b"}, DrainURL: "syslog://my-drain.drain:4123"},
			{Name: "drain-b", Apps: []string{"app-c", "app-d"}, DrainURL: "syslog://my-other-drain.drain:4123"},
		}

		command.MigrateSpaceDrain(
			cli,
			[]string{
				"syslog://my-drain.drain:4123",
				"--path", "/tmp/syslog-forwarder.zip",
			},
			downloader,
			refreshTokenFetcher,
			serviceDrainFetcher,
			logger,
			func() string { return "a-guid" },
		)

		Expect(cli.cliCommandArgs).To(HaveLen(6))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push",
				"space-drain",
				"-p", "/tmp/syslog-forwarder.zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(Equal([][]string{
			{"set-env", "space-drain", "SOURCE_HOSTNAME", "org.space.space-drain"},
			{"set-env", "space-drain", "CLIENT_ID", "cf"},
			{"set-env", "space-drain", "REFRESH_TOKEN", "some-refresh-token"},
			{"set-env", "space-drain", "SKIP_CERT_VERIFY", "true"},
			{"set-env", "space-drain", "SYSLOG_URL", "syslog://my-drain.drain:4123"},
		}))

		Expect(cli.cliCommandArgs[1]).To(Equal([]string{
			"start", "space-drain",
		}))

		Expect(cli.cliCommandArgs[2]).To(Equal([]string{
			"delete", "existing-space-drain", "-r",
		}))

		Expect(cli.cliCommandArgs[3]).To(Equal([]string{
			"unbind-service", "app-a", "drain-a",
		}))

		Expect(cli.cliCommandArgs[4]).To(Equal([]string{
			"unbind-service", "app-b", "drain-a",
		}))

		Expect(cli.cliCommandArgs[5]).To(Equal([]string{
			"delete-service", "drain-a",
		}))
	})

	It("does not delete the existing app if it has the same name", func() {
		cli.getAppsApps = []plugin_models.GetAppsModel{
			{
				Name: "space-drain",
			},
		}
		cli.getAppName = "space-drain"
		cli.getAppEnvVars = map[string]interface{}{
			"DRAIN_URL": "syslog://my-drain.drain:4123?drain-type=all",
		}
		serviceDrainFetcher.drains = []drain.Drain{
			{Name: "drain-a", Apps: []string{"app-a", "app-b"}, DrainURL: "syslog://my-drain.drain:4123"},
			{Name: "drain-b", Apps: []string{"app-c", "app-d"}, DrainURL: "syslog://my-other-drain.drain:4123"},
		}

		command.MigrateSpaceDrain(
			cli,
			[]string{
				"syslog://my-drain.drain:4123",
				"--path", "/tmp/syslog-forwarder.zip",
			},
			downloader,
			refreshTokenFetcher,
			serviceDrainFetcher,
			logger,
			func() string { return "a-guid" },
		)

		Expect(cli.cliCommandArgs).To(HaveLen(5))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push",
				"space-drain",
				"-p", "/tmp/syslog-forwarder.zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(Equal([][]string{
			{"set-env", "space-drain", "SOURCE_HOSTNAME", "org.space.space-drain"},
			{"set-env", "space-drain", "CLIENT_ID", "cf"},
			{"set-env", "space-drain", "REFRESH_TOKEN", "some-refresh-token"},
			{"set-env", "space-drain", "SKIP_CERT_VERIFY", "true"},
			{"set-env", "space-drain", "SYSLOG_URL", "syslog://my-drain.drain:4123"},
		}))

		Expect(cli.cliCommandArgs[1]).To(Equal([]string{
			"start", "space-drain",
		}))

		Expect(cli.cliCommandArgs[2]).To(Equal([]string{
			"unbind-service", "app-a", "drain-a",
		}))

		Expect(cli.cliCommandArgs[3]).To(Equal([]string{
			"unbind-service", "app-b", "drain-a",
		}))

		Expect(cli.cliCommandArgs[4]).To(Equal([]string{
			"delete-service", "drain-a",
		}))
	})

	It("downloads the syslog-forwarder if path is not given", func() {
		command.MigrateSpaceDrain(
			cli,
			[]string{"syslog://my-drain.drain:4123"},
			downloader,
			refreshTokenFetcher,
			serviceDrainFetcher,
			logger,
			func() string { return "a-guid" },
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push",
				"space-drain",
				"-p", downloader.path,
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(downloader.assetName).To(Equal("forwarder.zip"))
	})

	It("uses the drain-name flag", func() {
		command.MigrateSpaceDrain(
			cli,
			[]string{
				"syslog://my-drain.drain:4123",
				"--path", "/tmp/syslog-forwarder.zip",
				"--drain-name", "my-drain-name",
			},
			downloader,
			refreshTokenFetcher,
			serviceDrainFetcher,
			logger,
			func() string { return "a-guid" },
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push",
				"my-drain-name",
				"-p", "/tmp/syslog-forwarder.zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(Equal([][]string{
			{"set-env", "my-drain-name", "SOURCE_HOSTNAME", "org.space.my-drain-name"},
			{"set-env", "my-drain-name", "CLIENT_ID", "cf"},
			{"set-env", "my-drain-name", "REFRESH_TOKEN", "some-refresh-token"},
			{"set-env", "my-drain-name", "SKIP_CERT_VERIFY", "true"},
			{"set-env", "my-drain-name", "SYSLOG_URL", "syslog://my-drain.drain:4123"},
		}))
	})

	It("fatally logs if cf push fails", func() {
		cli.pushAppError = errors.New("an error")

		Expect(func() {
			command.MigrateSpaceDrain(
				cli,
				[]string{
					"syslog://my-drain.drain:4123",
					"--path", "/tmp/syslog-forwarder.zip",
				},
				downloader,
				refreshTokenFetcher,
				serviceDrainFetcher,
				logger,
				func() string { return "a-guid" },
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("an error"))
	})

	It("fatally logs if no SYSLOG_DRAIN_URL is provided", func() {
		Expect(func() {
			command.MigrateSpaceDrain(cli, []string{}, nil, nil, nil, logger, nil)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 0."))
	})

	It("fatally logs if getting current drains fails", func() {
		serviceDrainFetcher.err = errors.New("an error")

		Expect(func() {
			command.MigrateSpaceDrain(
				cli,
				[]string{
					"syslog://my-drain.drain:4123",
					"--path", "/tmp/syslog-forwarder.zip",
				},
				downloader,
				refreshTokenFetcher,
				serviceDrainFetcher,
				logger,
				func() string { return "a-guid" },
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Failed to fetch drains: an error"))
	})

	It("fatally logs if it fails to unbind a service", func() {
		serviceDrainFetcher.drains = []drain.Drain{
			{Name: "drain-a", Apps: []string{"app-a"}, DrainURL: "syslog://my-drain.drain:4123"},
		}
		cli.unbindServiceError = errors.New("an error")

		Expect(func() {
			command.MigrateSpaceDrain(
				cli,
				[]string{
					"syslog://my-drain.drain:4123",
					"--path", "/tmp/syslog-forwarder.zip",
				},
				downloader,
				refreshTokenFetcher,
				serviceDrainFetcher,
				logger,
				func() string { return "a-guid" },
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("an error"))
	})

	It("fatally logs if it fails to delete the service", func() {
		serviceDrainFetcher.drains = []drain.Drain{
			{Name: "drain-a", Apps: []string{"app-a"}, DrainURL: "syslog://my-drain.drain:4123"},
		}
		cli.deleteServiceError = errors.New("an error")

		Expect(func() {
			command.MigrateSpaceDrain(
				cli,
				[]string{
					"syslog://my-drain.drain:4123",
					"--path", "/tmp/syslog-forwarder.zip",
				},
				downloader,
				refreshTokenFetcher,
				serviceDrainFetcher,
				logger,
				func() string { return "a-guid" },
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("an error"))
	})

	It("fatally logs if getting apps fails", func() {
		cli.getAppsError = errors.New("an error")

		Expect(func() {
			command.MigrateSpaceDrain(
				cli,
				[]string{
					"syslog://my-drain.drain:4123",
					"--path", "/tmp/syslog-forwarder.zip",
				},
				downloader,
				refreshTokenFetcher,
				serviceDrainFetcher,
				logger,
				func() string { return "a-guid" },
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("an error"))
	})

	It("fatally logs if getting an individual app fails", func() {
		cli.getAppsApps = []plugin_models.GetAppsModel{
			{
				Name: "existing-space-drain",
			},
		}
		cli.getAppError = errors.New("an error")

		Expect(func() {
			command.MigrateSpaceDrain(
				cli,
				[]string{
					"syslog://my-drain.drain:4123",
					"--path", "/tmp/syslog-forwarder.zip",
				},
				downloader,
				refreshTokenFetcher,
				serviceDrainFetcher,
				logger,
				func() string { return "a-guid" },
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("an error"))
	})
})

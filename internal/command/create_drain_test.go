package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateDrain", func() {
	var (
		logger *stubLogger
		cli    *stubCliConnection
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
	})

	Context("with service adapter type", func() {
		It("creates and binds to a user provided service", func() {
			args := []string{"app-name", "syslog://a.com?a=b"}

			command.CreateDrain(cli, args, nil, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(ConsistOf(
				"create-user-provided-service",
				MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
				"-l",
				"syslog://a.com?a=b",
			))

			Expect(cli.cliCommandArgs[1]).To(ConsistOf(
				"bind-service",
				"app-name",
				MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
			))
		})

		Describe("drain name flag", func() {
			It("creates and binds to a user provided service with the given name", func() {
				args := []string{"--drain-name", "my-drain", "app-name", "syslog://a.com?a=b"}

				command.CreateDrain(cli, args, nil, logger)

				Expect(cli.cliCommandArgs).To(HaveLen(2))
				Expect(cli.cliCommandArgs[0]).To(ConsistOf(
					"create-user-provided-service",
					"my-drain",
					"-l", "syslog://a.com?a=b",
				))

				Expect(cli.cliCommandArgs[1]).To(ConsistOf(
					"bind-service",
					"app-name",
					"my-drain",
				))
			})
		})

		Describe("type flag", func() {
			It("adds the drain type to the syslog URL for metrics", func() {
				args := []string{"--type", "metrics", "app-name", "syslog://a.com"}

				command.CreateDrain(cli, args, nil, logger)

				Expect(cli.cliCommandArgs).To(HaveLen(2))
				Expect(cli.cliCommandArgs[0]).To(ConsistOf(
					"create-user-provided-service",
					MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
					"-l", "syslog://a.com?drain-type=metrics",
				))
			})

			It("adds the drain type to the syslog URL for logs", func() {
				args := []string{"--type", "logs", "app-name", "syslog://a.com"}

				command.CreateDrain(cli, args, nil, logger)

				Expect(cli.cliCommandArgs).To(HaveLen(2))
				Expect(cli.cliCommandArgs[0]).To(ConsistOf(
					"create-user-provided-service",
					MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
					"-l", "syslog://a.com?drain-type=logs",
				))
			})

			It("adds the drain type to the syslog URL for all", func() {
				args := []string{"--type", "all", "app-name", "syslog://a.com"}

				command.CreateDrain(cli, args, nil, logger)

				Expect(cli.cliCommandArgs).To(HaveLen(2))
				Expect(cli.cliCommandArgs[0]).To(ConsistOf(
					"create-user-provided-service",
					MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
					"-l", "syslog://a.com?drain-type=all",
				))
			})

			It("fatally logs for unknown drain types", func() {
				args := []string{"--type", "garbage", "app-name", "syslog://a.com"}

				Expect(func() {
					command.CreateDrain(cli, args, nil, logger)
				}).To(Panic())
				Expect(logger.fatalfMessage).To(Equal("Invalid type: garbage"))
			})
		})

		It("fatally logs if the drain URL is invalid", func() {
			args := []string{"app-name", "://://blablabla"}

			Expect(func() {
				command.CreateDrain(cli, args, nil, logger)
			}).To(Panic())
			Expect(logger.fatalfMessage).To(Equal("Invalid syslog drain URL: parse ://://blablabla: missing protocol scheme"))
		})

		It("fatally logs if the incorrect number of arguments are given", func() {
			Expect(func() {
				command.CreateDrain(nil, []string{}, nil, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 0."))

			Expect(func() {
				command.CreateDrain(nil, []string{"one", "two", "three", "four"}, nil, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 4."))
		})

		It("fatally logs when an invalid app name is given", func() {
			cli.getAppError = errors.New("not an app")

			Expect(func() {
				command.CreateDrain(cli, []string{"not-an-app", "syslog://a.com"}, nil, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("not an app"))
			Expect(cli.getAppName).To(Equal("not-an-app"))
		})

		It("fatally logs when creating the service binding fails", func() {
			cli.createServiceError = errors.New("failed to create")

			Expect(func() {
				command.CreateDrain(cli, []string{"app-name", "syslog://a.com"}, nil, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("failed to create"))
		})

		It("fatally logs when binding the service fails", func() {
			cli.bindServiceError = errors.New("failed to bind")

			Expect(func() {
				command.CreateDrain(cli, []string{"app-name", "syslog://a.com"}, nil, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("failed to bind"))
		})
	})

	Context("with application adapter type", func() {
		var (
			downloader *stubDownloader
			args       []string
		)

		BeforeEach(func() {
			cli.getAppGuid = "application-guid"
			cli.currentOrgName = "org-name"
			cli.currentSpaceName = "space-name"
			cli.apiEndpoint = "api.example.com"

			downloader = newStubDownloader()
			downloader.path = "/downloaded/temp/dir/syslog_forwarder"

			args = []string{
				"--adapter-type", "application",
				"--drain-name", "my-drain",
				"--username", "user",
				"--password", "pass",
				"app-name",
				"syslog://a.com?a=b",
			}
		})

		It("push a syslog forwarder app", func() {
			command.CreateDrain(cli, args, downloader, logger)

			Expect(downloader.assetName).To(Equal("syslog_forwarder"))
			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(Equal(
				[]string{
					"push", "my-drain",
					"-p", "/downloaded/temp/dir",
					"-b", "binary_buildpack",
					"-c", "./syslog_forwarder",
					"--no-start",
				},
			))

			Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
				[]string{"set-env", "my-drain", "SOURCE_ID", "application-guid"},
				[]string{"set-env", "my-drain", "SOURCE_HOST_NAME", "org-name.space-name.app-name"},

				[]string{"set-env", "my-drain", "UAA_ADDR", "uaa.example.com"},
				[]string{"set-env", "my-drain", "CLIENT_ID", "cf"},

				[]string{"set-env", "my-drain", "USERNAME", "user"},
				[]string{"set-env", "my-drain", "PASSWORD", "pass"},

				[]string{"set-env", "my-drain", "LOG_CACHE_HTTP_ADDR", "log-cache.example.com"},
				[]string{"set-env", "my-drain", "SYSLOG_ADDR", "syslog://a.com?a=b"},
			))

			Expect(cli.cliCommandArgs[1]).To(Equal(
				[]string{
					"start", "my-drain",
				},
			))
		})

		It("fatally logs when we fail to get current org", func() {
			cli.currentOrgError = errors.New("an error")

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("an error"))
		})

		It("fatally logs when we fail to get current space", func() {
			cli.currentSpaceError = errors.New("an error")

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("an error"))
		})

		It("fatally logs when we fail to get api endpoint", func() {
			cli.apiEndpointError = errors.New("an error")

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("an error"))
		})

		It("fatally logs if username is not provided", func() {
			args = []string{
				"--adapter-type", "application",
				"--drain-name", "my-drain",
				"--password", "pass",
				"app-name",
				"syslog://a.com?a=b",
			}

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("missing required flag: username"))
		})

		It("fatally logs if password is not provided", func() {
			args = []string{
				"--adapter-type", "application",
				"--drain-name", "my-drain",
				"--username", "user",
				"app-name",
				"syslog://a.com?a=b",
			}

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("missing required flag: password"))
		})

		It("fatally logs if push fails", func() {
			cli.pushAppError = errors.New("push error")

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("push error"))
		})

		It("fatally logs if set env fails", func() {
			cli.setEnvErrors = map[string]error{
				"SOURCE_ID": errors.New("set-env error"),
			}

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("set-env error"))
		})

		It("fatally logs if starting the app fails", func() {
			cli.startAppError = errors.New("start error")

			Expect(func() {
				command.CreateDrain(cli, args, downloader, logger)
			}).To(Panic())

			Expect(logger.fatalfMessage).To(Equal("start error"))
		})
	})

	It("fatally logs if adapter-type is not service or application", func() {
		args := []string{
			"--adapter-type", "foo",
			"app-name",
			"syslog://a.com?a=b",
		}

		Expect(func() {
			command.CreateDrain(cli, args, nil, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("unsupported adapter type, must be 'service' or 'application'"))
	})
})

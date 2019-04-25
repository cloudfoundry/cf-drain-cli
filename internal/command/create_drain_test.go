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

	It("creates and binds to a user provided service", func() {
		args := []string{"app-name", "syslog://a.com?a=b"}

		command.CreateDrain(cli, args, logger)

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
			args := []string{
				"app-name",
				"syslog://a.com?a=b",
				"--drain-name", "my-drain",
			}

			command.CreateDrain(cli, args, logger)

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

		It("creates random drain name if --drain-name flag is not given", func() {
			args := []string{"app-name", "syslog://a.com?a=b"}

			command.CreateDrain(cli, args, logger)

			drainName := cli.cliCommandArgs[0][1]

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(ConsistOf(
				"create-user-provided-service",
				drainName,
				"-l", "syslog://a.com?a=b",
			))

			Expect(cli.cliCommandArgs[1]).To(ConsistOf(
				"bind-service",
				"app-name",
				drainName,
			))
		})
	})

	Describe("type flag", func() {
		It("adds the drain type to the syslog URL for metrics", func() {
			args := []string{"--type", "metrics", "app-name", "syslog://a.com"}

			command.CreateDrain(cli, args, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(ConsistOf(
				"create-user-provided-service",
				MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
				"-l", "syslog://a.com?drain-type=metrics",
			))
		})

		It("adds the drain type to the syslog URL for logs", func() {
			args := []string{"--type", "logs", "app-name", "syslog://a.com"}

			command.CreateDrain(cli, args, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(ConsistOf(
				"create-user-provided-service",
				MatchRegexp("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"),
				"-l", "syslog://a.com?drain-type=logs",
			))
		})

		It("adds the drain type to the syslog URL for all", func() {
			args := []string{"--type", "all", "app-name", "syslog://a.com"}

			command.CreateDrain(cli, args, logger)

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
				command.CreateDrain(cli, args, logger)
			}).To(Panic())
			Expect(logger.fatalfMessage).To(Equal("Invalid type: garbage"))
		})
	})

	It("fatally logs if the drain URL is invalid", func() {
		args := []string{"app-name", "://://blablabla"}

		Expect(func() {
			command.CreateDrain(cli, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid syslog drain URL: parse ://://blablabla: missing protocol scheme"))
	})

	It("fatally logs if the incorrect number of arguments are given", func() {
		Expect(func() {
			command.CreateDrain(nil, []string{}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 0."))

		Expect(func() {
			command.CreateDrain(nil, []string{"one", "two", "three", "four"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 4."))
	})

	It("fatally logs when an invalid app name is given", func() {
		cli.getAppName = "not-an-app"
		cli.getAppError = errors.New("not an app")

		Expect(func() {
			command.CreateDrain(cli, []string{"not-an-app", "syslog://a.com"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("not an app"))
		Expect(cli.getAppName).To(Equal("not-an-app"))
	})

	It("fatally logs when creating the service binding fails", func() {
		cli.createServiceError = errors.New("failed to create")

		Expect(func() {
			command.CreateDrain(cli, []string{"app-name", "syslog://a.com"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("failed to create"))
	})

	It("fatally logs when binding the service fails", func() {
		cli.bindServiceError = errors.New("failed to bind")

		Expect(func() {
			command.CreateDrain(cli, []string{"app-name", "syslog://a.com"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("failed to bind"))
	})
})

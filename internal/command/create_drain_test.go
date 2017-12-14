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
		args := []string{"app-name", "my-drain", "syslog://a.com?a=b"}

		command.CreateDrain(cli, args, logger)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"create-user-provided-service",
			"my-drain",
			"-l", "syslog://a.com?a=b",
		}))
		Expect(cli.cliCommandArgs[1]).To(Equal([]string{
			"bind-service", "app-name", "my-drain",
		}))
	})

	Describe("type flag", func() {
		It("adds the drain type to the syslog URL for metrics", func() {
			args := []string{"--type", "metrics", "app-name", "my-drain", "syslog://a.com"}

			command.CreateDrain(cli, args, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(Equal([]string{
				"create-user-provided-service",
				"my-drain",
				"-l", "syslog://a.com?drain-type=metrics",
			}))
		})

		It("adds the drain type to the syslog URL for logs", func() {
			args := []string{"--type", "logs", "app-name", "my-drain", "syslog://a.com"}

			command.CreateDrain(cli, args, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(Equal([]string{
				"create-user-provided-service",
				"my-drain",
				"-l", "syslog://a.com?drain-type=logs",
			}))
		})

		It("adds the drain type to the syslog URL for all", func() {
			args := []string{"--type", "all", "app-name", "my-drain", "syslog://a.com"}

			command.CreateDrain(cli, args, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(Equal([]string{
				"create-user-provided-service",
				"my-drain",
				"-l", "syslog://a.com?drain-type=all",
			}))
		})

		It("fatally logs for unknown drain types", func() {
			args := []string{"--type", "garbage", "app-name", "my-drain", "syslog://a.com"}

			Expect(func() {
				command.CreateDrain(cli, args, logger)
			}).To(Panic())
			Expect(logger.fatalfMessage).To(Equal("Invalid type: garbage"))
		})
	})

	It("fatally logs if the drain URL is invalid", func() {
		args := []string{"app-name", "my-drain", "://://blablabla"}

		Expect(func() {
			command.CreateDrain(cli, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid syslog drain URL: parse ://://blablabla: missing protocol scheme"))
	})

	It("fatally logs if the incorrect number of arguments are given", func() {
		Expect(func() {
			command.CreateDrain(nil, []string{}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 3, got 0."))

		Expect(func() {
			command.CreateDrain(nil, []string{"one", "two", "three", "four"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 3, got 4."))
	})

	It("fatally logs when an invalid app name is given", func() {
		cli.getAppError = errors.New("not an app")

		Expect(func() {
			command.CreateDrain(cli, []string{"not-an-app", "my-drain", "syslog://a.com"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("not an app"))
		Expect(cli.getAppName).To(Equal("not-an-app"))
	})

	It("fatally logs when creating the service binding fails", func() {
		cli.createServiceError = errors.New("failed to create")

		Expect(func() {
			command.CreateDrain(cli, []string{"app-name", "my-drain", "syslog://a.com"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("failed to create"))
	})

	It("fatally logs when binding the service fails", func() {
		cli.bindServiceError = errors.New("failed to bind")

		Expect(func() {
			command.CreateDrain(cli, []string{"app-name", "my-drain", "syslog://a.com"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("failed to bind"))
	})
})

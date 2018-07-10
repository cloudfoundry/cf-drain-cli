package command_test

import (
	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnableStructuredLogging", func() {
	var (
		logger *stubLogger
		cli    *stubCliConnection
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
	})

	It("creates and binds to a user provided service", func() {
		args := []string{"app-name", "DogStatsD"}

		command.EnableStructuredLogging(cli, args, nil, logger)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(ConsistOf(
			"create-user-provided-service",
			MatchRegexp("cf-drain-.*"),
			"-l",
			"prism://DogStatsD",
		))

		Expect(cli.cliCommandArgs[1]).To(ConsistOf(
			"bind-service",
			"app-name",
			MatchRegexp("cf-drain-.*"),
		))
	})

	Describe("drain name flag", func() {
		It("creates and binds to a user provided service with the given name", func() {
			args := []string{
				"app-name",
				"DogStatsD",
				"--drain-name", "my-drain",
			}

			command.EnableStructuredLogging(cli, args, nil, logger)

			Expect(cli.cliCommandArgs).To(HaveLen(2))
			Expect(cli.cliCommandArgs[0]).To(ConsistOf(
				"create-user-provided-service",
				"my-drain",
				"-l", "prism://DogStatsD",
			))

			Expect(cli.cliCommandArgs[1]).To(ConsistOf(
				"bind-service",
				"app-name",
				"my-drain",
			))
		})
	})

	It("fatally logs if the incorrect number of arguments are given", func() {
		Expect(func() {
			command.EnableStructuredLogging(nil, []string{}, nil, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 0."))

		Expect(func() {
			command.EnableStructuredLogging(nil, []string{"one", "two", "three", "four"}, nil, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 4."))
	})
})

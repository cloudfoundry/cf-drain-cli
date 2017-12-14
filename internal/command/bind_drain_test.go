package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindDrain", func() {
	var (
		logger *stubLogger
		cli    *stubCliConnection
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
	})

	It("calls bind-service with the given app name and service", func() {
		args := []string{"app-name", "drain-name"}

		command.BindDrain(cli, args, logger)

		Expect(cli.cliCommandArgs).To(HaveLen(1))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"bind-service", "app-name", "drain-name",
		}))
	})

	It("fatally logs if it fails to bind to service", func() {
		cli.bindServiceError = errors.New("unable to bind")
		args := []string{"app-name", "drain-name"}

		Expect(func() {
			command.BindDrain(cli, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("unable to bind"))
	})

	It("expects to receive 2 arguments", func() {
		args := []string{"app-name", "drain-name", "extra"}

		Expect(func() {
			command.BindDrain(cli, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 3."))

		args = []string{"app-name"}
		Expect(func() {
			command.BindDrain(cli, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 1."))
	})
})

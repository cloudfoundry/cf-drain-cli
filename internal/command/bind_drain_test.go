package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindDrain", func() {
	var (
		logger       *stubLogger
		cli          *stubCliConnection
		drainFetcher *stubDrainFetcher
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "space-guid"
		drainFetcher = newStubDrainFetcher()
		drainFetcher.drains = []drain.Drain{
			{Name: "drain-name"},
		}
	})

	It("calls bind-service with the given app name and service", func() {
		args := []string{"app-name", "drain-name"}

		command.BindDrain(cli, drainFetcher, args, logger)

		Expect(cli.cliCommandArgs).To(HaveLen(1))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"bind-service", "app-name", "drain-name",
		}))
	})

	It("fatally logs if it fails to bind to service", func() {
		cli.bindServiceError = errors.New("unable to bind")
		args := []string{"app-name", "drain-name"}

		Expect(func() {
			command.BindDrain(cli, drainFetcher, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("unable to bind"))
	})

	It("expects to receive 2 arguments", func() {
		args := []string{"app-name", "drain-name", "extra"}

		Expect(func() {
			command.BindDrain(cli, drainFetcher, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 3."))

		args = []string{"app-name"}
		Expect(func() {
			command.BindDrain(cli, drainFetcher, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2, got 1."))
	})

	It("fatally logs if the drain does not exist", func() {
		args := []string{"app-name", "unknown-drain-name"}

		Expect(func() {
			command.BindDrain(cli, drainFetcher, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("unknown-drain-name is not a valid drain."))
	})

	It("fatally logs if it fails to get existing drains", func() {
		args := []string{"app-name", "drain-name"}
		drainFetcher.err = errors.New("Failed to fetch drains.")

		Expect(func() {
			command.BindDrain(cli, drainFetcher, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Failed to fetch drains."))
	})

	It("fatally logs if it fails to get space guid", func() {
		args := []string{"app-name", "drain-name"}
		cli.currentSpaceError = errors.New("Failed to get space.")

		Expect(func() {
			command.BindDrain(cli, drainFetcher, args, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Failed to get space."))
	})
})

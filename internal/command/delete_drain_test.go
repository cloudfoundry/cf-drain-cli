package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteDrain", func() {
	var (
		cli    *stubCliConnection
		logger *stubLogger
	)

	BeforeEach(func() {
		logger = &stubLogger{}

		cli = newStubCliConnection()
		cli.getServicesName = "my-drain"
		cli.getServicesApps = []string{"app-1", "app-2"}
	})

	It("unbinds and deletes the service", func() {
		command.DeleteDrain(cli, []string{"my-drain"}, logger)

		Expect(cli.cliCommandArgs).To(HaveLen(3))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"unbind-service", "app-1", "my-drain",
		}))
		Expect(cli.cliCommandArgs[1]).To(Equal([]string{
			"unbind-service", "app-2", "my-drain",
		}))
		Expect(cli.cliCommandArgs[2]).To(Equal([]string{
			"delete-service", "my-drain",
		}))
	})

	It("fatally logs with an incorrect number of arguments", func() {
		Expect(func() {
			command.DeleteDrain(cli, []string{}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 0."))

		Expect(func() {
			command.DeleteDrain(cli, []string{"one", "two"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 2."))
	})

	It("fatally logs when the service does not exist", func() {
		Expect(func() {
			command.DeleteDrain(cli, []string{"not-a-service"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Unable to find service not-a-service."))
	})

	It("fatally logs when getting the services fails", func() {
		cli.getServicesError = errors.New("no get services")

		Expect(func() {
			command.DeleteDrain(cli, []string{"my-drain"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("no get services"))
	})

	It("fatally logs when unbinding a service fails", func() {
		cli.unbindServiceError = errors.New("unbind failed")

		Expect(func() {
			command.DeleteDrain(cli, []string{"my-drain"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("unbind failed"))
	})

	It("fatally logs when deleting the service fails", func() {
		cli.deleteServiceError = errors.New("delete failed")

		Expect(func() {
			command.DeleteDrain(cli, []string{"my-drain"}, logger)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("delete failed"))
	})
})

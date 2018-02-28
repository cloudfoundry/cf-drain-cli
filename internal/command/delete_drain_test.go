package command_test

import (
	"bytes"
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteDrain", func() {
	var (
		cli    *stubCliConnection
		logger *stubLogger
		reader *bytes.Buffer
	)

	BeforeEach(func() {
		logger = &stubLogger{}

		cli = newStubCliConnection()
		cli.getServicesName = "my-drain"
		cli.getServicesApps = []string{"app-1", "app-2"}

		reader = bytes.NewBuffer(nil)
	})

	It("unbinds and deletes the service", func() {
		reader.WriteString("y\n")

		command.DeleteDrain(cli, []string{"my-drain"}, logger, reader)

		Expect(logger.printMessages).To(ConsistOf(
			"Are you sure you want to unbind my-drain from app-1, app-2 and delete my-drain? [y/N] ",
		))

		Expect(cli.cliCommandArgs).To(HaveLen(3))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"unbind-service", "app-1", "my-drain",
		}))
		Expect(cli.cliCommandArgs[1]).To(Equal([]string{
			"unbind-service", "app-2", "my-drain",
		}))
		Expect(cli.cliCommandArgs[2]).To(Equal([]string{
			"delete-service", "my-drain", "-f",
		}))
	})

	It("aborts if the user cancels the confirmation", func() {
		reader.WriteString("no\n")

		command.DeleteDrain(cli, []string{"my-drain"}, logger, reader)

		Expect(logger.printMessages).To(ConsistOf(
			"Are you sure you want to unbind my-drain from app-1, app-2 and delete my-drain? [y/N] ",
		))
		Expect(logger.printfMessages).To(ConsistOf(
			"Delete cancelled",
		))

		Expect(cli.cliCommandArgs).To(HaveLen(0))
	})

	It("is not case sensitive with the confirmation", func() {
		reader.WriteString("Y\n")

		command.DeleteDrain(cli, []string{"my-drain"}, logger, reader)

		Expect(logger.printMessages).To(ConsistOf(
			"Are you sure you want to unbind my-drain from app-1, app-2 and delete my-drain? [y/N] ",
		))

		Expect(cli.cliCommandArgs).To(HaveLen(3))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"unbind-service", "app-1", "my-drain",
		}))
		Expect(cli.cliCommandArgs[1]).To(Equal([]string{
			"unbind-service", "app-2", "my-drain",
		}))
		Expect(cli.cliCommandArgs[2]).To(Equal([]string{
			"delete-service", "my-drain", "-f",
		}))
	})

	It("fatally logs with an incorrect number of arguments", func() {
		reader.WriteString("y\n")

		Expect(func() {
			command.DeleteDrain(cli, []string{}, logger, reader)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 0."))

		Expect(func() {
			command.DeleteDrain(cli, []string{"one", "two"}, logger, reader)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 2."))
	})

	It("fatally logs when the service does not exist", func() {
		reader.WriteString("y\n")

		Expect(func() {
			command.DeleteDrain(cli, []string{"not-a-service"}, logger, reader)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("Unable to find service not-a-service."))
	})

	It("fatally logs when getting the services fails", func() {
		reader.WriteString("y\n")

		cli.getServicesError = errors.New("no get services")

		Expect(func() {
			command.DeleteDrain(cli, []string{"my-drain"}, logger, reader)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("no get services"))
	})

	It("fatally logs when unbinding a service fails", func() {
		reader.WriteString("y\n")

		cli.unbindServiceError = errors.New("unbind failed")

		Expect(func() {
			command.DeleteDrain(cli, []string{"my-drain"}, logger, reader)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("unbind failed"))
	})

	It("fatally logs when deleting the service fails", func() {
		reader.WriteString("y\n")

		cli.deleteServiceError = errors.New("delete failed")

		Expect(func() {
			command.DeleteDrain(cli, []string{"my-drain"}, logger, reader)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("delete failed"))
	})
})

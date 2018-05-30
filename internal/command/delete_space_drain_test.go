package command_test

import (
	"bytes"
	"errors"
	"io"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cli/plugin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteSpaceDrain", func() {

	var (
		cli                 *stubCliConnection
		logger              *stubLogger
		reader              *bytes.Buffer
		deleteDrain         *stubDeleteDrain
		serviceDrainFetcher *stubDrainFetcher
	)

	BeforeEach(func() {
		logger = &stubLogger{}

		cli = newStubCliConnection()
		cli.getServicesName = "my-drain"
		cli.getServicesApps = []string{"app-1", "app-2"}

		reader = bytes.NewBuffer(nil)
		deleteDrain = newStubDeleteDrain()
		serviceDrainFetcher = newStubDrainFetcher()
	})

	It("deletes the space drain app", func() {
		// Upper case
		reader.WriteString("Y\n")
		command.DeleteSpaceDrain(cli, []string{"my-drain"}, logger, reader, serviceDrainFetcher, deleteDrain.deleteDrain)

		Expect(cli.getAppName).To(Equal("my-drain"))

		Expect(cli.cliCommandArgs).To(HaveLen(1))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"delete", "my-drain", "-f",
		}))
	})

	It("deletes the space drain app without confirmation", func() {
		command.DeleteSpaceDrain(cli, []string{"my-drain", "--force"}, logger, nil, serviceDrainFetcher, deleteDrain.deleteDrain)

		Expect(cli.cliCommandArgs).To(HaveLen(1))
		Expect(cli.cliCommandArgs[0]).To(Equal([]string{
			"delete", "my-drain", "-f",
		}))
	})

	It("deletes the drain", func() {
		// Lower case
		reader.WriteString("y\n")
		command.DeleteSpaceDrain(cli, []string{"my-drain"}, logger, reader, serviceDrainFetcher, deleteDrain.deleteDrain)

		Expect(deleteDrain.cli).To(Equal(cli))
		Expect(deleteDrain.log).To(Equal(logger))
		Expect(deleteDrain.in).To(BeNil())
		Expect(deleteDrain.serviceDrainFetcher).To(Equal(serviceDrainFetcher))
		Expect(deleteDrain.args).To(Equal([]string{
			"my-drain",
			"--force",
		}))
	})

	It("fatals if the drain name is not provided", func() {
		Expect(func() {
			command.DeleteSpaceDrain(cli, nil, logger, nil, serviceDrainFetcher, deleteDrain.deleteDrain)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 0."))
	})

	It("fatals if given too many arguments", func() {
		Expect(func() {
			command.DeleteSpaceDrain(cli, []string{"a", "b"}, logger, nil, serviceDrainFetcher, deleteDrain.deleteDrain)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 1, got 2."))
	})

	It("fatals if deleting the space drain app fails", func() {
		cli.deleteAppError = errors.New("some-error")
		Expect(func() {
			command.DeleteSpaceDrain(cli, []string{"my-drain", "-f"}, logger, reader, serviceDrainFetcher, deleteDrain.deleteDrain)
		}).To(Panic())
	})

	It("fatals if checkig the existence of the space drain app fails", func() {
		cli.getAppError = errors.New("some-error")
		Expect(func() {
			command.DeleteSpaceDrain(cli, []string{"my-drain", "-f"}, logger, reader, serviceDrainFetcher, deleteDrain.deleteDrain)
		}).To(Panic())
	})

	It("aborts if the user cancels the confirmation", func() {
		reader.WriteString("no\n")

		command.DeleteSpaceDrain(cli, []string{"my-drain"}, logger, reader, serviceDrainFetcher, deleteDrain.deleteDrain)

		Expect(logger.printMessages).To(ConsistOf(
			"Are you sure you want to delete the space drain? [y/N] ",
		))
		Expect(logger.printfMessages).To(ConsistOf(
			"Delete cancelled",
		))

		Expect(cli.cliCommandArgs).To(HaveLen(0))
	})
})

type stubDeleteDrain struct {
	args                []string
	cli                 plugin.CliConnection
	log                 command.Logger
	in                  io.Reader
	serviceDrainFetcher command.DrainFetcher
}

func newStubDeleteDrain() *stubDeleteDrain {
	return &stubDeleteDrain{}
}

func (s *stubDeleteDrain) deleteDrain(cli plugin.CliConnection, args []string, log command.Logger, in io.Reader, serviceDrainFetcher command.DrainFetcher) {
	s.args = args
	s.cli = cli
	s.log = log
	s.in = in
	s.serviceDrainFetcher = serviceDrainFetcher
}

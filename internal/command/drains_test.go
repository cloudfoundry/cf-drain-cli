package command_test

import (
	"bytes"
	"errors"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Drains", func() {
	var (
		logger       *stubLogger
		cli          *stubCliConnection
		drainFetcher *stubDrainFetcher
		tableWriter  *bytes.Buffer
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		drainFetcher = newStubDrainFetcher()
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "my-space-guid"
		tableWriter = bytes.NewBuffer(nil)
	})

	It("writes the headers", func() {
		command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)

		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"name      bound apps  type",
			"",
		}))
	})

	It("writes the drain name in the first column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{Name: "drain-1"},
			{Name: "drain-2"},
		}
		command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)

		// Header + 2 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"name      bound apps  type",
			"drain-1               ",
			"drain-2               ",
			"",
		}))
	})

	It("writes the app guid in the second column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{Name: "drain-1", Apps: []string{"app-1", "app-2"}},
			{Name: "drain-2", Apps: []string{"app-1"}},
		}
		command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)

		// Header + 2 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"name      bound apps    type",
			"drain-1   app-1, app-2  ",
			"drain-2   app-1         ",
			"",
		}))
	})

	It("writes the drain type in the third column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{Name: "drain-1", Apps: []string{"app-1", "app-2"}, Type: "metrics"},
			{Name: "drain-2", Apps: []string{"app-1"}, Type: "logs"},
		}
		command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)

		// Header + 2 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"name      bound apps    type",
			"drain-1   app-1, app-2  metrics",
			"drain-2   app-1         logs",
			"",
		}))
	})

	It("fatally logs when failing to get current space", func() {
		cli.currentSpaceError = errors.New("no space error")

		Expect(func() {
			command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("no space error"))
	})

	It("fatally logs when failing to fetch drains", func() {
		drainFetcher.err = errors.New("omg error")

		Expect(func() {
			command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Failed to fetch drains: omg error"))
	})
})

type stubDrainFetcher struct {
	drains []cloudcontroller.Drain
	err    error
}

func newStubDrainFetcher() *stubDrainFetcher {
	return &stubDrainFetcher{}
}

func (f *stubDrainFetcher) Drains(spaceGuid string) ([]cloudcontroller.Drain, error) {
	return f.drains, f.err
}

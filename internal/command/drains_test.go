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

	It("writes the drain type in the third column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{
				Name:     "drain-1",
				Apps:     []string{"app-1", "app-2"},
				Type:     "metrics",
				DrainURL: "syslog://my-drain:1233",
			},
			{
				Name:     "drain-2",
				Apps:     []string{"app-1"},
				Type:     "logs",
				DrainURL: "syslog-tls://my-drain:1234",
			},
		}
		command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)

		// Header + 2 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"App       Drain     Type      URL",
			"app-1     drain-1   Metrics   syslog://my-drain:1233",
			"app-2     drain-1   Metrics   syslog://my-drain:1233",
			"app-1     drain-2   Logs      syslog-tls://my-drain:1234",
			"",
		}))
	})

	It("sanitizes drain urls", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{
				Name:     "drain-1",
				Apps:     []string{"app-1", "app-2"},
				Type:     "metrics",
				DrainURL: "syslog://username:password@my-drain:1233?some-query=secret&drain-type=metrics",
			},
		}
		command.Drains(cli, drainFetcher, []string{}, logger, tableWriter)

		// Header + 2 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"App       Drain     Type      URL",
			"app-1     drain-1   Metrics   syslog://<redacted>:<redacted>@my-drain:1233?some-query=<redacted>",
			"app-2     drain-1   Metrics   syslog://<redacted>:<redacted>@my-drain:1233?some-query=<redacted>",
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

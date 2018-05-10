package command_test

import (
	"bytes"
	"errors"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Drains", func() {
	var (
		logger                               *stubLogger
		cli                                  *stubCliConnection
		drainFetchers                        []command.DrainFetcher
		appDrainFetcher, serviceDrainFetcher *stubDrainFetcher
		tableWriter                          *bytes.Buffer
	)

	var _ command.DrainFetcher = newStubDrainFetcher()

	BeforeEach(func() {
		logger = &stubLogger{}
		serviceDrainFetcher = newStubDrainFetcher()
		appDrainFetcher = newStubDrainFetcher()
		drainFetchers = []command.DrainFetcher{serviceDrainFetcher, appDrainFetcher}
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "my-space-guid"
		tableWriter = bytes.NewBuffer(nil)
	})

	It("reports the drain type and adapter type", func() {
		serviceDrainFetcher.drains = []drain.Drain{
			{
				Name:        "drain-1",
				Apps:        []string{"app-1", "app-2"},
				Type:        "metrics",
				DrainURL:    "syslog://my-drain:1233",
				AdapterType: "service",
			},
			{
				Name:        "drain-2",
				Apps:        []string{"app-1"},
				Type:        "logs",
				DrainURL:    "syslog-tls://my-drain:1234",
				AdapterType: "service",
			},
		}
		appDrainFetcher.drains = []drain.Drain{
			{
				Name:        "drain-3",
				Apps:        []string{"app-1"},
				Type:        "all",
				DrainURL:    "https://my-drain:1235",
				AdapterType: "application",
			},
		}

		command.Drains(cli, []string{}, logger, tableWriter, drainFetchers...)

		// Header + 3 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"App       Drain     Type      URL                         AdapterType",
			"app-1     drain-1   Metrics   syslog://my-drain:1233      service",
			"app-2     drain-1   Metrics   syslog://my-drain:1233      service",
			"app-1     drain-2   Logs      syslog-tls://my-drain:1234  service",
			"app-1     drain-3   All       https://my-drain:1235       application",
			"",
		}))
	})

	It("sanitizes drain urls", func() {
		serviceDrainFetcher.drains = []drain.Drain{
			{
				Name:        "drain-1",
				Apps:        []string{"app-1", "app-2"},
				Type:        "metrics",
				DrainURL:    "syslog://username:password@my-drain:1233?some-query=secret&drain-type=metrics",
				AdapterType: "service",
			},
		}
		command.Drains(cli, []string{}, logger, tableWriter, drainFetchers...)

		// Header + 2 drains
		Expect(strings.Split(tableWriter.String(), "\n")).To(Equal([]string{
			"App       Drain     Type      URL                                                                 AdapterType",
			"app-1     drain-1   Metrics   syslog://<redacted>:<redacted>@my-drain:1233?some-query=<redacted>  service",
			"app-2     drain-1   Metrics   syslog://<redacted>:<redacted>@my-drain:1233?some-query=<redacted>  service",
			"",
		}))
	})

	It("fatally logs when failing to get current space", func() {
		cli.currentSpaceError = errors.New("no space error")

		Expect(func() {
			command.Drains(cli, []string{}, logger, tableWriter, drainFetchers...)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("no space error"))
	})

	It("fatally logs when failing to fetch drains", func() {
		serviceDrainFetcher.err = errors.New("omg error")

		Expect(func() {
			command.Drains(cli, []string{}, logger, tableWriter, drainFetchers...)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Failed to fetch drains: omg error"))
	})
})

type stubDrainFetcher struct {
	drains []drain.Drain
	err    error
}

func newStubDrainFetcher() *stubDrainFetcher {
	return &stubDrainFetcher{}
}

func (f *stubDrainFetcher) Drains(spaceGuid string) ([]drain.Drain, error) {
	return f.drains, f.err
}

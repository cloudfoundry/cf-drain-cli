package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Drains", func() {
	var (
		logger       *stubLogger
		cli          *stubCliConnection
		ccClient     *stubCloudControllerClient
		drainFetcher *stubDrainFetcher
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		drainFetcher = newStubDrainFetcher()
		ccClient = newStubCloudControllerClient()
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "my-space-guid"
	})

	It("writes the headers", func() {
		command.Drains(cli, drainFetcher, []string{}, logger)

		Expect(logger.printfMessages).To(HaveLen(1))
		Expect(logger.printfMessages[0]).To(MatchRegexp(`\Aname\s+bound apps\s+type`))
	})

	It("writes the drain name in the first column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{Name: "drain-1"},
			{Name: "drain-2"},
		}
		command.Drains(cli, drainFetcher, []string{}, logger)

		// Header + 2 drains
		Expect(logger.printfMessages).To(HaveLen(3))
		Expect(logger.printfMessages[1]).To(MatchRegexp(`\Adrain-1`))
		Expect(logger.printfMessages[2]).To(MatchRegexp(`\Adrain-2`))
	})

	It("writes the app guid in the second column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{Name: "drain-1", Apps: []string{"app-1", "app-2"}},
			{Name: "drain-2", Apps: []string{"app-1"}},
		}
		command.Drains(cli, drainFetcher, []string{}, logger)

		// Header + 2 drains
		Expect(logger.printfMessages).To(HaveLen(3))
		Expect(logger.printfMessages[1]).To(MatchRegexp(`\Adrain-1\s+app-1,\s+app-2`))
		Expect(logger.printfMessages[2]).To(MatchRegexp(`\Adrain-2\s+app-1`))
	})

	It("writes the drain type in the third column", func() {
		drainFetcher.drains = []cloudcontroller.Drain{
			{Name: "drain-1", Apps: []string{"app-1", "app-2"}, Type: "metrics"},
			{Name: "drain-2", Apps: []string{"app-1"}, Type: "logs"},
		}
		command.Drains(cli, drainFetcher, []string{}, logger)

		// Header + 2 drains
		Expect(logger.printfMessages).To(HaveLen(3))
		Expect(logger.printfMessages[1]).To(MatchRegexp(`\Adrain-1\s+app-1,\s+app-2\s+metrics`))
		Expect(logger.printfMessages[2]).To(MatchRegexp(`\Adrain-2\s+app-1\s+logs`))
	})

	It("fatally logs when failing to get current space", func() {
		cli.currentSpaceError = errors.New("no space error")

		Expect(func() {
			command.Drains(cli, drainFetcher, []string{}, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("no space error"))
	})

	It("fatally logs when failing to fetch drains", func() {
		drainFetcher.err = errors.New("omg error")

		Expect(func() {
			command.Drains(cli, drainFetcher, []string{}, logger)
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

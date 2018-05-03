package drain_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
)

var _ = Describe("ApplicationSpaceDrains", func() {

	var (
		appLister   *spyAppLister
		envProvider *stubEnvProvider
		drainLister drain.ApplicationDrainLister
	)

	BeforeEach(func() {
		appLister = newSpyAppLister()
		envProvider = newStubEnvProvider()

		appLister.apps = []cloudcontroller.App{
			cloudcontroller.App{
				Name: "space-forwarder-11111111-1111-1111-1111-111111111111",
				Guid: "11111111-1111-1111-1111-111111111111",
			},
			cloudcontroller.App{
				Name: "app-1",
				Guid: "22222222-2222-2222-2222-222222222222",
			},
			cloudcontroller.App{
				Name: "app-2",
				Guid: "33333333-3333-3333-3333-333333333333",
			},
		}
		envProvider.envs = map[string]string{
			"DRAIN_TYPE": "all",
			"DRAIN_URL":  "https://the-syslog-drain.com",
		}

		drainLister = drain.NewApplicationDrainLister(appLister, envProvider)

	})

	It("returns application space drains", func() {
		drains, err := drainLister.Drains("space-guid")
		Expect(err).ToNot(HaveOccurred())

		Expect(drains).To(HaveLen(1))

		Expect(drains[0]).To(Equal(
			drain.Drain{
				Apps: []string{
					"space-forwarder-11111111-1111-1111-1111-111111111111",
					"app-1",
					"app-2",
				},
				AppGuids: []string{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
					"33333333-3333-3333-3333-333333333333",
				},
				Type:        "all",
				DrainURL:    "https://the-syslog-drain.com",
				AdapterType: "application",
			},
		))
	})
	It("logs an error when it fails to retrieve environment variables", func() {
		e := errors.New("some err")
		envProvider.err = e

		_, err := drainLister.Drains("space-guid")

		Expect(err).To(HaveOccurred())
	})
})

func newSpyAppLister() *spyAppLister {
	return &spyAppLister{}
}

type spyAppLister struct {
	apps           []cloudcontroller.App
	requestedSpace string
}

func (s *spyAppLister) ListApps(spaceGuid string) ([]cloudcontroller.App, error) {
	s.requestedSpace = spaceGuid
	return s.apps, nil
}

func newStubEnvProvider() *stubEnvProvider {
	return &stubEnvProvider{}
}

type stubEnvProvider struct {
	envs map[string]string
	err  error
}

func (e *stubEnvProvider) EnvVars(appGuid string) (map[string]string, error) {
	return e.envs, e.err
}

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
		envProvider *spyEnvProvider
		drainLister drain.ApplicationDrainLister
	)

	BeforeEach(func() {
		appLister = newSpyAppLister()
		envProvider = newSpyEnvProvider()

		drainLister = drain.NewApplicationDrainLister(appLister, envProvider)
	})

	It("returns all application drains", func() {
		appLister.apps = []cloudcontroller.App{
			cloudcontroller.App{
				Name: "xxxxxxxx-00000000-0000-0000-0000-000000000000",
				Guid: "00000000-0000-0000-0000-000000000000",
			},
			cloudcontroller.App{
				Name: "yyyyyyyyyyyyyyy-11111111-1111-1111-1111-111111111111",
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

		envProvider.envs = map[string]map[string]string{
			"00000000-0000-0000-0000-000000000000": {
				"DRAIN_SCOPE": "single",
				"SOURCE_ID":   "22222222-2222-2222-2222-222222222222",
				"DRAIN_TYPE":  "logs",
				"SYSLOG_URL":  "syslog://the-syslog-drain.com",
			},
			"11111111-1111-1111-1111-111111111111": {
				"DRAIN_SCOPE": "space",
				"DRAIN_TYPE":  "all",
				"DRAIN_URL":   "https://the-syslog-drain.com",
			},
		}

		drains, err := drainLister.Drains("space-guid")
		Expect(err).ToNot(HaveOccurred())

		Expect(drains).To(HaveLen(2))

		Expect(drains).To(ConsistOf(
			[]drain.Drain{
				drain.Drain{
					Name: "xxxxxxxx-00000000-0000-0000-0000-000000000000",
					Guid: "00000000-0000-0000-0000-000000000000",
					Apps: []string{
						"app-1",
					},
					AppGuids: []string{
						"22222222-2222-2222-2222-222222222222",
					},
					Type:        "logs",
					DrainURL:    "syslog://the-syslog-drain.com",
					AdapterType: "application",
				},
				drain.Drain{
					Name: "yyyyyyyyyyyyyyy-11111111-1111-1111-1111-111111111111",
					Guid: "11111111-1111-1111-1111-111111111111",
					Apps: []string{
						"xxxxxxxx-00000000-0000-0000-0000-000000000000",
						"yyyyyyyyyyyyyyy-11111111-1111-1111-1111-111111111111",
						"app-1",
						"app-2",
					},
					AppGuids: []string{
						"00000000-0000-0000-0000-000000000000",
						"11111111-1111-1111-1111-111111111111",
						"22222222-2222-2222-2222-222222222222",
						"33333333-3333-3333-3333-333333333333",
					},
					Type:        "all",
					DrainURL:    "https://the-syslog-drain.com",
					AdapterType: "application",
				},
			},
		))
	})

	Context("when syslog forwarder app does not contain env var SOURCE_ID", func() {
		It("does not report that drain ", func() {
			appLister.apps = []cloudcontroller.App{
				cloudcontroller.App{
					Name: "cf-drain-00000000-0000-0000-0000-000000000000",
					Guid: "00000000-0000-0000-0000-000000000000",
				},
				cloudcontroller.App{
					Name: "space-forwarder-11111111-1111-1111-1111-111111111111",
					Guid: "11111111-1111-1111-1111-111111111111",
				},
				cloudcontroller.App{
					Name: "app-1",
					Guid: "22222222-2222-2222-2222-222222222222",
				},
			}

			envProvider.envs = map[string]map[string]string{
				"00000000-0000-0000-0000-000000000000": {
					"DRAIN_TYPE": "logs",
					"SYSLOG_URL": "syslog://the-syslog-drain.com",
				},
				"11111111-1111-1111-1111-111111111111": {
					"DRAIN_SCOPE": "space",
					"DRAIN_TYPE":  "all",
					"DRAIN_URL":   "https://the-syslog-drain.com",
				},
			}

			drains, err := drainLister.Drains("space-guid")
			Expect(err).ToNot(HaveOccurred())

			Expect(drains).To(HaveLen(1))

			Expect(drains).To(ConsistOf(
				drain.Drain{
					Name: "space-forwarder-11111111-1111-1111-1111-111111111111",
					Guid: "11111111-1111-1111-1111-111111111111",
					Apps: []string{
						"cf-drain-00000000-0000-0000-0000-000000000000",
						"space-forwarder-11111111-1111-1111-1111-111111111111",
						"app-1",
					},
					AppGuids: []string{
						"00000000-0000-0000-0000-000000000000",
						"11111111-1111-1111-1111-111111111111",
						"22222222-2222-2222-2222-222222222222",
					},
					Type:        "all",
					DrainURL:    "https://the-syslog-drain.com",
					AdapterType: "application",
				},
			))
		})
	})

	It("logs an error when it fails to retrieve environment variables", func() {
		appLister.apps = []cloudcontroller.App{
			cloudcontroller.App{
				Name: "space-forwarder-00000000-0000-0000-0000-000000000000",
				Guid: "00000000-0000-0000-0000-000000000000",
			},
		}
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

func newSpyEnvProvider() *spyEnvProvider {
	return &spyEnvProvider{}
}

type spyEnvProvider struct {
	envs map[string]map[string]string
	err  error
}

func (e *spyEnvProvider) EnvVars(appGuid string) (map[string]string, error) {
	return e.envs[appGuid], e.err
}

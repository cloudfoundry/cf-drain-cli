package command_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
)

var _ = Describe("PushSpaceDrain", func() {
	var (
		logger              *stubLogger
		cli                 *stubCliConnection
		refreshTokenFetcher *stubRefreshTokenFetcher
		guidFetcher         func() string
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
		cli.apiEndpoint = "https://api.something.com"
		cli.currentSpaceName = "space"
		cli.currentOrgName = "org"
		cli.getServiceGuid = "service-guid"
		cli.sslDisabled = true

		refreshTokenFetcher = newStubRefreshTokenFetcher()
		refreshTokenFetcher.token = "refresh-token"

		guidFetcher = func() string { return "test-group" }
	})

	It("pushes app from the given space-drain zip file", func() {
		command.PushServiceDrain(
			cli,
			[]string{
				"service-name",
				"https://syslog-drain",
				"--path", "service-drain-zip",
			},
			refreshTokenFetcher,
			logger,
			guidFetcher,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "service-name-forwarder",
				"-p", "service-drain-zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "service-name-forwarder", "SOURCE_ID", "service-guid"},
			[]string{"set-env", "service-name-forwarder", "SOURCE_HOSTNAME", "org.space.service-name-forwarder"},
			[]string{"set-env", "service-name-forwarder", "CLIENT_ID", "cf"},
			[]string{"set-env", "service-name-forwarder", "REFRESH_TOKEN", "refresh-token"},
			[]string{"set-env", "service-name-forwarder", "CACHE_SIZE", "0"},
			[]string{"set-env", "service-name-forwarder", "SKIP_CERT_VERIFY", "true"},
			[]string{"set-env", "service-name-forwarder", "GROUP_NAME", "test-group"},
			[]string{"set-env", "service-name-forwarder", "SYSLOG_URL", "https://syslog-drain"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "service-name-forwarder",
			},
		))
	})

	DescribeTable("fatally logs if interactions with the plugin fails", func(setup func(), msg string) {
		setup()

		Expect(func() {
			command.PushServiceDrain(
				cli,
				[]string{
					"service-name",
					"https://some-drain",
					"--path", "some-temp-dir",
				},
				refreshTokenFetcher,
				logger,
				guidFetcher,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).Should(Equal(msg))
	},
		Entry("service fails", func() { cli.getServiceError = errors.New("some-error") }, "some-error"),
		Entry("skip SSL fails", func() { cli.sslDisabledError = errors.New("some-error") }, "some-error"),
		Entry("current space fails", func() { cli.currentSpaceError = errors.New("some-error") }, "some-error"),
		Entry("current org fails", func() { cli.currentOrgError = errors.New("some-error") }, "some-error"),
		Entry("cli command fails", func() { cli.pushAppError = errors.New("some-error") }, "some-error"),
		Entry("refresh token fails", func() { refreshTokenFetcher.err = errors.New("some-error") }, "some-error"),
	)

	It("fatally logs if there are extra command line arguments", func() {
		Expect(func() {
			command.PushServiceDrain(
				cli,
				[]string{
					"service-name",
					"https://some-drain",
					"--path", "some-temp-dir",
					"some-unknown-arg",
				},
				refreshTokenFetcher,
				logger,
				guidFetcher,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 2 got 3."))
	})

	DescribeTable("fatally logs when number of args is wrong", func(args []string, len int) {
		Expect(func() {
			command.PushServiceDrain(
				cli,
				args,
				refreshTokenFetcher,
				logger,
				guidFetcher,
			)
		}).To(Panic())

		msg := fmt.Sprintf("Invalid arguments, expected 2 got %d.", len)
		Expect(logger.fatalfMessage).Should(Equal(msg))
	},
		Entry("too many", []string{
			"service-name",
			"https://some-drain",
			"--path",
			"some-temp-dir",
			"some-unknown-arg",
		}, 3),
		Entry("too few", []string{}, 0),
	)
})

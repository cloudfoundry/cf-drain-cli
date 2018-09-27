package command_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
)

var _ = Describe("PushSpaceServiceDrain", func() {
	var (
		logger              *stubLogger
		cli                 *stubCliConnection
		downloader          *stubDownloader
		refreshTokenFetcher *stubRefreshTokenFetcher
		groupNameProvider   func() string
		guidProvider        func() string
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
		cli.apiEndpoint = "https://api.something.com"
		cli.currentSpaceName = "space"
		cli.currentOrgName = "org"
		cli.getServiceGuid = "service-guid"
		cli.sslDisabled = true

		downloader = newStubDownloader()
		downloader.path = "/downloaded/temp/dir/forwarder.zip"

		refreshTokenFetcher = newStubRefreshTokenFetcher()
		refreshTokenFetcher.token = "refresh-token"

		groupNameProvider = func() string { return "test-group" }
		guidProvider = func() string { return "a-guid" }
	})

	It("pushes app from the given space-drain zip file", func() {
		command.PushSpaceServiceDrain(
			cli,
			[]string{
				"https://syslog-drain",
				"--path", "service-drain-zip",
			},
			downloader,
			refreshTokenFetcher,
			logger,
			groupNameProvider,
			guidProvider,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "space-services-forwarder-a-guid",
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
			[]string{"set-env", "space-services-forwarder-a-guid", "SOURCE_HOSTNAME", "org.space"},
			[]string{"set-env", "space-services-forwarder-a-guid", "INCLUDE_SERVICES", "true"},
			[]string{"set-env", "space-services-forwarder-a-guid", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-services-forwarder-a-guid", "REFRESH_TOKEN", "refresh-token"},
			[]string{"set-env", "space-services-forwarder-a-guid", "CACHE_SIZE", "0"},
			[]string{"set-env", "space-services-forwarder-a-guid", "SKIP_CERT_VERIFY", "true"},
			[]string{"set-env", "space-services-forwarder-a-guid", "GROUP_NAME", "test-group"},
			[]string{"set-env", "space-services-forwarder-a-guid", "SYSLOG_URL", "https://syslog-drain"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-services-forwarder-a-guid",
			},
		))
	})

	It("allows the user to name their forwarder app", func() {
		command.PushSpaceServiceDrain(
			cli,
			[]string{
				"https://syslog-drain",
				"--path", "service-drain-zip",
				"--name", "forwarder-name",
			},
			downloader,
			refreshTokenFetcher,
			logger,
			groupNameProvider,
			guidProvider,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "forwarder-name",
				"-p", "service-drain-zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))
	})

	It("downloads the app before pushing app", func() {
		command.PushSpaceServiceDrain(
			cli,
			[]string{
				"https://some-drain",
			},
			downloader,
			refreshTokenFetcher,
			logger,
			groupNameProvider,
			guidProvider,
		)

		Expect(downloader.assetName).To(Equal("forwarder.zip"))

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "space-services-forwarder-a-guid",
				"-p", "/downloaded/temp/dir/forwarder.zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))
	})

	It("pushes downloaded app", func() {
		command.PushSpaceServiceDrain(
			cli,
			[]string{
				"https://some-drain",
			},
			downloader,
			refreshTokenFetcher,
			logger,
			groupNameProvider,
			guidProvider,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "space-services-forwarder-a-guid",
				"-p", "/downloaded/temp/dir/forwarder.zip",
				"-i", "3",
				"-b", "binary_buildpack",
				"-c", "./run.sh",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(downloader.assetName).To(Equal("forwarder.zip"))
		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "space-services-forwarder-a-guid", "SOURCE_HOSTNAME", "org.space"},
			[]string{"set-env", "space-services-forwarder-a-guid", "INCLUDE_SERVICES", "true"},
			[]string{"set-env", "space-services-forwarder-a-guid", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-services-forwarder-a-guid", "REFRESH_TOKEN", "refresh-token"},
			[]string{"set-env", "space-services-forwarder-a-guid", "CACHE_SIZE", "0"},
			[]string{"set-env", "space-services-forwarder-a-guid", "SKIP_CERT_VERIFY", "true"},
			[]string{"set-env", "space-services-forwarder-a-guid", "GROUP_NAME", "test-group"},
			[]string{"set-env", "space-services-forwarder-a-guid", "SYSLOG_URL", "https://some-drain"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-services-forwarder-a-guid",
			},
		))
	})

	DescribeTable("fatally logs if interactions with the plugin fails", func(setup func(), msg string) {
		setup()

		Expect(func() {
			command.PushSpaceServiceDrain(
				cli,
				[]string{
					"https://some-drain",
					"--path", "some-temp-dir",
				},
				downloader,
				refreshTokenFetcher,
				logger,
				groupNameProvider,
				guidProvider,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).Should(Equal(msg))
	},
		Entry("skip SSL fails", func() { cli.sslDisabledError = errors.New("some-error") }, "some-error"),
		Entry("current space fails", func() { cli.currentSpaceError = errors.New("some-error") }, "some-error"),
		Entry("current org fails", func() { cli.currentOrgError = errors.New("some-error") }, "some-error"),
		Entry("cli command fails", func() { cli.pushAppError = errors.New("some-error") }, "some-error"),
		Entry("refresh token fails", func() { refreshTokenFetcher.err = errors.New("some-error") }, "some-error"),
	)

	DescribeTable("fatally logs when number of args is wrong", func(args []string, len int) {
		Expect(func() {
			command.PushSpaceServiceDrain(
				cli,
				args,
				downloader,
				refreshTokenFetcher,
				logger,
				groupNameProvider,
				guidProvider,
			)
		}).To(Panic())

		msg := fmt.Sprintf("Invalid arguments, expected 1 got %d.", len)
		Expect(logger.fatalfMessage).Should(Equal(msg))
	},
		Entry("too many", []string{
			"https://some-drain",
			"--path",
			"some-temp-dir",
			"some-unknown-arg",
		}, 2),
		Entry("too few", []string{}, 0),
	)
})

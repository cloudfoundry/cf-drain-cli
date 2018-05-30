package command_test

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
)

var _ = Describe("PushSpaceDrain", func() {
	var (
		logger              *stubLogger
		cli                 *stubCliConnection
		downloader          *stubDownloader
		refreshTokenFetcher *stubRefreshTokenFetcher
		reader              *strings.Reader
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "space-guid"
		cli.apiEndpoint = "https://api.something.com"
		downloader = newStubDownloader()
		downloader.path = "/downloaded/temp/dir/space_drain"
		reader = strings.NewReader("y\n")

		refreshTokenFetcher = newStubRefreshTokenFetcher()
		refreshTokenFetcher.token = "some-refresh-token"
	})

	It("pushes app from the given space-drain directory", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
			},
			downloader,
			refreshTokenFetcher,
			logger,
		)

		Expect(logger.printMessages).To(ConsistOf(
			"The space drain functionality is an experimental feature. " +
				"See https://github.com/cloudfoundry/cf-drain-cli#space-drain-experimental for more details.\n" +
				"Do you wish to proceed? [y/N] ",
		))

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "some-drain",
				"-p", "some-temp-dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "some-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "some-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "some-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "some-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "some-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "some-drain", "REFRESH_TOKEN", "some-refresh-token"},
			[]string{"set-env", "some-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "some-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "some-drain",
			},
		))
	})

	It("downloads the app before pushing app from the given space-drain directory", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			[]string{
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
			},
			downloader,
			refreshTokenFetcher,
			logger,
		)

		Expect(downloader.assetName).To(Equal("space_drain"))

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "some-drain",
				"-p", "/downloaded/temp/dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))
	})

	It("accepts capital Y for warning prompt", func() {
		reader = strings.NewReader("Y\n")
		command.PushSpaceDrain(
			cli,
			reader,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
			},
			downloader,
			refreshTokenFetcher,
			logger,
		)

		Expect(logger.printMessages).To(ConsistOf(
			"The space drain functionality is an experimental feature. " +
				"See https://github.com/cloudfoundry/cf-drain-cli#space-drain-experimental for more details.\n" +
				"Do you wish to proceed? [y/N] ",
		))

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "some-drain",
				"-p", "some-temp-dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "some-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "some-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "some-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "some-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "some-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "some-drain", "REFRESH_TOKEN", "some-refresh-token"},
			[]string{"set-env", "some-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "some-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "some-drain",
			},
		))
	})

	It("does not show warning prompt with --force flag", func() {
		command.PushSpaceDrain(
			cli,
			nil,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
				"--force",
			},
			downloader,
			refreshTokenFetcher,
			logger,
		)

		Expect(logger.printMessages).To(BeEmpty())

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "some-drain",
				"-p", "some-temp-dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "some-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "some-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "some-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "some-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "some-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "some-drain", "REFRESH_TOKEN", "some-refresh-token"},
			[]string{"set-env", "some-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "some-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "some-drain",
			},
		))
	})

	It("pushes downloaded app", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			[]string{
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
			},
			downloader,
			refreshTokenFetcher,
			logger,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "some-drain",
				"-p", "/downloaded/temp/dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(downloader.assetName).To(Equal("space_drain"))
		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "some-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "some-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "some-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "some-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "some-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "some-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "some-drain", "REFRESH_TOKEN", "some-refresh-token"},
			[]string{"set-env", "some-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "some-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "some-drain",
			},
		))
	})

	It("defaults to space-drain if the drain-name is not provided", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			[]string{
				"--path", "some-temp-dir",
				"--drain-url", "https://some-drain",
				"--force",
			},
			downloader,
			refreshTokenFetcher,
			logger,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "space-drain",
				"-p", "some-temp-dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ContainElement(
			[]string{"set-env", "space-drain", "DRAIN_NAME", "space-drain"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-drain",
			},
		))
	})

	DescribeTable("fatally logs if setting env variables fails", func(env string) {
		cli.setEnvErrors[env] = errors.New("some-error")

		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).Should(Equal("some-error"))
	},
		Entry("SPACE_ID", "SPACE_ID"),
		Entry("DRAIN_NAME", "DRAIN_NAME"),
		Entry("DRAIN_URL", "DRAIN_URL"),
		Entry("API_ADDR", "API_ADDR"),
		Entry("UAA_ADDR", "UAA_ADDR"),
		Entry("CLIENT_ID", "CLIENT_ID"),
		Entry("REFRESH_TOKEN", "REFRESH_TOKEN"),
		Entry("SKIP_CERT_VERIFY", "SKIP_CERT_VERIFY"),
		Entry("DRAIN_SCOPE", "DRAIN_SCOPE"),
	)

	It("fatally logs if confirmation is given anything other than y", func() {
		reader = strings.NewReader("no\n")

		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("OK, exiting."))
	})

	It("fatally logs if fetching the space fails", func() {
		cli.currentSpaceError = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	It("fatally logs if fetching the api endpoint fails", func() {
		cli.apiEndpointError = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	It("fatally logs if fetching the refresh token fails", func() {
		refreshTokenFetcher.err = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	It("fatally logs if the push fails", func() {
		cli.pushAppError = errors.New("failed to push")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("failed to push"))
	})

	It("fatally logs if the space-drain drain-url is not provided", func() {
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("the required flag `--drain-url' was not specified"))
	})

	It("fatally logs if there are extra command line arguments", func() {
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"some-unknown-arg",
				},
				downloader,
				refreshTokenFetcher,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 0, got 1."))
	})
})

type stubRefreshTokenFetcher struct {
	token string
	err   error
}

func newStubRefreshTokenFetcher() *stubRefreshTokenFetcher {
	return &stubRefreshTokenFetcher{}
}

func (s *stubRefreshTokenFetcher) RefreshToken() (string, error) {
	return s.token, s.err
}

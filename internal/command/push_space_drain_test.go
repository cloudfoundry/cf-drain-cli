package command_test

import (
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
)

var _ = Describe("PushSpaceDrain", func() {
	var (
		logger     *stubLogger
		cli        *stubCliConnection
		downloader *stubDownloader
		reader     *strings.Reader
		pr         func(int) ([]byte, error)
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "space-guid"
		cli.apiEndpoint = "https://api.something.com"
		downloader = newStubDownloader()
		downloader.path = "/downloaded/temp/dir/space_drain"
		reader = strings.NewReader("y\n")
		pr = func(int) ([]byte, error) {
			return []byte("some-password"), nil
		}
	})

	It("pushes app from the given space-drain directory", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			nil,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
				"--username", "some-user",
				"--password", "some-given-password",
			},
			downloader,
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
				"push", "space-drain",
				"-p", "some-temp-dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "space-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "space-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "space-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "space-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "space-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-drain", "USERNAME", "some-user"},
			[]string{"set-env", "space-drain", "PASSWORD", "some-given-password"},
			[]string{"set-env", "space-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "space-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-drain",
			},
		))
	})

	It("downloads the app before pushing app from the given space-drain directory", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			pr,
			[]string{
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
				"--username", "some-user",
			},
			downloader,
			logger,
		)

		Expect(downloader.assetName).To(Equal("space_drain"))

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "space-drain",
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
			pr,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
				"--username", "some-user",
			},
			downloader,
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
				"push", "space-drain",
				"-p", "some-temp-dir",
				"-b", "binary_buildpack",
				"-c", "./space_drain",
				"--health-check-type", "process",
				"--no-start",
				"--no-route",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "space-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "space-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "space-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "space-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "space-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-drain", "USERNAME", "some-user"},
			[]string{"set-env", "space-drain", "PASSWORD", "some-password"},
			[]string{"set-env", "space-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "space-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-drain",
			},
		))
	})

	It("does not show warning prompt with --force flag", func() {
		command.PushSpaceDrain(
			cli,
			nil,
			pr,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
				"--username", "some-user",
				"--force",
			},
			downloader,
			logger,
		)

		Expect(logger.printMessages).To(BeEmpty())

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

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "space-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "space-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "space-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "space-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "space-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-drain", "USERNAME", "some-user"},
			[]string{"set-env", "space-drain", "PASSWORD", "some-password"},
			[]string{"set-env", "space-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "space-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-drain",
			},
		))
	})

	It("pushes downloaded app", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			pr,
			[]string{
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--type", "metrics",
				"--username", "some-user",
			},
			downloader,
			logger,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(2))
		Expect(cli.cliCommandArgs[0]).To(Equal(
			[]string{
				"push", "space-drain",
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
			[]string{"set-env", "space-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "space-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_TYPE", "metrics"},
			[]string{"set-env", "space-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "space-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "space-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-drain", "USERNAME", "some-user"},
			[]string{"set-env", "space-drain", "PASSWORD", "some-password"},
			[]string{"set-env", "space-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "space-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[1]).To(Equal(
			[]string{
				"start", "space-drain",
			},
		))
	})

	It("prompts for password if --username is provided", func() {
		command.PushSpaceDrain(
			cli,
			reader,
			func(int) ([]byte, error) { return []byte("user-provided-password"), nil },
			[]string{
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--username", "some-user",
			},
			downloader,
			logger,
		)

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ContainElement(
			[]string{"set-env", "space-drain", "PASSWORD", "user-provided-password"},
		))
	})

	It("creates a user if username is not provided", func() {
		guid := "12345678-1234-1234-1234-123456789abc"
		cli.getAppGuid = guid
		cli.currentSpaceName = "SPACE"
		cli.currentOrgName = "ORG"

		command.PushSpaceDrain(
			cli,
			reader,
			pr,
			[]string{
				"--path", "some-temp-dir",
				"--drain-name", "some-drain",
				"--drain-url", "https://some-drain",
				"--force",
			},
			downloader,
			logger,
		)

		Expect(cli.cliCommandArgs).To(HaveLen(4))
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

		Expect(cli.cliCommandArgs[1]).To(HaveLen(3))
		Expect(cli.cliCommandArgs[1][0]).To(Equal("create-user"))
		Expect(cli.cliCommandArgs[1][1]).To(Equal(fmt.Sprintf("space-drain-%s", guid)))
		Expect(cli.cliCommandArgs[1][2]).ToNot(BeEmpty())
		generatedPassword := cli.cliCommandArgs[1][2]

		Expect(cli.cliCommandArgs[2]).To(Equal(
			[]string{
				"set-space-role",
				fmt.Sprintf("space-drain-%s", guid),
				"ORG", "SPACE",
				"SpaceDeveloper",
			},
		))

		Expect(cli.cliCommandWithoutTerminalOutputArgs).To(ConsistOf(
			[]string{"set-env", "space-drain", "SPACE_ID", "space-guid"},
			[]string{"set-env", "space-drain", "DRAIN_NAME", "some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_URL", "https://some-drain"},
			[]string{"set-env", "space-drain", "DRAIN_TYPE", "all"},
			[]string{"set-env", "space-drain", "API_ADDR", "https://api.something.com"},
			[]string{"set-env", "space-drain", "UAA_ADDR", "https://uaa.something.com"},
			[]string{"set-env", "space-drain", "CLIENT_ID", "cf"},
			[]string{"set-env", "space-drain", "USERNAME", fmt.Sprintf("space-drain-%s", guid)},
			[]string{"set-env", "space-drain", "PASSWORD", generatedPassword},
			[]string{"set-env", "space-drain", "SKIP_CERT_VERIFY", "false"},
			[]string{"set-env", "space-drain", "DRAIN_SCOPE", "space"},
		))

		Expect(cli.cliCommandArgs[3]).To(Equal(
			[]string{
				"start", "space-drain",
			},
		))
	})

	It("fatally logs if user-provided password is blank", func() {
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				func(int) ([]byte, error) { return []byte{}, nil },
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Password cannot be blank."))
	})

	It("fatally logs if reading password input fails", func() {
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				func(int) ([]byte, error) { return []byte("don't use this"), errors.New("some-error") },
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	DescribeTable("fatally logs if setting env variables fails", func(env string) {
		cli.setEnvErrors[env] = errors.New("some-error")

		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
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
		Entry("USERNAME", "USERNAME"),
		Entry("PASSWORD", "PASSWORD"),
		Entry("SKIP_CERT_VERIFY", "SKIP_CERT_VERIFY"),
		Entry("DRAIN_SCOPE", "DRAIN_SCOPE"),
	)

	It("fatally logs if confirmation is given anything other than y", func() {
		reader = strings.NewReader("no\n")

		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
				logger,
			)
		}).To(Panic())

		Expect(logger.fatalfMessage).To(Equal("OK, exiting."))
	})

	It("fatally logs if creating user fails", func() {
		cli.createUserError = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	It("fatally logs if fetching the app fails", func() {
		cli.getAppError = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	It("fatally logs if fetching the space fails", func() {
		cli.currentSpaceError = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("some-error"))
	})

	It("fatally logs if fetching the org fails", func() {
		cli.currentOrgError = errors.New("some-error")
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
				},
				downloader,
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
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
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
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("failed to push"))
	})

	It("fatally logs if the space-drain drain-name is not provided", func() {
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("the required flag `--drain-name' was not specified"))
	})

	It("fatally logs if the space-drain drain-url is not provided", func() {
		Expect(func() {
			command.PushSpaceDrain(
				cli,
				reader,
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--username", "some-user",
				},
				downloader,
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
				pr,
				[]string{
					"--path", "some-temp-dir",
					"--drain-name", "some-drain",
					"--drain-url", "https://some-drain",
					"--username", "some-user",
					"some-unknown-arg",
				},
				downloader,
				logger,
			)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 0, got 1."))
	})
})
